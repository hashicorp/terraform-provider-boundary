// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var targetPasswordRead = `
data "boundary_target" "target_foo" {
	depends_on = [ boundary_target.foo ]
	name = "test"
	scope_id = boundary_scope.proj1.id
}`

func TestAccTargetRead(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg, firstProjectFoo, fooTargetWithIPAddress, targetPasswordRead),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetResourceExists(provider, "boundary_target.foo"),
					resource.TestCheckResourceAttrSet("data.boundary_target.target_foo", IDKey),
					resource.TestCheckResourceAttrSet("data.boundary_target.target_foo", NameKey),
					resource.TestCheckResourceAttr("data.boundary_target.target_foo", DescriptionKey, fooTargetDescription),
					resource.TestCheckResourceAttr("data.boundary_target.target_foo", TypeKey, "tcp"),
				),
			},
		},
	})
}
