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
	testRoleName = "my_role"
)

var roleReadGlobal = fmt.Sprintf(`
resource "boundary_user" "user" {
  name     = "my_user"
  scope_id    = "global"
}

resource "boundary_role" "role" {
  name        = "%s"
  description = "test role global"
  scope_id    = "global"
  grant_strings = [
    "ids=*;type=*;actions=read"
  ]
  principal_ids = [ boundary_user.user.id ]
}

data "boundary_role" "role" {
  depends_on = [ boundary_role.role ]
  name       = "%s"
}
`, testRoleName, testRoleName)

var roleReadOrg = fmt.Sprintf(`
resource "boundary_user" "user" {
  name     = "my_user"
  scope_id    = "global"
}

resource "boundary_role" "role" {
  name        = "%s"
  description = "test role org"
  scope_id    = boundary_scope.org1.id
  grant_strings = [
    "ids=*;type=*;actions=read"
  ]
  principal_ids = [ boundary_user.user.id ]
}

data "boundary_role" "role" {
  depends_on = [ boundary_role.role ]
  name       = "%s"
  scope_id   = boundary_scope.org1.id
}
`, testRoleName, testRoleName)

func TestAccRoleReadGlobal(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	dataSourceName := "data.boundary_role.role"

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, roleReadGlobal),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, IDKey),
					resource.TestCheckResourceAttrSet(dataSourceName, ScopeIdKey),
					resource.TestCheckResourceAttr(dataSourceName, NameKey, testRoleName),
					resource.TestCheckResourceAttrSet(dataSourceName, DescriptionKey),
					resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("%s.#", roleGrantStringsKey), "1"),
					resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("%s.0", roleGrantStringsKey), "ids=*;type=*;actions=read"),
					resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("%s.#", roleGrantScopeIdsKey), "1"),
					resource.TestCheckResourceAttrSet(dataSourceName, "scope.0.id"),
					resource.TestCheckResourceAttr(dataSourceName, "scope.0.name", "global"),
					resource.TestCheckResourceAttr(dataSourceName, "scope.0.type", "global"),
					resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("%s.#", rolePrincipalIdsKey), "1"),
					resource.TestCheckResourceAttrPair(dataSourceName, fmt.Sprintf("%s.0", rolePrincipalIdsKey), "boundary_user.user", "id"),
				),
			},
		},
	})
}

func TestAccRoleReadOrg(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	dataSourceName := "data.boundary_role.role"

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg, roleReadOrg),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, IDKey),
					resource.TestCheckResourceAttrSet(dataSourceName, ScopeIdKey),
					resource.TestCheckResourceAttr(dataSourceName, NameKey, testRoleName),
					resource.TestCheckResourceAttrSet(dataSourceName, DescriptionKey),
					resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("%s.#", roleGrantStringsKey), "1"),
					resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("%s.0", roleGrantStringsKey), "ids=*;type=*;actions=read"),
					resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("%s.#", roleGrantScopeIdsKey), "1"),
					resource.TestCheckResourceAttrSet(dataSourceName, "scope.0.id"),
					resource.TestCheckResourceAttr(dataSourceName, "scope.0.name", "org1"),
					resource.TestCheckResourceAttr(dataSourceName, "scope.0.type", "org"),
					resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("%s.#", rolePrincipalIdsKey), "1"),
					resource.TestCheckResourceAttrPair(dataSourceName, fmt.Sprintf("%s.0", rolePrincipalIdsKey), "boundary_user.user", "id"),
				),
			},
		},
	})
}
