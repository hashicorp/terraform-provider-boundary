// Copyright IBM Corp. 2020, 2025
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
	testGroupName = "test_group"
)

var groupReadGlobal = fmt.Sprintf(`

resource "boundary_user" "user" {
	description = "user"
	scope_id    = "global"
	depends_on  = [boundary_role.org1_admin]
}

resource "boundary_group" "group" {
	name 	    = "%s"
	description = "test"
	scope_id    = "global"
	member_ids  = [boundary_user.user.id]
	depends_on  = [boundary_user.user]
}

data "boundary_group" "group" {
	depends_on = [ boundary_group.group ]
	name = "%s"
}`, testGroupName, testGroupName)

var groupReadOrg = fmt.Sprintf(`
resource "boundary_user" "user" {
	description = "user"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}

resource "boundary_group" "group" {
	name 	    = "%s"
	description = "test"
	scope_id    = boundary_scope.org1.id
	member_ids  = [boundary_user.user.id]
	depends_on  = [boundary_user.user]
}

data "boundary_group" "group" {
	depends_on = [ boundary_group.group ]
	name = "%s"
	scope_id = boundary_scope.org1.id
}`, testGroupName, testGroupName)

func TestAccGroupReadGlobal(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg, groupReadGlobal),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists(provider, "boundary_group.group"),
					resource.TestCheckResourceAttrSet("data.boundary_group.group", IDKey),
					resource.TestCheckResourceAttrSet("data.boundary_group.group", ScopeIdKey),
					resource.TestCheckResourceAttr("data.boundary_group.group", NameKey, testGroupName),
					resource.TestCheckResourceAttrSet("data.boundary_group.group", DescriptionKey),
					resource.TestCheckResourceAttrSet("data.boundary_group.group", fmt.Sprintf("%s.#", GroupMemberIdsKey)),
					resource.TestCheckResourceAttrSet("data.boundary_group.group", "scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_group.group", "scope.0.name", "global"),
					resource.TestCheckResourceAttr("data.boundary_group.group", "scope.0.type", "global"),
				),
			},
		},
	})
}

func TestAccGroupReadOrg(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg, groupReadOrg),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists(provider, "boundary_group.group"),
					resource.TestCheckResourceAttrSet("data.boundary_group.group", IDKey),
					resource.TestCheckResourceAttrSet("data.boundary_group.group", ScopeIdKey),
					resource.TestCheckResourceAttr("data.boundary_group.group", NameKey, testGroupName),
					resource.TestCheckResourceAttrSet("data.boundary_group.group", DescriptionKey),
					resource.TestCheckResourceAttrSet("data.boundary_group.group", fmt.Sprintf("%s.#", GroupMemberIdsKey)),
					resource.TestCheckResourceAttrSet("data.boundary_group.group", "scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_group.group", "scope.0.name", "org1"),
					resource.TestCheckResourceAttr("data.boundary_group.group", "scope.0.type", "org"),
				),
			},
		},
	})
}
