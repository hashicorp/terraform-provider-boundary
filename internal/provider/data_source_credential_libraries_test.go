package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/boundary/testing/vault"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	fooCredentialLibrariesDataMissingCredentialStoreId = `
data "boundary_credential_libraries" "foo" {}
`
	fooCredentialLibrariesData = `
data "boundary_credential_libraries" "foo" {
	credential_store_id = boundary_credential_library_vault.example.credential_store_id
}
`
)

func TestAccDataSourceCredentialLibraries(t *testing.T) {
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
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooCredentialLibrariesDataMissingCredentialStoreId),
				ExpectError: regexp.MustCompile("credential_store_id: This field must be a valid credential store id."),
			},
			{
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultCredLibResource, fooCredentialLibrariesData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.boundary_credential_libraries.foo", "credential_store_id"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.%", "10"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.authorized_actions.#", "4"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.authorized_actions.0", "no-op"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.authorized_actions.1", "read"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.authorized_actions.2", "update"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.authorized_actions.3", "delete"),
					resource.TestCheckResourceAttrSet("data.boundary_credential_libraries.foo", "items.0.created_time"),
					resource.TestCheckResourceAttrSet("data.boundary_credential_libraries.foo", "items.0.credential_store_id"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.description", "the foo"),
					resource.TestCheckResourceAttrSet("data.boundary_credential_libraries.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.name", "foo"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.scope.0.description", "foo"),
					resource.TestCheckResourceAttrSet("data.boundary_credential_libraries.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.scope.0.name", "proj1"),
					resource.TestCheckResourceAttrSet("data.boundary_credential_libraries.foo", "items.0.scope.0.parent_scope_id"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.scope.0.type", "project"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.type", "vault"),
					resource.TestCheckResourceAttrSet("data.boundary_credential_libraries.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_credential_libraries.foo", "items.0.version", "1"),
				),
			},
		},
	})
}
