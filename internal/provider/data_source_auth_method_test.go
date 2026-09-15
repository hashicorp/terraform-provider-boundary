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

const (
	testAuthMethodName = "test_auth_method"
)

func authMethodReadGlobal(suffix string) string {
	name := testAuthMethodName + "-" + suffix
	return fmt.Sprintf(`
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
}`, name, name)
}

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
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	suffix := id.UniqueId()

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg(suffix), authMethodReadGlobal(suffix)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", IDKey),
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", ScopeIdKey),
					resource.TestMatchResourceAttr("data.boundary_auth_method.auth_method", NameKey, regexache.MustCompile(`^`+testAuthMethodName)),
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
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	suffix := id.UniqueId()

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg(suffix), authMethodReadOrg),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", IDKey),
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", ScopeIdKey),
					resource.TestCheckResourceAttr("data.boundary_auth_method.auth_method", NameKey, testAuthMethodName),
					resource.TestCheckResourceAttr("data.boundary_auth_method.auth_method", TypeKey, "password"),
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", DescriptionKey),
					resource.TestCheckResourceAttrSet("data.boundary_auth_method.auth_method", "scope.0.id"),
					resource.TestMatchResourceAttr("data.boundary_auth_method.auth_method", "scope.0.name", regexache.MustCompile(`^org1-`)),
					resource.TestCheckResourceAttr("data.boundary_auth_method.auth_method", "scope.0.type", "org"),
				),
			},
		},
	})
}
