package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentiallibraries"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	vaultCredStoreResc          = "boundary_credential_store_vault.example"
	vaultCredStoreName          = "foo"
	vaultCredStoreDesc          = "the foo"
	vaultCredStoreAddr          = "127.0.0.1:9100"
	vaultCredStoreNamespace     = "default"
	vaultCredStoreCaCert        = ""
	vaultCredStoreTlsServerName = "example.com"
	vaultCredStoreTlsSkipVerify = true
	vaultCredStoreToken         = ""
	vaultCredStoreClientCert    = ""
	vaultCredStoreClientKey     = ""
	vaultCredStoreUpdate        = "updated"
)

var vaultCredStoreResource = fmt.Sprintf(`
resource "boundary_credential_store_vault" "example" {
  name  = "%s"
	description = "%s"
	address = "%s"
	namespace = "%s"
	vault_ca_cert = "%s"
	tls_server_name = "%s"
	tls_skip_verify = "%s"
	vault_token = "%s"
	client_certificate = "%s"
	client_certificate_key = "%s"
}`, vaultCredStoreName,
	vaultCredStoreDesc,
	vaultCredStoreAddr,
	vaultCredStoreNamespace,
	vaultCredStoreCaCert,
	vaultCredStoreTlsServerName,
	vaultCredStoreTlsSkipVerify,
	vaultCredStoreToken,
	vaultCredStoreClientCert,
	vaultCredStoreClientKey)

var vaultCredStoreResourceUpdate = fmt.Sprintf(`
resource "boundary_credential_store_vault" "example" {
  name  = "%s"
	description = "%s"
	address = "%s"
	namespace = "%s"
	vault_ca_cert = "%s"
	tls_server_name = "%s"
	tls_skip_verify = "%s"
	vault_token = "%s"
	client_certificate = "%s"
	client_certificate_key = "%s"
}`, vaultCredStoreName+vaultCredStoreUpdate,
	vaultCredStoreDesc+vaultCredStoreUpdate,
	vaultCredStoreAddr,
	vaultCredStoreNamespace,
	vaultCredStoreCaCert,
	vaultCredStoreTlsServerName,
	vaultCredStoreTlsSkipVerify,
	vaultCredStoreToken,
	vaultCredStoreClientCert,
	vaultCredStoreClientKey)

func TestAccCredentialStoreVault(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAuthMethodResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, vaultCredStoreResource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredResc, "name", vaultCredStoreName),
					resource.TestCheckResourceAttr(vaultCredResc, "description", vaultCredStoreDesc),
					resource.TestCheckResourceAttr(vaultCredResc, "vault_path", vaultCredStorePath),
					resource.TestCheckResourceAttr(vaultCredResc, "vault_http_method", vaultCredStoreMethod),
					resource.TestCheckResourceAttr(vaultCredResc, "vault_http_request_body", vaultCredStoreRequestBody),

					testAccCheckCredentialStoreVaultResourceExists(provider, vaultCredResc),
				),
			},
			importStep(vaultCredResc),

			{
				// update
				Config: testConfig(url, fooOrg, vaultCredStoreResource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredResc, "name", vaultCredStoreName+vaultCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultCredResc, "description", vaultCredStoreDesc+vaultCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultCredResc, "vault_path", vaultCredStorePath),
					resource.TestCheckResourceAttr(vaultCredResc, "vault_http_method", vaultCredStoreMethod),
					resource.TestCheckResourceAttr(vaultCredResc, "vault_http_request_body", vaultCredStoreRequestBody),

					testAccCheckCredentialStoreVaultResourceExists(provider, vaultCredResc),
				),
			},
			importStep(vaultCredResc),
		},
	})
}

func testAccCheckCredentialStoreVaultResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("No ID is set")
		}

		md := testProvider.Meta().(*metaData)

		c := credentiallibraries.NewClient(md.client)

		if _, err := c.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error reading %q: %w", id, err)
		}

		return nil
	}
}

func testAccCheckCredentialStoreVaultResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_credential_library_vault":
				id := rs.Primary.ID

				c := credentiallibraries.NewClient(md.client)

				_, err := c.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed vault credential library %q: %v", id, err)
				}

			default:
				continue
			}
		}
		return nil
	}
}
