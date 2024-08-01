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
	fooRolesDataMissingScope = `
data "boundary_roles" "foo" {}
`

	fooRolesData = `
data "boundary_roles" "foo" {
	depends_on = [boundary_role.foo]
	scope_id = boundary_role.foo.scope_id
}
`
)

func TestAccDataSourceRoles(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooRolesDataMissingScope),
				ExpectError: regexp.MustCompile("Improperly formatted field."),
			},
			{
				Config: testConfig(url, fooOrg, orgRole, fooRolesData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.%", "15"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.authorized_actions.#", "13"),
					resource.TestCheckResourceAttrSet("data.boundary_roles.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.description", "bar"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.grant_strings.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.grants.#", "0"),
					resource.TestCheckResourceAttrSet("data.boundary_roles.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.name", "test"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.principal_ids.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.principals.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.scope.0.description", ""),
					resource.TestCheckResourceAttrSet("data.boundary_roles.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.scope.0.name", "org1"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.scope.0.parent_scope_id", "global"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.scope.0.type", "org"),
					resource.TestCheckResourceAttrSet("data.boundary_roles.foo", "items.0.scope_id"),
					resource.TestCheckResourceAttrSet("data.boundary_roles.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.version", "2"),
					resource.TestCheckResourceAttrSet("data.boundary_roles.foo", "scope_id"),
				),
			},
		},
	})
}
