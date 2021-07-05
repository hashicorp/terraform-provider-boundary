package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	fooAuthMethodsDataMissingScope = `
data "boundary_auth_methods" "foo" {}
`
	fooAuthMethodsData = `
data "boundary_auth_methods" "foo" {
	scope_id = boundary_auth_method.foo.scope_id
}
`
)

func TestAccDataSourceAuthMethods(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories((&provider)),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooAuthMethodsDataMissingScope),
				ExpectError: regexp.MustCompile("scope_id: This field must be 'global' or a valid org scope id."),
			},
			{
				Config: testConfig(url, fooOrg, fooBaseAuthMethod, fooAuthMethodsData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.%", "11"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.authorized_actions.#", "5"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.authorized_actions.0", "no-op"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.authorized_actions.1", "read"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.authorized_actions.2", "update"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.authorized_actions.3", "delete"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.authorized_actions.4", "authenticate"),
					resource.TestCheckResourceAttrSet("data.boundary_auth_methods.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.description", "test auth method"),
					resource.TestCheckResourceAttrSet("data.boundary_auth_methods.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.is_primary", "false"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.name", "test"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.scope.0.description", ""),
					resource.TestCheckResourceAttrSet("data.boundary_auth_methods.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.scope.0.name", "org1"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.scope.0.parent_scope_id", "global"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.scope.0.type", "org"),
					resource.TestCheckResourceAttrSet("data.boundary_auth_methods.foo", "items.0.scope_id"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.type", "password"),
					resource.TestCheckResourceAttrSet("data.boundary_auth_methods.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_auth_methods.foo", "items.0.version", "1"),
					resource.TestCheckResourceAttrSet("data.boundary_auth_methods.foo", "scope_id"),
				),
			},
		},
	})
}
