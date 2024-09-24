package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	fooGroupsDataMissingScopeId = `
data "boundary_groups" "foo" {}
`
	fooGroupsData = `
data "boundary_groups" "foo" {
	scope_id = boundary_group.with_members.scope_id
}
`
)

func TestAccDataSourceGroups(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooGroupsDataMissingScopeId),
				ExpectError: regexp.MustCompile("scope_id: Incorrectly formatted identifier."),
			},
			{
				Config: testConfig(url, fooOrg, orgGroupWithMembers, fooGroupsData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.%", "11"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.authorized_actions.#", "7"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.authorized_actions.0", "no-op"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.authorized_actions.1", "read"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.authorized_actions.2", "update"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.authorized_actions.3", "delete"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.authorized_actions.4", "add-members"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.authorized_actions.5", "set-members"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.authorized_actions.6", "remove-members"),
					resource.TestCheckResourceAttrSet("data.boundary_groups.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.description", "with members"),
					resource.TestCheckResourceAttrSet("data.boundary_groups.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.member_ids.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.members.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.name", ""),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.scope.0.description", ""),
					resource.TestCheckResourceAttrSet("data.boundary_groups.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.scope.0.name", "org1"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.scope.0.parent_scope_id", "global"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.scope.0.type", "org"),
					resource.TestCheckResourceAttrSet("data.boundary_groups.foo", "items.0.scope_id"),
					resource.TestCheckResourceAttrSet("data.boundary_groups.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_groups.foo", "items.0.version", "2"),
					resource.TestCheckResourceAttrSet("data.boundary_groups.foo", "scope_id"),
				),
			},
		},
	})
}
