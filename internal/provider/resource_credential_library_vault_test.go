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
	vaultCredResc            = "boundary_credential_library_vault.example"
	vaultCredLibName         = "foo"
	vaultCredLibDesc         = "the foo"
	vaultCredLibStoreId      = ""
	vaultCredLibPath         = "/foo/bar"
	vaultCredLibMethod       = "POST"
	vaultCredLibRequestBody  = ""
	vaultCredLibStringUpdate = "_random"
)

var vaultCredLibResource = fmt.Sprintf(`
resource "boundary_credential_library_vault" "example" {
  name  = "%s"
	description = "%s"
	credential_store_id = "%s"
  vault_path = "%s"
  vault_http_method = "%s"
  vault_http_request_body = "%s"
}`, vaultCredLibName,
	vaultCredLibDesc,
	vaultCredLibStoreId,
	vaultCredLibPath,
	vaultCredLibMethod,
	vaultCredLibRequestBody)

var vaultCredLibResourceUpdate = fmt.Sprintf(`
resource "boundary_credential_library_vault" "example" {
  name  = "%s"
	description = "%s"
	credential_store_id = "%s"
  vault_path = "%s"
  vault_http_method = "%s"
  vault_http_request_body = "%s"
}`, vaultCredLibName+vaultCredLibStringUpdate,
	vaultCredLibDesc+vaultCredLibStringUpdate,
	vaultCredLibStoreId,
	vaultCredLibPath,
	vaultCredLibMethod,
	vaultCredLibRequestBody)

func TestAccCredentialLibraryVault(t *testing.T) {
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
				Config: testConfig(url, fooOrg, vaultCredLibResource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredResc, "name", vaultCredLibName),
					resource.TestCheckResourceAttr(vaultCredResc, "description", vaultCredLibDesc),
					resource.TestCheckResourceAttr(vaultCredResc, "vault_path", vaultCredLibPath),
					resource.TestCheckResourceAttr(vaultCredResc, "vault_http_method", vaultCredLibMethod),
					resource.TestCheckResourceAttr(vaultCredResc, "vault_http_request_body", vaultCredLibRequestBody),

					testAccCheckCredentialLibraryVaultResourceExists(provider, vaultCredResc),
				),
			},
			importStep(vaultCredResc),

			{
				// update
				Config: testConfig(url, fooOrg, vaultCredLibResource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredResc, "name", vaultCredLibName+vaultCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultCredResc, "description", vaultCredLibDesc+vaultCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultCredResc, "vault_path", vaultCredLibPath),
					resource.TestCheckResourceAttr(vaultCredResc, "vault_http_method", vaultCredLibMethod),
					resource.TestCheckResourceAttr(vaultCredResc, "vault_http_request_body", vaultCredLibRequestBody),

					testAccCheckCredentialLibraryVaultResourceExists(provider, vaultCredResc),
				),
			},
			importStep(vaultCredResc),
		},
	})
}

func testAccCheckCredentialLibraryVaultResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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

func testAccCheckCredentialLibraryVaultResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
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
