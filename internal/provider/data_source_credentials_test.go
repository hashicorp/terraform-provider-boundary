// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	fooCredentialsDataMissingCredentialStoreId = `
data "boundary_credentials" "foo" {}
`

	fooCredentialsData = `
data "boundary_credentials" "foo" {
	depends_on = [boundary_credential_username_password.example]
	credential_store_id = boundary_credential_username_password.example.credential_store_id
}
`
)

func TestAccDataSourceCredentials(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	res := usernamePasswordCredResource(
		usernamePasswordCredName,
		usernamePasswordCredDesc,
		usernamePasswordCredUsername,
		usernamePasswordCredPassword,
	)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooCredentialsDataMissingCredentialStoreId),
				ExpectError: regexp.MustCompile(""),
			},
			{
				Config: testConfig(url, fooOrg, firstProjectFoo, res, fooCredentialsData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.boundary_credentials.foo", "credential_store_id"),
					resource.TestCheckResourceAttr("data.boundary_credentials.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_credentials.foo", "items.0.%", "10"),
					resource.TestCheckResourceAttr("data.boundary_credentials.foo", "items.0.authorized_actions.#", "4"),
					resource.TestCheckResourceAttrSet("data.boundary_credentials.foo", "items.0.created_time"),
					resource.TestCheckResourceAttrSet("data.boundary_credentials.foo", "items.0.credential_store_id"),
					resource.TestCheckResourceAttr("data.boundary_credentials.foo", "items.0.description", "the foo"),
					resource.TestCheckResourceAttrSet("data.boundary_credentials.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_credentials.foo", "items.0.name", "foo"),
					resource.TestCheckResourceAttr("data.boundary_credentials.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_credentials.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_credentials.foo", "items.0.scope.0.description", "foo"),
					resource.TestCheckResourceAttrSet("data.boundary_credentials.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_credentials.foo", "items.0.scope.0.name", "proj1"),
					resource.TestCheckResourceAttrSet("data.boundary_credentials.foo", "items.0.scope.0.parent_scope_id"),
					resource.TestCheckResourceAttr("data.boundary_credentials.foo", "items.0.scope.0.type", "project"),
					resource.TestCheckResourceAttr("data.boundary_credentials.foo", "items.0.type", "username_password"),
					resource.TestCheckResourceAttrSet("data.boundary_credentials.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_credentials.foo", "items.0.version", "1"),
				),
			},
		},
	})
}
