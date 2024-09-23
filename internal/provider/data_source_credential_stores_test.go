// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0
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
	fooCredentialStoresDataMissingScopeId = `
data "boundary_credential_stores" "foo" {}
`
	fooCredentialStoresData = `
data "boundary_credential_stores" "foo" {
	depends_on = [boundary_credential_store_vault.example]
	scope_id = boundary_credential_store_vault.example.scope_id
}
`
)

func TestAccDataSourceCredentialStores(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	vc := vault.NewTestVaultServer(t)
	_, token := vc.CreateToken(t)
	res := vaultCredStoreResource(vc,
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
				Config:      testConfig(url, fooCredentialStoresDataMissingScopeId),
				ExpectError: regexp.MustCompile("Improperly formatted identifier."),
			},
			{
				Config: testConfig(url, fooOrg, firstProjectFoo, res, fooCredentialStoresData),

				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.boundary_credential_stores.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_credential_stores.foo", "items.0.%", "10"),
					resource.TestCheckResourceAttrSet("data.boundary_credential_stores.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_credential_stores.foo", "items.0.description", "the foo"),
					resource.TestCheckResourceAttrSet("data.boundary_credential_stores.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_credential_stores.foo", "items.0.name", "foo"),
					resource.TestCheckResourceAttr("data.boundary_credential_stores.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_credential_stores.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_credential_stores.foo", "items.0.scope.0.description", "foo"),
					resource.TestCheckResourceAttrSet("data.boundary_credential_stores.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_credential_stores.foo", "items.0.scope.0.name", "proj1"),
					resource.TestCheckResourceAttrSet("data.boundary_credential_stores.foo", "items.0.scope.0.parent_scope_id"),
					resource.TestCheckResourceAttr("data.boundary_credential_stores.foo", "items.0.scope.0.type", "project"),
					resource.TestCheckResourceAttrSet("data.boundary_credential_stores.foo", "items.0.scope_id"),
					resource.TestCheckResourceAttr("data.boundary_credential_stores.foo", "items.0.type", "vault"),
					resource.TestCheckResourceAttrSet("data.boundary_credential_stores.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_credential_stores.foo", "items.0.version", "1"),
					resource.TestCheckResourceAttrSet("data.boundary_credential_stores.foo", "scope_id"),
				),
			},
		},
	})
}
