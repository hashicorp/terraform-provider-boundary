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
	testHost = `
resource "boundary_host_catalog" "foo" {
	depends_on  = [boundary_role.proj1_admin]
	type 		= "static"
	name        = "foo"
	description = "bar"
	scope_id    = boundary_scope.proj1.id
}

resource "boundary_host" "foo" {
	depends_on  	= [boundary_host_catalog.foo]
	type            = "static"
	host_catalog_id = boundary_host_catalog.foo.id
	name            = "host_1"
	description     = "My first host!"
	address         = "10.0.0.1"
}
`

	fooHostsDataMissingHostCatalogId = `
data "boundary_hosts" "foo" {}
`
	fooHostsData = `
data "boundary_hosts" "foo" {
	depends_on = [boundary_host.foo]
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
				ExpectError: regexp.MustCompile("Improperly formatted identifier."),
			},
			{
				Config: testConfig(url, fooOrg, firstProjectFoo, testHost, fooHostsData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.boundary_hosts.foo", "host_catalog_id"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.%", "16"),
					resource.TestCheckResourceAttrSet("data.boundary_hosts.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.description", "My first host!"),
					resource.TestCheckResourceAttrSet("data.boundary_hosts.foo", "items.0.host_catalog_id"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.host_set_ids.#", "0"),
					resource.TestCheckResourceAttrSet("data.boundary_hosts.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_hosts.foo", "items.0.name", "host_1"),
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
