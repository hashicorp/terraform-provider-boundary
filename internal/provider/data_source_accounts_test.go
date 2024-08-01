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
	fooAccountDataMissingAuthMethodId = `
data "boundary_accounts" "foo" {}
`

	fooAccountData = `
data "boundary_accounts" "foo" {
	depends_on = [boundary_account.foo]
	auth_method_id = boundary_auth_method.foo.id
}
`
)

func TestAccDataSourceAccounts(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooAccountDataMissingAuthMethodId),
				ExpectError: regexp.MustCompile("Invalid formatted identifier."),
			},
			{
				Config: testConfig(url, fooOrg, fooAccount, fooAccountData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.boundary_accounts.foo", "auth_method_id"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.%", "11"),
					resource.TestCheckResourceAttrSet("data.boundary_accounts.foo", "items.0.auth_method_id"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.authorized_actions.#", "6"),
					resource.TestCheckResourceAttrSet("data.boundary_accounts.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.description", "test account"),
					resource.TestCheckResourceAttrSet("data.boundary_accounts.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.managed_group_ids.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.name", "test"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.scope.0.description", ""),
					resource.TestCheckResourceAttrSet("data.boundary_accounts.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.scope.0.name", "org1"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.scope.0.parent_scope_id", "global"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.scope.0.type", "org"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.type", "password"),
					resource.TestCheckResourceAttrSet("data.boundary_accounts.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.version", "1"),
				),
			},
		},
	})
}
