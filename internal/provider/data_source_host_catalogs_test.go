// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	fooHostCatalogsDataMissingSopeId = `
data "boundary_host_catalogs" "foo" {}
`
	fooHostCatalogsData = `
data "boundary_host_catalogs" "foo" {
	depends_on = [boundary_host_catalog.foo]
	scope_id = boundary_host_catalog.foo.scope_id
}
`
)

func TestAccDataSourceHostCatalogs(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resName := "boundary_host_catalog"
	typeStr := `type = "static"`

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooHostCatalogsDataMissingSopeId),
				ExpectError: regexp.MustCompile("Improperly formatted identifier."),
			},
			{
				Config: testConfig(url, fooOrg, firstProjectFoo, fmt.Sprintf(projHostCatalog, resName, typeStr), fooHostCatalogsData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.boundary_host_catalogs.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_host_catalogs.foo", "items.0.%", "13"),
					resource.TestCheckResourceAttr("data.boundary_host_catalogs.foo", "items.0.authorized_actions.#", "4"),
					resource.TestCheckResourceAttrSet("data.boundary_host_catalogs.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_host_catalogs.foo", "items.0.description", "bar"),
					resource.TestCheckResourceAttrSet("data.boundary_host_catalogs.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_host_catalogs.foo", "items.0.name", "foo"),
					resource.TestCheckResourceAttr("data.boundary_host_catalogs.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_host_catalogs.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_host_catalogs.foo", "items.0.scope.0.description", "foo"),
					resource.TestCheckResourceAttrSet("data.boundary_host_catalogs.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_host_catalogs.foo", "items.0.scope.0.name", "proj1"),
					resource.TestCheckResourceAttrSet("data.boundary_host_catalogs.foo", "items.0.scope.0.parent_scope_id"),
					resource.TestCheckResourceAttr("data.boundary_host_catalogs.foo", "items.0.scope.0.type", "project"),
					resource.TestCheckResourceAttrSet("data.boundary_host_catalogs.foo", "items.0.scope_id"),
					resource.TestCheckResourceAttr("data.boundary_host_catalogs.foo", "items.0.type", "static"),
					resource.TestCheckResourceAttrSet("data.boundary_host_catalogs.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_host_catalogs.foo", "items.0.version", "1"),
					resource.TestCheckResourceAttrSet("data.boundary_host_catalogs.foo", "scope_id"),
				),
			},
		},
	})
}
