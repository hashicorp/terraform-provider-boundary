package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentiallibraries"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/boundary/testing/vault"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	vaultCredResc            = "boundary_credential_library_vault.example"
	vaultCredTypedResc       = "boundary_credential_library_vault.typed_example"
	vaultCredLibName         = "foo"
	vaultCredLibDesc         = "the foo"
	vaultCredLibPath         = "/foo/bar"
	vaultCredLibMethodGet    = "GET"
	vaultCredLibMethodPost   = "POST"
	vaultCredLibRequestBody  = "foobar"
	vaultCredLibStringUpdate = "_random"
)

var vaultCredLibResource = fmt.Sprintf(`
resource "boundary_credential_library_vault" "example" {
	name  = "%s"
	description = "%s"
	credential_store_id = boundary_credential_store_vault.example.id
  	path = "%s"
  	http_method = "%s"
}`, vaultCredLibName,
	vaultCredLibDesc,
	vaultCredLibPath,
	vaultCredLibMethodGet)

var vaultCredLibResourceUpdate = fmt.Sprintf(`
resource "boundary_credential_library_vault" "example" {
  	name  = "%s"
	description = "%s"
  	credential_store_id = boundary_credential_store_vault.example.id
  	path = "%s"
  	http_method = "%s"
  	http_request_body = "%s"
}`, vaultCredLibName+vaultCredLibStringUpdate,
	vaultCredLibDesc+vaultCredLibStringUpdate,
	vaultCredLibPath+vaultCredLibStringUpdate,
	vaultCredLibMethodPost,
	vaultCredLibRequestBody)

var vaultTypedCredLibResource = fmt.Sprintf(`
resource "boundary_credential_library_vault" "typed_example" {
	name                = "%s"
	description         = "%s"
	credential_store_id = boundary_credential_store_vault.example.id
  	path                = "%s"
  	http_method         = "%s"
	credential_type     = "ssh_private_key"
}`, vaultCredLibName,
	vaultCredLibDesc,
	vaultCredLibPath,
	vaultCredLibMethodGet)

func TestAccCredentialLibraryVault(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	vc := vault.NewTestVaultServer(t)
	_, token := vc.CreateToken(t)
	credStoreRes := vaultCredStoreResource(vc,
		vaultCredStoreName,
		vaultCredStoreDesc,
		vaultCredStoreNamespace,
		"www.original.com",
		token,
		true)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialLibraryVaultResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultCredLibResource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredResc, NameKey, vaultCredLibName),
					resource.TestCheckResourceAttr(vaultCredResc, DescriptionKey, vaultCredLibDesc),
					resource.TestCheckResourceAttr(vaultCredResc, credentialLibraryVaultPathKey, vaultCredLibPath),
					resource.TestCheckResourceAttr(vaultCredResc, credentialLibraryVaultHttpMethodKey, vaultCredLibMethodGet),
					resource.TestCheckResourceAttr(vaultCredResc, credentialLibraryVaultHttpRequestBodyKey, ""),

					testAccCheckCredentialLibraryVaultResourceExists(provider, vaultCredResc),
				),
			},
			importStep(vaultCredResc),

			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultCredLibResourceUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredResc, NameKey, vaultCredLibName+vaultCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultCredResc, DescriptionKey, vaultCredLibDesc+vaultCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultCredResc, credentialLibraryVaultPathKey, vaultCredLibPath+vaultCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultCredResc, credentialLibraryVaultHttpMethodKey, vaultCredLibMethodPost),
					resource.TestCheckResourceAttr(vaultCredResc, credentialLibraryVaultHttpRequestBodyKey, vaultCredLibRequestBody),

					testAccCheckCredentialLibraryVaultResourceExists(provider, vaultCredResc),
				),
			},
			importStep(vaultCredResc),

			{
				// create typed credential library, note credential type is immutable so no need for update test
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultCredLibResourceUpdate, vaultTypedCredLibResource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredTypedResc, NameKey, vaultCredLibName),
					resource.TestCheckResourceAttr(vaultCredTypedResc, DescriptionKey, vaultCredLibDesc),
					resource.TestCheckResourceAttr(vaultCredTypedResc, credentialLibraryVaultPathKey, vaultCredLibPath),
					resource.TestCheckResourceAttr(vaultCredTypedResc, credentialLibraryVaultHttpMethodKey, vaultCredLibMethodGet),
					resource.TestCheckResourceAttr(vaultCredTypedResc, credentialLibraryVaultHttpRequestBodyKey, ""),
					resource.TestCheckResourceAttr(vaultCredTypedResc, credentialLibraryCredentialTypeKey, "ssh_private_key"),

					testAccCheckCredentialLibraryVaultResourceExists(provider, vaultCredTypedResc),
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
			return fmt.Errorf("not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("no ID is set")
		}

		md := testProvider.Meta().(*metaData)
		c := credentiallibraries.NewClient(md.client)
		if _, err := c.Read(context.Background(), id); err != nil {
			return fmt.Errorf("got an error reading %q: %w", id, err)
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
