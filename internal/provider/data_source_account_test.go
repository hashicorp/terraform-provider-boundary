// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/YakDriver/regexache"
	"github.com/hashicorp/cap/oidc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

var accountPasswordRead = `
data "boundary_account" "acc_password" {
	depends_on = [ boundary_account_password.foo ]
	name = "test"
	auth_method_id = boundary_auth_method.foo.id
}`

var accountLdapRead = fmt.Sprintf(`
data "boundary_account" "acc_ldap" {
	depends_on = [ boundary_account_ldap.foo ]
	name = "%s"
	auth_method_id = boundary_auth_method_ldap.foo.id
}`, testAccountLdapName)

var accountOidcRead = `
data "boundary_account" "acc_oidc" {
	depends_on = [ boundary_account_oidc.foo ]
	name = "test"
	auth_method_id = boundary_auth_method_oidc.foo.id
}`

func TestAccAccountReadPassword(t *testing.T) {
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	suffix := id.UniqueId()

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg(suffix), fooAccountPassword, accountPasswordRead),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccountResourceExists(provider, "boundary_account_password.foo"),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_password", IDKey),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_password", AuthMethodIdKey),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_password", NameKey),
					resource.TestCheckResourceAttr("data.boundary_account.acc_password", DescriptionKey, fooAccountPasswordDesc),
					resource.TestCheckResourceAttr("data.boundary_account.acc_password", TypeKey, "password"),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_password", "scope.0.id"),
					resource.TestMatchResourceAttr("data.boundary_account.acc_password", "scope.0.name", regexache.MustCompile(`^org1-`)),
					resource.TestCheckResourceAttr("data.boundary_account.acc_password", "scope.0.type", "org"),
				),
			},
		},
	})
}

func TestAccAccountReadLdap(t *testing.T) {
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	suffix := id.UniqueId()

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg(suffix), testAccountLdap, accountLdapRead),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccountResourceExists(provider, "boundary_account_ldap.foo"),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_ldap", IDKey),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_ldap", AuthMethodIdKey),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_ldap", NameKey),
					resource.TestCheckResourceAttr("data.boundary_account.acc_ldap", DescriptionKey, testAccountLdapDesc),
					resource.TestCheckResourceAttr("data.boundary_account.acc_ldap", TypeKey, "ldap"),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_ldap", "scope.0.id"),
					resource.TestMatchResourceAttr("data.boundary_account.acc_ldap", "scope.0.name", regexache.MustCompile(`^org1-`)),
					resource.TestCheckResourceAttr("data.boundary_account.acc_ldap", "scope.0.type", "org"),
				),
			},
		},
	})
}

func TestAccAccountReadOidc(t *testing.T) {
	tp := oidc.StartTestProvider(t)
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	suffix := id.UniqueId()

	var provider *schema.Provider
	tpCert := strings.TrimSpace(tp.CACert())
	createConfig := fmt.Sprintf(fooAccountOidc, tp.Addr(), tpCert, fooAccountOidcDesc, tp.ExpectedSubject(), tp.Addr())

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg(suffix), createConfig, accountOidcRead),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccountResourceExists(provider, "boundary_account_oidc.foo"),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_oidc", IDKey),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_oidc", AuthMethodIdKey),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_oidc", NameKey),
					resource.TestCheckResourceAttr("data.boundary_account.acc_oidc", DescriptionKey, fooAccountOidcDesc),
					resource.TestCheckResourceAttr("data.boundary_account.acc_oidc", TypeKey, "oidc"),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_oidc", "scope.0.id"),
					resource.TestMatchResourceAttr("data.boundary_account.acc_oidc", "scope.0.name", regexache.MustCompile(`^org1-`)),
					resource.TestCheckResourceAttr("data.boundary_account.acc_oidc", "scope.0.type", "org"),
				),
			},
		},
	})
}
