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
	fooAccountPasswordDesc       = "test account"
	fooAccountPasswordDescUpdate = "test account update"
)

var (
	fooAccountPassword = fmt.Sprintf(`
resource "boundary_auth_method" "foo" {
	name        = "test"
	description = "test account"
	type        = "password"
	scope_id    = boundary_scope.org1.id
	depends_on = [boundary_role.org1_admin]
}

resource "boundary_account_password" "foo" {
	name           = "test"
	description    = "%s"
	type           = "password"
	login_name     = "foo"
	password       = "foofoofoo"
	auth_method_id = boundary_auth_method.foo.id
}`, fooAccountPasswordDesc)

	fooAccountPasswordUpdate = fmt.Sprintf(`
resource "boundary_auth_method" "foo" {
	name        = "test"
	description = "test account"
	type        = "password"
	scope_id    = boundary_scope.org1.id
	depends_on = [boundary_role.org1_admin]
}

resource "boundary_account_password" "foo" {
	name           = "test"
	description    = "%s"
	type           = "password"
	login_name     = "foo"
	password       = "foofoofoo"
	auth_method_id = boundary_auth_method.foo.id
}`, fooAccountPasswordDescUpdate)

	fooAccountPasswordWithoutTypeField = fmt.Sprintf(`
	resource "boundary_auth_method" "foo" {
		name        = "test"
		description = "test account"
		type        = "password"
		scope_id    = boundary_scope.org1.id
		depends_on = [boundary_role.org1_admin]
	}
	
	resource "boundary_account_password" "foo" {
		name           = "test"
		description    = "%s"
		login_name     = "foo"
		password       = "foofoofoo"
		auth_method_id = boundary_auth_method.foo.id
	}`, fooAccountPasswordDescUpdate)
)

func TestAccAccount(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAccountResourceDestroy(t, provider, passwordAccountType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, fooAccountPassword),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_account_password.foo", "description", fooAccountPasswordDesc),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "type", "password"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "login_name", "foo"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "password", "foofoofoo"),
					testAccCheckAccountResourceExists(provider, "boundary_account_password.foo"),
				),
			},
			importStep("boundary_account_password.foo", "password"),
			{
				// update
				Config: testConfig(url, fooOrg, fooAccountPasswordUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_account_password.foo", "description", fooAccountPasswordDescUpdate),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "type", "password"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "login_name", "foo"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "password", "foofoofoo"),
					testAccCheckAccountResourceExists(provider, "boundary_account_password.foo"),
				),
			},
			importStep("boundary_account_password.foo", "password"),
			{
				// update without passing type field
				Config: testConfig(url, fooOrg, fooAccountPasswordWithoutTypeField),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_account_password.foo", "description", fooAccountPasswordDescUpdate),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "type", "password"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "login_name", "foo"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "password", "foofoofoo"),
					testAccCheckAccountResourceExists(provider, "boundary_account_password.foo"),
				),
			},
			importStep("boundary_account_password.foo", "password"),
		},
	})
}
