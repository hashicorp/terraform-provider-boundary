// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentialstores"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/boundary/testing/vault"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"golang.org/x/crypto/blake2b"
)

const (
	vaultCredStoreResc      = "boundary_credential_store_vault.example"
	vaultCredStoreName      = "foo"
	vaultCredStoreDesc      = "the foo"
	vaultCredStoreNamespace = "default"
	vaultCredStoreUpdate    = "_random"
)

func vaultCredStoreResource(vc *vault.TestVaultServer, name, description, namespace, tlsServer, token string, skipVerify bool) string {
	caCert := fmt.Sprintf("\"%s\"", strings.Replace(string(vc.CaCert), "\n", `\n`, -1))
	clientCert := fmt.Sprintf("\"%s\"", strings.Replace(string(vc.ClientCert), "\n", `\n`, -1))
	clientKey := fmt.Sprintf("\"%s\"", strings.Replace(string(vc.ClientKey), "\n", `\n`, -1))

	return fmt.Sprintf(`
resource "boundary_credential_store_vault" "example" {
	name  = "%s"
	description = "%s"
	scope_id = boundary_scope.proj1.id
	address = "%s"
	namespace = "%s"
	ca_cert = %s
	tls_server_name = "%s"
	tls_skip_verify = "%v"
	token = "%s"
	client_certificate = %s
	client_certificate_key = %s
	depends_on  = [boundary_role.proj1_admin]
}`, name,
		description,
		vc.Addr,
		namespace,
		caCert,
		tlsServer,
		skipVerify,
		token,
		clientCert,
		clientKey)
}

func tokenHmac(token, accessor string) string {
	key := blake2b.Sum256([]byte(accessor))
	mac := hmac.New(sha256.New, key[:])
	_, _ = mac.Write([]byte(token))
	hmac := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(hmac)
}

func TestAccCredentialStoreVault(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	vc := vault.NewTestVaultServer(t)
	secret, token := vc.CreateToken(t)
	tHmac := tokenHmac(token, secret.Auth.Accessor)
	res := vaultCredStoreResource(vc,
		vaultCredStoreName,
		vaultCredStoreDesc,
		vaultCredStoreNamespace,
		"www.original.com",
		token,
		false)

	vcUpdate := vault.NewTestVaultServer(t, vault.WithTestVaultTLS(vault.TestClientTLS))
	secret, tokenUpdate := vcUpdate.CreateToken(t)
	tHmacUpdate := tokenHmac(tokenUpdate, secret.Auth.Accessor)
	resUpdate := vaultCredStoreResource(vcUpdate,
		vaultCredStoreName+vaultCredStoreUpdate,
		vaultCredStoreDesc+vaultCredStoreUpdate,
		vaultCredStoreNamespace+vaultCredStoreUpdate,
		"www.updated.com",
		tokenUpdate,
		false)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialStoreResourceDestroy(t, provider, vaultStoreCredentialStoreType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredStoreResc, NameKey, vaultCredStoreName),
					resource.TestCheckResourceAttr(vaultCredStoreResc, DescriptionKey, vaultCredStoreDesc),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultAddressKey, vc.Addr),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultNamespaceKey, vaultCredStoreNamespace),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultCaCertKey, ""),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultTlsServerNameKey, "www.original.com"),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultTlsSkipVerifyKey, "false"),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultTokenKey, token),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultTokenHmacKey, tHmac),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultClientCertificateKey, ""),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultClientCertificateKeyKey, ""),

					testAccCheckCredentialStoreResourceExists(provider, vaultCredStoreResc),
				),
			},
			importStep(vaultCredStoreResc, credentialStoreVaultTokenKey, credentialStoreVaultClientCertificateKeyKey),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, resUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredStoreResc, NameKey, vaultCredStoreName+vaultCredStoreUpdate),
					resource.TestCheckResourceAttr(vaultCredStoreResc, DescriptionKey, vaultCredStoreDesc+vaultCredStoreUpdate),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultAddressKey, vcUpdate.Addr),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultNamespaceKey, vaultCredStoreNamespace+vaultCredStoreUpdate),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultCaCertKey, string(vcUpdate.CaCert)),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultTlsServerNameKey, "www.updated.com"),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultTlsSkipVerifyKey, "false"),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultTokenKey, tokenUpdate),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultTokenHmacKey, tHmacUpdate),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultClientCertificateKey, string(vcUpdate.ClientCert)),
					resource.TestCheckResourceAttr(vaultCredStoreResc, credentialStoreVaultClientCertificateKeyKey, string(vcUpdate.ClientKey)),

					testAccCheckCredentialStoreResourceExists(provider, vaultCredStoreResc),
				),
			},
			importStep(vaultCredStoreResc, credentialStoreVaultTokenKey, credentialStoreVaultClientCertificateKeyKey),
			{
				// Run a plan only update and verify no changes
				PlanOnly: true,
				Config:   testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(vaultCredStoreResc, credentialStoreVaultTokenKey, credentialStoreVaultClientCertificateKeyKey),
			{
				// update again but apply a preConfig to externally update resource
				// TODO: Boundary currently causes an error on moving back to a previously
				// used token, for now verify that a plan only step had changes
				PreConfig:          func() { externalUpdate(t, provider) },
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Config:             testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(vaultCredStoreResc, credentialStoreVaultTokenKey, credentialStoreVaultClientCertificateKeyKey),
		},
	})
}

var storeId string

// externalUpdate uses the global storeId, therefore this function cannot be called until
// a previous test step that calls testAccCheckCredentialStoreVaultResourceExists has completed.
func externalUpdate(t *testing.T, testProvider *schema.Provider) {
	if storeId == "" {
		t.Fatal("storeId must be set before testing an external update")
	}
	vs := vault.NewTestVaultServer(t, vault.WithTestVaultTLS(vault.TestClientTLS))
	_, token := vs.CreateToken(t)

	md := testProvider.Meta().(*metaData)
	c := credentialstores.NewClient(md.client)
	cr, err := c.Read(context.Background(), storeId)
	if err != nil {
		t.Fatal(fmt.Errorf("got an error reading %q: %w", storeId, err))
	}

	// update Vault server to existing store
	var opts []credentialstores.Option
	opts = append(opts, credentialstores.WithVaultCredentialStoreToken(token))
	opts = append(opts, credentialstores.WithVaultCredentialStoreAddress(vs.Addr))
	opts = append(opts, credentialstores.WithVaultCredentialStoreCaCert(string(vs.CaCert)))
	opts = append(opts, credentialstores.WithVaultCredentialStoreClientCertificate(string(vs.ClientCert)))
	opts = append(opts, credentialstores.WithVaultCredentialStoreClientCertificateKey(string(vs.ClientKey)))

	_, err = c.Update(context.Background(), cr.Item.Id, cr.Item.Version, opts...)
	if err != nil {
		t.Fatal(fmt.Errorf("got an error updating %q: %w", cr.Item.Id, err))
	}
}

func testAccCheckCredentialStoreResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("no ID is set")
		}
		storeId = id

		md := testProvider.Meta().(*metaData)
		c := credentialstores.NewClient(md.client)
		if _, err := c.Read(context.Background(), id); err != nil {
			return fmt.Errorf("got an error reading %q: %w", id, err)
		}

		return nil
	}
}

type credentialDestroyStoreType string

const (
	staticStoreCredentialStoreType credentialDestroyStoreType = "boundary_credential_store_static"
	vaultStoreCredentialStoreType  credentialDestroyStoreType = "boundary_credential_store_vault"
)

func testAccCheckCredentialStoreResourceDestroy(t *testing.T, testProvider *schema.Provider, typ credentialDestroyStoreType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case string(typ):
				id := rs.Primary.ID

				c := credentialstores.NewClient(md.client)
				_, err := c.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed static credential store %q: %v", id, err)
				}
			default:
				continue
			}
		}
		return nil
	}
}
