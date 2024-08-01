package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	fooUsersDataMissingScope = `
data "boundary_users" "foo" {}
`
	fooUsersData = `
data "boundary_users" "foo" {
	scope_id = boundary_user.org1.scope_id
}
`
)

func TestAccDataSourceUsers(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]
	token := tc.Token().Token

	var provider *schema.Provider

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooUsersDataMissingScope),
				ExpectError: regexp.MustCompile("scope_id: Must be 'global' or a valid org scope id when listing."),
			},
			{
				Config: testConfigWithToken(url, token, fooOrg, orgUser, fooUsersData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.%", "15"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.account_ids.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.accounts.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.authorized_actions.#", "7"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.authorized_actions.0", "no-op"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.authorized_actions.1", "read"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.authorized_actions.2", "update"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.authorized_actions.3", "delete"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.authorized_actions.4", "add-accounts"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.authorized_actions.5", "set-accounts"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.authorized_actions.6", "remove-accounts"),
					resource.TestCheckResourceAttrSet("data.boundary_users.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.description", "bar"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.email", ""),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.full_name", ""),
					resource.TestCheckResourceAttrSet("data.boundary_users.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.login_name", ""),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.name", "test"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.primary_account_id", ""),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.scope.0.description", ""),
					resource.TestCheckResourceAttrSet("data.boundary_users.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.scope.0.name", "org1"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.scope.0.parent_scope_id", "global"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.scope.0.type", "org"),
					resource.TestCheckResourceAttrSet("data.boundary_users.foo", "items.0.scope_id"),
					resource.TestCheckResourceAttrSet("data.boundary_users.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_users.foo", "items.0.version", "1"),
					resource.TestCheckResourceAttrSet("data.boundary_users.foo", "scope_id"),
				),
			},
		},
	})
}
