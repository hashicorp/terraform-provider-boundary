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
	testAuthMethodName = "test_auth_method"
)

var authMethodReadGlobal = fmt.Sprintf(`
resource "boundary_auth_method" "auth_method" {
	name 		= "%s"
	description = "test"
	scope_id    = "global"
	type 		= "password"
	depends_on  = [boundary_role.org1_admin]
}

data "boundary_auth_method" "auth_method" {
	depends_on = [ boundary_auth_method.auth_method ]
	name 	   = "%s"
}`, testAuthMethodName, testAuthMethodName)

var authMethodReadOrg = fmt.Sprintf(`
resource "boundary_auth_method" "auth_method" {
	name 		= "%s"
	description = "test"
	scope_id    = boundary_scope.org1.id
	type 		= "password"
	depends_on  = [boundary_role.org1_admin]
}

data "boundary_auth_method" "auth_method" {
	depends_on = [ boundary_auth_method.auth_method ]
	name = "%s"
	scope_id = boundary_scope.org1.id
}`, testAuthMethodName, testAuthMethodName)

func TestAccAuthMethodReadGlobal(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg, authMethodReadGlobal),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", IDKey),
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", ScopeIdKey),
					resource.TestCheckResourceAttr("data.boundary_auth_method.auth_method", NameKey, testAuthMethodName),
					resource.TestCheckResourceAttr("data.boundary_auth_method.auth_method", TypeKey, "password"),
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", DescriptionKey),
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", "scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_auth_method.auth_method", "scope.0.name", "global"),
					resource.TestCheckResourceAttr("data.boundary_auth_method.auth_method", "scope.0.type", "global"),
				),
			},
		},
	})
}

func TestAccAuthMethodReadOrg(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg, authMethodReadOrg),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", IDKey),
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", ScopeIdKey),
					resource.TestCheckResourceAttr("data.boundary_auth_method.auth_method", NameKey, testAuthMethodName),
					resource.TestCheckResourceAttr("data.boundary_auth_method.auth_method", TypeKey, "password"),
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", DescriptionKey),
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", "scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_auth_method.auth_method", "scope.0.name", "org1"),
					resource.TestCheckResourceAttr("data.boundary_auth_method.auth_method", "scope.0.type", "org"),
				),
			},
		},
	})
}
