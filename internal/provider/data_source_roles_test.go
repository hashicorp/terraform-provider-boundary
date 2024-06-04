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
				ExpectError: regexp.MustCompile("scope_id: Improperly formatted field."),
			},
			{
				Config: testConfig(url, fooOrg, orgRole, fooRolesData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.%", "14"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.authorized_actions.#", "10"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.authorized_actions.0", "no-op"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.authorized_actions.1", "read"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.authorized_actions.2", "update"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.authorized_actions.3", "delete"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.authorized_actions.4", "add-principals"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.authorized_actions.5", "set-principals"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.authorized_actions.6", "remove-principals"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.authorized_actions.7", "add-grants"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.authorized_actions.8", "set-grants"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.authorized_actions.9", "remove-grants"),
					resource.TestCheckResourceAttrSet("data.boundary_roles.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.description", "bar"),
					resource.TestCheckResourceAttrSet("data.boundary_roles.foo", "items.0.grant_scope_id"),
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
					resource.TestCheckResourceAttr("data.boundary_roles.foo", "items.0.version", "1"),
					resource.TestCheckResourceAttrSet("data.boundary_roles.foo", "scope_id"),
				),
			},
		},
	})
}
