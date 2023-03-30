// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/api/managedgroups"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	fooManagedGroupLdap = fmt.Sprintf(`
resource "boundary_managed_group_ldap" "foo" {
	name           = "%s"
	description    = "%s"
	auth_method_id = boundary_auth_method_ldap.test-ldap.id
	group_names    = ["admin", "users"]
}`, managedGroupName, managedGroupDescription)

	fooManagedGroupLdapUpdate = fmt.Sprintf(`
resource "boundary_managed_group_ldap" "foo" {
	name           = "%s"
	description    = "%s"
	auth_method_id = boundary_auth_method_ldap.test-ldap.id
	group_names    = ["admin-updated", "users-updated"]
}`, managedGroupName+managedGroupUpdate, managedGroupDescription+managedGroupUpdate)
)

func TestAccManagedGroupLdap(t *testing.T) {
	wrapper := testWrapper(context.Background(), t, tcRecoveryKey)
	tc := controller.NewTestController(t, append(tcConfig, controller.WithRecoveryKms(wrapper))...)

	createConfig := fmt.Sprintf(testAuthMethodLdap, testAuthMethodLdapDesc, testAuthMethodLdapCert)

	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckManagedGroupResourceDestroy(t, provider, ldapManagedGroupType),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, createConfig, fooManagedGroupLdap),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckManagedGroupResourceExists(provider, "boundary_managed_group_ldap.foo"),
					resource.TestCheckResourceAttr("boundary_managed_group_ldap.foo", DescriptionKey, managedGroupDescription),
					resource.TestCheckResourceAttr("boundary_managed_group_ldap.foo", NameKey, managedGroupName),
					testAccCheckManagedGrpAttrAryValueSet(provider, "boundary_managed_group_ldap.foo", managedGroupLdapGroupNamesKey, []string{"admin", "users"}),
				),
			},
			importStep("boundary_managed_group_ldap.foo"),
			{
				// test update
				Config: testConfig(url, fooOrg, createConfig, fooManagedGroupLdapUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckManagedGroupResourceExists(provider, "boundary_managed_group_ldap.foo"),
					resource.TestCheckResourceAttr("boundary_managed_group_ldap.foo", DescriptionKey, managedGroupDescription+managedGroupUpdate),
					resource.TestCheckResourceAttr("boundary_managed_group_ldap.foo", NameKey, managedGroupName+managedGroupUpdate),
					testAccCheckManagedGrpAttrAryValueSet(provider, "boundary_managed_group_ldap.foo", managedGroupLdapGroupNamesKey, []string{"admin-updated", "users-updated"}),
				),
			},
			importStep("boundary_managed_group_ldap.foo"),
		},
	})
}

func testAccCheckManagedGrpAttrAryValueSet(testProvider *schema.Provider, name string, key string, strAry []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("auth method resource not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("auth method resource ID is not set")
		}

		md := testProvider.Meta().(*metaData)
		grpClient := managedgroups.NewClient(md.client)

		amr, err := grpClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading auth method %q: %v", id, err)
		}

		for _, got := range amr.Item.Attributes[key].([]interface{}) {
			ok := false
			for _, expected := range strAry {
				if strings.TrimSpace(got.(string)) == strings.TrimSpace(expected) {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("value not found in boundary\n %s: %s\n", key, got.(string))
			}
		}

		return nil
	}
}
