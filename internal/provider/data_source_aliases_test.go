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
	fooAliasesDataMissingScope = `
data "boundary_aliases" "foo" {}
`

	fooAliasesData = `
data "boundary_aliases" "foo" {
	depends_on = [boundary_alias_target.example]
	scope_id = boundary_alias_target.example.scope_id
}
`
)

func TestAccDataSourceAliases(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	res := targetAliasResource(targetAliasName, targetAliasDesc, targetAliasValue)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooAliasesDataMissingScope),
				ExpectError: regexp.MustCompile(""),
			},
			{
				Config: testConfig(url, fooOrg, firstProjectFoo, fooBarTarget, res, fooAliasesData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.boundary_aliases.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_aliases.foo", "items.0.%", "12"),
					resource.TestCheckResourceAttr("data.boundary_aliases.foo", "items.0.authorized_actions.#", "4"),
					resource.TestCheckResourceAttrSet("data.boundary_aliases.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_aliases.foo", "items.0.description", "the foo"),
					resource.TestCheckResourceAttrSet("data.boundary_aliases.foo", "items.0.destination_id"),
					resource.TestCheckResourceAttrSet("data.boundary_aliases.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_aliases.foo", "items.0.name", "foo"),
					resource.TestCheckResourceAttr("data.boundary_aliases.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_aliases.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_aliases.foo", "items.0.scope.0.description", "Global Scope"),
					resource.TestCheckResourceAttrSet("data.boundary_aliases.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_aliases.foo", "items.0.scope.0.name", "global"),
					resource.TestCheckResourceAttr("data.boundary_aliases.foo", "items.0.scope.0.parent_scope_id", ""),
					resource.TestCheckResourceAttr("data.boundary_aliases.foo", "items.0.scope.0.type", "global"),
					resource.TestCheckResourceAttrSet("data.boundary_aliases.foo", "items.0.scope_id"),
					resource.TestCheckResourceAttrSet("data.boundary_aliases.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_aliases.foo", "items.0.version", "1"),
					resource.TestCheckResourceAttrSet("data.boundary_aliases.foo", "scope_id"),
				),
			},
		},
	})
}
