// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	fooAuthMethodDesc       = "test auth method"
	fooAuthMethodDescUpdate = "test auth method update"
)

var (
	fooAuthMethod = fmt.Sprintf(`
resource "boundary_auth_method_password" "foo" {
	name        = "test"
	description = "%s"
	type        = "password"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`, fooAuthMethodDesc)

	fooAuthMethodUpdate = fmt.Sprintf(`
resource "boundary_auth_method_password" "foo" {
	name        = "test"
	description = "%s"
	type        = "password"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`, fooAuthMethodDescUpdate)
)

func TestAccAuthMethodPassword(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAuthMethodResourceDestroy(t, provider, passwordAuthMethodType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, fooAuthMethod),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "description", fooAuthMethodDesc),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "type", "password"),
					testAccCheckAuthMethodResourceExists(provider, "boundary_auth_method_password.foo"),
				),
			},
			importStep("boundary_auth_method_password.foo"),
			{
				// update
				Config: testConfig(url, fooOrg, fooAuthMethodUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "description", fooAuthMethodDescUpdate),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "type", "password"),
					testAccCheckAuthMethodResourceExists(provider, "boundary_auth_method_password.foo"),
				),
			},
			importStep("boundary_auth_method_password.foo"),
		},
	})
}
