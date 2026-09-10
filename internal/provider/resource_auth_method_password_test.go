// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
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

	fooAuthMethodIsPrimaryUpdate = fmt.Sprintf(`
resource "boundary_auth_method_password" "foo" {
	name        = "test"
	description = "%s"
	type        = "password"
	scope_id    = boundary_scope.org1.id
	is_primary_for_scope = true
	depends_on  = [boundary_role.org1_admin]
}`, fooAuthMethodDesc)
)

func TestAccAuthMethodPassword(t *testing.T) {
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	suffix := id.UniqueId()
	url := cfg.BoundaryAddr

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAuthMethodResourceDestroy(t, provider, passwordAuthMethodType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg(suffix), fooAuthMethod),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "description", fooAuthMethodDesc),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "type", "password"),
					testAccCheckAuthMethodResourceExists(provider, "boundary_auth_method_password.foo"),
					testAccIsPrimaryForScope(provider, "boundary_auth_method_password.foo", false),
				),
			},
			importStep("boundary_auth_method_password.foo"),
			{
				// update
				Config: testConfig(url, fooOrg(suffix), fooAuthMethodUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "description", fooAuthMethodDescUpdate),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "type", "password"),
					testAccCheckAuthMethodResourceExists(provider, "boundary_auth_method_password.foo"),
					testAccIsPrimaryForScope(provider, "boundary_auth_method_password.foo", false),
				),
			},
			importStep("boundary_auth_method_password.foo"),
		},
	})
}

func TestAccAuthMethodPasswordIsPrimary(t *testing.T) {
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	suffix := id.UniqueId()
	url := cfg.BoundaryAddr

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAuthMethodResourceDestroy(t, provider, passwordAuthMethodType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg(suffix), fooAuthMethod),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "description", fooAuthMethodDesc),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "type", "password"),
					testAccCheckAuthMethodResourceExists(provider, "boundary_auth_method_password.foo"),
					testAccIsPrimaryForScope(provider, "boundary_auth_method_password.foo", false),
				),
			},
			importStep("boundary_auth_method_password.foo"),
			{
				// update
				Config: testConfig(url, fooOrg(suffix), fooAuthMethodIsPrimaryUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "description", fooAuthMethodDesc),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "type", "password"),
					testAccCheckAuthMethodResourceExists(provider, "boundary_auth_method_password.foo"),
					testAccIsPrimaryForScope(provider, "boundary_auth_method_password.foo", true),
				),
			},
		},
	})
}
