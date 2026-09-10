// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/cap/ldap"
	"github.com/hashicorp/cap/oidc"
	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
	"github.com/hashicorp/go-kms-wrapping/v2/aead"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/jimlambrt/gldap"
	"github.com/jimlambrt/gldap/testdirectory"
	"github.com/stretchr/testify/require"
)

var (
	tcLoginName = "admin"
	tcPassword  = "password"
	tcPAUM      = "ampw_1234567890"
	tcConfig    = []controller.Option{
		controller.WithDefaultPasswordAuthMethodId(tcPAUM),
		controller.WithDefaultLoginName(tcLoginName),
		controller.WithDefaultPassword(tcPassword),
	}
)

func providerFactories(p **schema.Provider) map[string]func() (*schema.Provider, error) {
	// TODO: eventually rework this to real factories...
	*p = New()
	return map[string]func() (*schema.Provider, error){
		"boundary": func() (*schema.Provider, error) {
			return *p, nil
		},
	}
}

func testWrapper(ctx context.Context, t *testing.T, key string) wrapping.Wrapper {
	var keyBytes []byte
	switch key {
	case "":
		keyBytes = make([]byte, 32)
		n, err := rand.Read(keyBytes)
		if err != nil {
			t.Fatal(err)
		}
		if n != 32 {
			t.Fatal(n)
		}
		key = base64.StdEncoding.EncodeToString(keyBytes)
	default:
		var err error
		keyBytes, err = base64.StdEncoding.DecodeString(key)
		if err != nil {
			t.Fatal(err)
		}
	}
	wrapper := aead.NewWrapper()

	_, err := wrapper.SetConfig(ctx, wrapping.WithKeyId(key))
	if err != nil {
		t.Fatal(err)
	}
	if err := wrapper.SetAesGcmKeyBytes(keyBytes); err != nil {
		t.Fatal(err)
	}
	return wrapper
}

func testConfig(url string, res ...string) string {
	provider := fmt.Sprintf(`
provider "boundary" {
	addr             = "%s"
	auth_method_id       = "%s"
	password_auth_method_login_name = "%s"
	password_auth_method_password = "%s"
}`, url, tcPAUM, tcLoginName, tcPassword)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func testConfigWithToken(url, token string, res ...string) string {
	provider := fmt.Sprintf(`
provider "boundary" {
	addr  = "%s"
	token = "%s"
}`, url, token)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func testConfigWithDefaultAuthMethod(url string, res ...string) string {
	provider := fmt.Sprintf(`
provider "boundary" {
	addr  = "%s"
	auth_method_login_name = "%s"
	auth_method_password = "%s"
}`, url, tcLoginName, tcPassword)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func testConfigWithDeprecatedAuthMethod(url string, res ...string) string {
	provider := fmt.Sprintf(`
provider "boundary" {
	addr  = "%s"
	password_auth_method_login_name = "%s"
	password_auth_method_password = "%s"
}`, url, tcLoginName, tcPassword)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func testConfigWithOIDCAuthMethod(url string, res ...string) string {
	provider := fmt.Sprintf(`
provider "boundary" {
	addr  = "%s"
	auth_method_id = "amoidc_0000000000"
}`, url)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func testConfigWithoutAMPWCredentials(url string, res ...string) string {
	provider := fmt.Sprintf(`
provider "boundary" {
	addr           = "%s"
	auth_method_id = "%s"
}`, url, tcPAUM)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func testConfigWithRecovery(t *testing.T, url string, res ...string) string {
	t.Helper()
	cfg, err := loadTestConfig()
	if err != nil {
		t.Fatal(err)
	}

	key := requireRecoveryKey(t, cfg)
	provider := fmt.Sprintf(`
provider "boundary" {
	addr             = "%s"
	recovery_kms_hcl = <<DOC
	kms "aead" {
		purpose = ["recovery", "config"]
		aead_type = "aes-gcm"
		key = "%s"
		key_id = "global_recovery"
	}
	DOC
}`, url, key)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func testConfigWithLDAPAuthMethod(url string, scopeId string, loginName string, password string, res ...string) string {
	provider := fmt.Sprintf(`
provider "boundary" {
	addr                   = "%s"
	scope_id               = "%s"
	auth_method_login_name = "%s"
	auth_method_password   = "%s"
}`, url, scopeId, loginName, password)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func importStep(name string, ignore ...string) resource.TestStep {
	step := resource.TestStep{
		ResourceName:      name,
		ImportState:       true,
		ImportStateVerify: true,
	}

	if len(ignore) > 0 {
		step.ImportStateVerifyIgnore = ignore
	}

	return step
}

func TestProvider(t *testing.T) {
	if err := New().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestConfigWithLdapAuthMethod(t *testing.T) {
	td := createDefaultLdap(t)
	defer td.Stop()
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	suffix := id.UniqueId()
	ldapLoginName := "alice"
	ldapPassword := "password"

	createLdapAMConfig := fmt.Sprintf(testPrimaryAuthMethodLdap, td.Host(), td.Port(), testdirectory.DefaultUserDN, testdirectory.DefaultGroupDN)
	createLdapAccountConfig := fmt.Sprintf(testProviderLdapAccountConfig, ldapLoginName)

	// provider is set by providerFactories when resource.Test initialises.
	// steps is declared as a var (slice = reference type) so step 1's Check can
	// write the org scope ID into steps[1].Config before the framework reads it.
	var provider *schema.Provider
	var steps []resource.TestStep
	steps = []resource.TestStep{
		{
			// Step 1: create LDAP auth method as primary for the org.
			// Capture the org scope ID and write it into step 2's provider block.
			Config: testConfig(url, fooOrg(suffix), createLdapAMConfig, createLdapAccountConfig),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapInsecureTlsField, "true"),
				func(s *terraform.State) error {
					rs, ok := s.RootModule().Resources["boundary_scope.org1"]
					if !ok {
						return fmt.Errorf("boundary_scope.org1 not found in state")
					}
					steps[1].Config = testConfigWithLDAPAuthMethod(url, rs.Primary.ID, ldapLoginName, ldapPassword, fooOrg(suffix), createLdapAMConfig, createLdapAccountConfig)
					return nil
				},
			),
		},
		{
			// Step 2: provider uses scope_id=org1 to auto-discover the LDAP primary AM.
			// Config is written by step 1's Check above; placeholder avoids empty-step error.
			Config: testConfig(url, fooOrg(suffix), createLdapAMConfig, createLdapAccountConfig),
			Check: resource.ComposeTestCheckFunc(
				func(s *terraform.State) error { return testProviderTokenExists(provider)(s) },
			),
		},
		{
			// Step 3: switch back to password auth and verify it still works after
			// LDAP was primary for the org.
			Config: testConfig(url, fooOrg(suffix)),
			Check: resource.ComposeTestCheckFunc(
				func(s *terraform.State) error { return testProviderTokenExists(provider)(s) },
			),
		},
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAuthMethodResourceDestroy(t, provider, ldapAuthMethodType),
		Steps:             steps,
	})
}

func TestConfigWithDefaultAuthMethod(t *testing.T) {
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	suffix := id.UniqueId()

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckScopeResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				Config: testConfigWithDefaultAuthMethod(url, fooOrg(suffix), firstProjectFoo, secondProject),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists(provider, "boundary_scope.org1"),
					testProviderTokenExists(provider),
				),
			},
		},
	})
}

func TestConfigWithDeprecatedAuthMethod(t *testing.T) {
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	suffix := id.UniqueId()

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckScopeResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				Config: testConfigWithDeprecatedAuthMethod(url, fooOrg(suffix), firstProjectFoo, secondProject),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists(provider, "boundary_scope.org1"),
					testProviderTokenExists(provider),
				),
			},
		},
	})
}

func TestConfigWithoutAMPWCredentials(t *testing.T) {
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	suffix := id.UniqueId()

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfigWithoutAMPWCredentials(url, fooOrg(suffix), firstProjectFoo, secondProject),
				ExpectError: regexp.MustCompile("auth method login name not set, please set auth_method_login_name on the provider"),
			},
		},
	})
}

func TestConfigWithOIDCAuthMethod(t *testing.T) {
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	suffix := id.UniqueId()

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfigWithOIDCAuthMethod(url, fooOrg(suffix), firstProjectFoo, secondProject),
				ExpectError: regexp.MustCompile("OIDC auth method is currently not supported by Boundary Terraform Provider. only password auth method is supported at this time"),
			},
		},
	})
}

// Create OIDC auth method and set it as the primary auth method.
// Attempt to authenticate with recovery to test checks for default auth method
func TestRecoveryWithOIDCDefaultAuthMethod(t *testing.T) {
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	suffix := id.UniqueId()

	tp := oidc.StartTestProvider(t)
	tpCert := strings.TrimSpace(tp.CACert())
	createConfig := fmt.Sprintf(fooAuthMethodOidc, fooAuthMethodOidcDesc, tp.Addr(), tpCert)
	updateConfig := fmt.Sprintf(fooAuthMethodOidcUpdate, fooAuthMethodOidcDescUpdate, fooAuthMethodOidcCaCerts)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAuthMethodResourceDestroy(t, provider, oidcAuthMethodType),
		Steps: []resource.TestStep{
			{
				// create auth method
				Config: testConfig(url, fooOrg(suffix), createConfig),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_oidc.foo", "description", fooAuthMethodOidcDesc),
					resource.TestCheckResourceAttr("boundary_auth_method_oidc.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method_oidc.foo", authmethodOidcIssuerKey, tp.Addr()),
				),
			},
			importStep("boundary_auth_method_oidc.foo", "client_secret", "is_primary_for_scope"),
			{
				// set auth method as primary auth method
				Config: testConfig(url, fooOrg(suffix), updateConfig),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_oidc.foo", "name", "test"),
					testAccIsPrimaryForScope(provider, "boundary_auth_method_oidc.foo", true),
				),
			},
			{
				// authenticate provider with recovery kms with unsupported OIDC primary auth method
				Config: testConfigWithRecovery(t, url, fooOrg(suffix), updateConfig),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_oidc.foo", "name", "test"),
					testAccIsPrimaryForScope(provider, "boundary_auth_method_oidc.foo", true),
				),
			},
			importStep("boundary_auth_method_oidc.foo", "client_secret", "is_primary_for_scope", authmethodOidcMaxAgeKey),
		},
	})
}

func testProviderTokenExists(testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		md := testProvider.Meta().(*metaData)
		if md.client.Token() == "" {
			return fmt.Errorf("token not set")
		}
		return nil
	}
}

func createDefaultLdap(t *testing.T) *testdirectory.Directory {
	td := testdirectory.Start(
		t,
		testdirectory.WithDefaults(t, &testdirectory.Defaults{AllowAnonymousBind: true}),
		testdirectory.WithNoTLS(t),
	)

	groups := []*gldap.Entry{
		testdirectory.NewGroup(t, "admin", []string{"alice"}),
	}

	users := testdirectory.NewUsers(t, []string{"alice"}, testdirectory.WithMembersOf(t, "admin"))

	for _, u := range users {
		u.Attributes = append(
			u.Attributes,
			gldap.NewEntryAttribute(ldap.DefaultADUserPasswordAttribute, []string{"password"}),
			gldap.NewEntryAttribute(ldap.DefaultOpenLDAPUserPasswordAttribute, []string{"password"}),
			gldap.NewEntryAttribute("fullName", []string{"test-full-name"}),
		)
	}

	td.SetUsers(users...)
	td.SetGroups(groups...)

	return td
}
