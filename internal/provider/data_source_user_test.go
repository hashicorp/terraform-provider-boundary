// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/YakDriver/regexache"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
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
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	token := cfg.BoundaryAuthToken
	suffix := id.UniqueId()

	resourceName := "boundary_user.org1"
	dataSourceName := "data.boundary_user.org1"

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckUserResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfigWithToken(url, token, fooOrg(suffix), orgUserDataSource),
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
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	token := cfg.BoundaryAuthToken
	suffix := id.UniqueId()

	dataSourceName := "data.boundary_user.admin"

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfigWithToken(url, token, fooOrg(suffix), globalUserDataSource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, NameKey, "admin"),
					resource.TestCheckResourceAttr(dataSourceName, DescriptionKey, "Initial admin user within the \"global\" scope"),
					resource.TestCheckResourceAttr(dataSourceName, LoginNameKey, tcLoginName),
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
