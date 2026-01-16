// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/YakDriver/regexache"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	orgUserDataSource = fmt.Sprintf(`
resource "boundary_user" "org1" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}
data "boundary_user" "org1" {
	name     = "test"
	scope_id = boundary_scope.org1.id
	depends_on  = [boundary_user.org1]
}`, fooDescription)

	globalUserDataSource = `
data "boundary_user" "admin" {
	name        = "admin"
	depends_on  = [boundary_role.org1_admin]
}`
)

// NOTE: this test also tests out the direct token auth mechanism.

func TestAccUserDataSource_basicOrgUser(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]
	token := tc.Token().Token

	resourceName := "boundary_user.org1"
	dataSourceName := "data.boundary_user.org1"

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckUserResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfigWithToken(url, token, fooOrg, orgUserDataSource),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists(provider, resourceName),
					resource.TestCheckResourceAttr(dataSourceName, DescriptionKey, fooDescription),
					resource.TestCheckResourceAttr(dataSourceName, NameKey, "test"),
				),
			},
		},
	})
}

func TestAccUserDataSource_globalAdminUser(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]
	token := tc.Token().Token

	dataSourceName := "data.boundary_user.admin"

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfigWithToken(url, token, fooOrg, globalUserDataSource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, NameKey, "admin"),
					resource.TestCheckResourceAttr(dataSourceName, DescriptionKey, "Initial admin user within the \"global\" scope"),
					resource.TestCheckResourceAttr(dataSourceName, LoginNameKey, "testuser"),
					resource.TestMatchResourceAttr(dataSourceName, IDKey, regexache.MustCompile(`^u_.+`)),
					resource.TestMatchResourceAttr(dataSourceName, PrimaryAccountIdKey, regexache.MustCompile(`^acctpw_.+`)),
					resource.TestCheckResourceAttr(dataSourceName, "authorized_actions.#", "8"),
					resource.TestCheckResourceAttr(dataSourceName, "scope.0.name", "global"),
					resource.TestCheckResourceAttr(dataSourceName, "scope.0.id", "global"),
					resource.TestCheckResourceAttr(dataSourceName, "scope.0.type", "global"),
					resource.TestCheckResourceAttr(dataSourceName, "scope.0.description", "Global Scope"),
				),
			},
		},
	})
}
