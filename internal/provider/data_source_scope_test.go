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
	orgName        = "test org scope"
	projectName    = "test project scope"
	notProjectName = "test project scope with wrong name"
	scopeDesc      = "created to test the scope datasource"
)

var scopeCreateAndRead = fmt.Sprintf(`
resource "boundary_scope" "global" {
	global_scope = true
	name = "global"
	description = "Global Scope"
	scope_id = "global"
}

resource "boundary_scope" "org" {
	scope_id = boundary_scope.global.id
	name = "%s"
	description = "%s"
}

resource "boundary_scope" "project" {
	depends_on = [boundary_role.org_admin]
	scope_id = boundary_scope.org.id
	name = "%s"
	description = "%s"
}

resource "boundary_role" "org_admin" {
	scope_id = "global"
	grant_scope_id = boundary_scope.org.id
	grant_strings = ["id=*;type=*;actions=*"]
	principal_ids = ["u_auth"]
}

data "boundary_scope" "org" {
	depends_on = [boundary_scope.org]
	scope_id = "global"
	name = "%s"
}

data "boundary_scope" "project" {
	depends_on = [boundary_scope.project]
	scope_id = data.boundary_scope.org.id
	name = "%s"
}`, orgName, scopeDesc, projectName, scopeDesc, orgName, projectName)

func TestAccScopeRead(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckScopeResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// create and read
				Config: testConfig(url, scopeCreateAndRead),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists(provider, "boundary_scope.org"),
					resource.TestCheckResourceAttr("boundary_scope.org", "description", scopeDesc),
					resource.TestCheckResourceAttr("boundary_scope.org", "name", orgName),
					testAccCheckScopeResourceExists(provider, "boundary_scope.project"),
					resource.TestCheckResourceAttr("boundary_scope.project", "description", scopeDesc),
					resource.TestCheckResourceAttr("boundary_scope.project", "name", projectName),
					// Check attributes on the org datasource
					resource.TestCheckResourceAttrSet("data.boundary_scope.org", "scope_id"),
					resource.TestCheckResourceAttrSet("data.boundary_scope.org", "id"),
					resource.TestCheckResourceAttr("data.boundary_scope.org", "name", orgName),
					resource.TestCheckResourceAttr("data.boundary_scope.org", "description", scopeDesc),
					// Check attributes on the project datasource
					resource.TestCheckResourceAttrSet("data.boundary_scope.project", "scope_id"),
					resource.TestCheckResourceAttrSet("data.boundary_scope.project", "id"),
					resource.TestCheckResourceAttr("data.boundary_scope.project", "name", projectName),
					resource.TestCheckResourceAttr("data.boundary_scope.project", "description", scopeDesc),
				),
			},
		},
	})
}
