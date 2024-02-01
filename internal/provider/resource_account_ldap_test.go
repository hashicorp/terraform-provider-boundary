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
	testAccountLdapDesc       = "test account"
	testAccountLdapDescUpdate = "test account update"
	testAccountLdapName       = "test account name"
	testAccountLdapNameUpdate = "test account name updated"
)

var (
	testAccountLdap = fmt.Sprintf(`
resource "boundary_auth_method_ldap" "foo" {
	name        = "test"
	description = "test account"
	type        = "ldap"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]

	urls        = ["ldaps://ldap1", "ldaps://ldap2"]
}

resource "boundary_account_ldap" "foo" {
	name           = "%s"
	description    = "%s"
	type           = "ldap"
	login_name     = "foo"
	auth_method_id = boundary_auth_method_ldap.foo.id
}`, testAccountLdapName, testAccountLdapDesc)

	testAccountLdapUpdate = fmt.Sprintf(`
resource "boundary_auth_method_ldap" "foo" {
	name        = "test"
	description = "test account"
	type        = "ldap"
	scope_id    = boundary_scope.org1.id
	depends_on = [boundary_role.org1_admin]
	urls         	  = ["ldaps://ldap1", "ldaps://ldap2"]
}

resource "boundary_account_ldap" "foo" {
	name           = "%s"
	description    = "%s"
	type           = "ldap"
	login_name     = "foo"
	auth_method_id = boundary_auth_method_ldap.foo.id
}`, testAccountLdapNameUpdate, testAccountLdapDescUpdate)

	testAccountLdapWithoutTypeField = fmt.Sprintf(`
resource "boundary_auth_method_ldap" "foo" {
	name        = "test"
	description = "test account"
	type        = "ldap"
	scope_id    = boundary_scope.org1.id
	depends_on = [boundary_role.org1_admin]
	urls         	  = ["ldaps://ldap1", "ldaps://ldap2"]
}

resource "boundary_account_ldap" "foo" {
	name           = "%s"
	description    = "%s"
	login_name     = "foo"
	auth_method_id = boundary_auth_method_ldap.foo.id
}`, testAccountLdapNameUpdate, testAccountLdapDescUpdate)

	testProviderLdapAccountConfig = `
resource "boundary_account_ldap" "test-ldap" {
	name           = "alice"
	description    = "test"
	type           = "ldap"
	login_name     = "%s"
	auth_method_id = boundary_auth_method_ldap.test-ldap.id
}

resource "boundary_user" "alice" {
	name        = "alice"
	description = "test user resource"
	account_ids = [boundary_account_ldap.test-ldap.id]
	scope_id = boundary_scope.global.id
}

resource "boundary_role" "ldap_principal" {
	scope_id = boundary_scope.global.id
	grant_scope_id = boundary_scope.global.id
	grant_strings = ["ids=*;type=*;actions=*"]
	principal_ids = [boundary_user.alice.id]
}`
)

func TestAccLdapAccount(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAccountResourceDestroy(t, provider, baseAccountType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, testAccountLdap),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_account_ldap.foo", "description", testAccountLdapDesc),
					resource.TestCheckResourceAttr("boundary_account_ldap.foo", "name", testAccountLdapName),
					resource.TestCheckResourceAttr("boundary_account_ldap.foo", "type", "ldap"),
					resource.TestCheckResourceAttr("boundary_account_ldap.foo", "login_name", "foo"),
					testAccCheckAccountResourceExists(provider, "boundary_account_ldap.foo"),
				),
			},
			importStep("boundary_account_ldap.foo", "ldap"),
			{
				// update
				Config: testConfig(url, fooOrg, testAccountLdapUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_account_ldap.foo", "description", testAccountLdapDescUpdate),
					resource.TestCheckResourceAttr("boundary_account_ldap.foo", "name", testAccountLdapNameUpdate),
					resource.TestCheckResourceAttr("boundary_account_ldap.foo", "type", "ldap"),
					resource.TestCheckResourceAttr("boundary_account_ldap.foo", "login_name", "foo"),
					testAccCheckAccountResourceExists(provider, "boundary_account_ldap.foo"),
				),
			},
			importStep("boundary_account_ldap.foo", "ldap"),
			{
				// update without passing type field
				Config: testConfig(url, fooOrg, testAccountLdapWithoutTypeField),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_account_ldap.foo", "description", testAccountLdapDescUpdate),
					resource.TestCheckResourceAttr("boundary_account_ldap.foo", "name", testAccountLdapNameUpdate),
					resource.TestCheckResourceAttr("boundary_account_ldap.foo", "type", "ldap"),
					resource.TestCheckResourceAttr("boundary_account_ldap.foo", "login_name", "foo"),
					testAccCheckAccountResourceExists(provider, "boundary_account_ldap.foo"),
				),
			},
			importStep("boundary_account_ldap.foo", "ldap"),
		},
	})
}
