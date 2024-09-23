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
	fooWorkersDataMissingScopeId = `
data "boundary_workers" "foo" {}
`

	fooWorkersData = `
data "boundary_workers" "foo" {
	depends_on = [boundary_worker.controller_led]
	scope_id = boundary_worker.controller_led.scope_id
}
`
)

func TestAccDataSourceWorkers(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooWorkersDataMissingScopeId),
				ExpectError: regexp.MustCompile(""),
			},
			{
				Config: testConfig(url, controllerLedCreate, fooWorkersData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.boundary_workers.foo", "scope_id"),
					resource.TestCheckResourceAttr("data.boundary_workers.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_workers.foo", "items.0.%", "18"),
					resource.TestCheckResourceAttr("data.boundary_workers.foo", "items.0.authorized_actions.#", "7"),
					resource.TestCheckResourceAttrSet("data.boundary_workers.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_workers.foo", "items.0.description", "self managed worker description"),
					resource.TestCheckResourceAttrSet("data.boundary_workers.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_workers.foo", "items.0.name", "self managed worker"),
					resource.TestCheckResourceAttr("data.boundary_workers.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_workers.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_workers.foo", "items.0.scope.0.description", "Global Scope"),
					resource.TestCheckResourceAttrSet("data.boundary_workers.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_workers.foo", "items.0.scope.0.name", "global"),
					resource.TestCheckResourceAttr("data.boundary_workers.foo", "items.0.scope.0.type", "global"),
					resource.TestCheckResourceAttr("data.boundary_workers.foo", "items.0.type", "pki"),
					resource.TestCheckResourceAttrSet("data.boundary_workers.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_workers.foo", "items.0.version", "1"),
				),
			},
		},
	})
}
