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
	auth_method_id = boundary_account.foo.auth_method_id
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
				ExpectError: regexp.MustCompile("auth_method_id: Invalid formatted identifier."),
			},
			{
				Config: testConfig(url, fooOrg, fooAccount, fooAccountData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.boundary_accounts.foo", "auth_method_id"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.%", "11"),
					resource.TestCheckResourceAttrSet("data.boundary_accounts.foo", "items.0.auth_method_id"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.authorized_actions.#", "6"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.authorized_actions.0", "no-op"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.authorized_actions.1", "read"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.authorized_actions.2", "update"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.authorized_actions.3", "delete"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.authorized_actions.4", "set-password"),
					resource.TestCheckResourceAttr("data.boundary_accounts.foo", "items.0.authorized_actions.5", "change-password"),
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
