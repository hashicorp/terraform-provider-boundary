// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var fooScopesData = `
data "boundary_scopes" "foo" {}
`

func TestAccDataSourceScopes(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooScopesData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.%", "12"),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.authorized_actions.#", "6"),
					resource.TestCheckResourceAttrSet("data.boundary_scopes.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.description", "Provides an initial org scope in Boundary"),
					resource.TestCheckResourceAttrSet("data.boundary_scopes.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.name", "Generated org scope"),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.primary_auth_method_id", ""),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.scope.0.description", "Global Scope"),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.scope.0.id", "global"),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.scope.0.name", "global"),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.scope.0.parent_scope_id", ""),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.scope.0.type", "global"),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.scope_id", "global"),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.type", "org"),
					resource.TestCheckResourceAttrSet("data.boundary_scopes.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_scopes.foo", "items.0.version", "1"),
				),
			},
		},
	})
}
