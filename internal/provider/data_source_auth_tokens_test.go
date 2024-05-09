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

var fooAuthTokensDataMissingScopeId = `
data "boundary_auth_tokens" "foo" {}
`

var fooAuthTokensData = `
data "boundary_auth_tokens" "foo" {
	scope_id = "global"
}
`

func TestAccDataAuthTokens(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooAuthTokensDataMissingScopeId),
				ExpectError: regexp.MustCompile("Improperly formatted identifier."),
			},
			{
				Config: testConfig(url, fooAuthTokensData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.boundary_auth_tokens.foo", "scope_id"),
					resource.TestCheckResourceAttrSet("data.boundary_auth_tokens.foo", "items.0.account_id"),
					resource.TestCheckResourceAttrSet("data.boundary_auth_tokens.foo", "items.0.auth_method_id"),
					resource.TestCheckResourceAttr("data.boundary_auth_tokens.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttrSet("data.boundary_auth_tokens.foo", "items.0.scope.0.description"),
					resource.TestCheckResourceAttrSet("data.boundary_auth_tokens.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_auth_tokens.foo", "items.0.scope.0.name", "global"),
					resource.TestCheckResourceAttr("data.boundary_auth_tokens.foo", "items.0.scope.0.parent_scope_id", ""),
					resource.TestCheckResourceAttr("data.boundary_auth_tokens.foo", "items.0.scope.0.type", "global"),
				),
			},
		},
	})
}
