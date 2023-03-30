// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/managedgroups"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/cap/oidc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	managedGroupName        = "test"
	managedGroupDescription = "test managed group"
	managedGroupUpdate      = "_update"
)

var (
	fooManagedGroup = fmt.Sprintf(`
resource "boundary_managed_group" "foo" {
	name           = "%s"
	description    = "%s"
	auth_method_id = boundary_auth_method_oidc.foo.id
	filter         = "name == \"foo\""
}`, managedGroupName, managedGroupDescription)

	fooManagedGroupUpdate = fmt.Sprintf(`
resource "boundary_managed_group" "foo" {
	name           = "%s"
	description    = "%s"
	auth_method_id = boundary_auth_method_oidc.foo.id
	filter         = "name == \"bar\""
}`, managedGroupName+managedGroupUpdate, managedGroupDescription+managedGroupUpdate)
)

func TestAccManagedGroup(t *testing.T) {
	wrapper := testWrapper(context.Background(), t, tcRecoveryKey)
	tp := oidc.StartTestProvider(t)
	tc := controller.NewTestController(t, append(tcConfig, controller.WithRecoveryKms(wrapper))...)

	tpCert := strings.TrimSpace(tp.CACert())
	createConfig := fmt.Sprintf(fooAuthMethodOidc, fooAuthMethodOidcDesc, tp.Addr(), tpCert)

	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckManagedGroupResourceDestroy(t, provider, baseManagedGroupType),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, createConfig, fooManagedGroup),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckManagedGroupResourceExists(provider, "boundary_managed_group.foo"),
					resource.TestCheckResourceAttr("boundary_managed_group.foo", DescriptionKey, managedGroupDescription),
					resource.TestCheckResourceAttr("boundary_managed_group.foo", NameKey, managedGroupName),
					resource.TestCheckResourceAttr("boundary_managed_group.foo", managedGroupFilterKey, `name == "foo"`),
				),
			},
			importStep("boundary_managed_group.foo"),
			{
				// test update
				Config: testConfig(url, fooOrg, createConfig, fooManagedGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckManagedGroupResourceExists(provider, "boundary_managed_group.foo"),
					resource.TestCheckResourceAttr("boundary_managed_group.foo", DescriptionKey, managedGroupDescription+managedGroupUpdate),
					resource.TestCheckResourceAttr("boundary_managed_group.foo", NameKey, managedGroupName+managedGroupUpdate),
					resource.TestCheckResourceAttr("boundary_managed_group.foo", managedGroupFilterKey, `name == "bar"`),
				),
			},
		},
	})
}

func testAccCheckManagedGroupResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("No ID is set")
		}

		md := testProvider.Meta().(*metaData)
		grpClient := managedgroups.NewClient(md.client)

		_, err := grpClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading group %q: %v", id, err)
		}

		return nil
	}
}

type managedGroupType string

const (
	baseManagedGroupType managedGroupType = "boundary_managed_group"
	ldapManagedGroupType managedGroupType = "boundary_managed_group_ldap"
)

func testAccCheckManagedGroupResourceDestroy(t *testing.T, testProvider *schema.Provider, typ managedGroupType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}

		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case string(typ):
				grpClient := managedgroups.NewClient(md.client)
				id := rs.Primary.ID

				_, err := grpClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed resource %q: %v", id, apiErr)
				}

			default:
				continue
			}
		}
		return nil
	}
}
