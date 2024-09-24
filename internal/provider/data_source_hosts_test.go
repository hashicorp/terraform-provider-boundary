package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	fooHostsDataMissingHostCatalogId = `
data "boundary_hosts" "foo" {}
`
	fooHostsData = `
data "boundary_hosts" "foo" {
	host_catalog_id = boundary_host.foo.host_catalog_id
}
`
)

func TestAccDataSourceHosts(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooHostsDataMissingHostCatalogId),
				ExpectError: regexp.MustCompile("host_catalog_id: The field is incorrectly formatted."),
			},
			{
				Config: testConfig(url, fooOrg, firstProjectFoo, projHost, fooHostsData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.boundary_hosts.foo", "host_catalog_id"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.%", "11"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.authorized_actions.#", "4"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.authorized_actions.0", "no-op"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.authorized_actions.1", "read"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.authorized_actions.2", "update"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.authorized_actions.3", "delete"),
					resource.TestCheckResourceAttrSet("data.boundary_hosts.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.description", "test host"),
					resource.TestCheckResourceAttrSet("data.boundary_hosts.foo", "items.0.host_catalog_id"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.host_set_ids.#", "0"),
					resource.TestCheckResourceAttrSet("data.boundary_hosts.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.name", "test"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.scope.0.description", "foo"),
					resource.TestCheckResourceAttrSet("data.boundary_hosts.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.scope.0.name", "proj1"),
					resource.TestCheckResourceAttrSet("data.boundary_hosts.foo", "items.0.scope.0.parent_scope_id"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.scope.0.type", "project"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.type", "static"),
					resource.TestCheckResourceAttrSet("data.boundary_hosts.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.version", "1"),
				),
			},
		},
	})
}
