package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/managedgroups"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	managedGroupName        = "test_managed_group"
	managedGroupDescription = "test managed group"
	managedGroupUpdate      = "_update"
)

var (
	orgManagedGroup = fmt.Sprintf(`
resource "boundary_managed_group" "test" {
	name        = "%s"
	description = "%s"
	scope_id    = boundary_scope.org1.id
	depends_on = [boundary_role.org1_admin]
}`, managedGroupName, managedGroupDescription)

	orgManagedGroupUpdate = fmt.Sprintf(`
resource "boundary_managed_group" "test" {
	name        = "%s"
	description = "%s"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`, managedGroupName+managedGroupUpdate, managedGroupDescription+managedGroupUpdate)
)

func TestAccManagedGroup(t *testing.T) {
	wrapper := testWrapper(t, tcRecoveryKey)
	tc := controller.NewTestController(t, append(tcConfig, controller.WithRecoveryKms(wrapper))...)

	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckManagedGroupResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfigWithRecovery(url, fooOrg, orgManagedGroup),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckManagedGroupResourceExists(provider, "boundary_managed_group.org1"),
					resource.TestCheckResourceAttr("boundary_managed_group.org1", DescriptionKey, managedGroupDescription),
					resource.TestCheckResourceAttr("boundary_managed_group.org1", NameKey, managedGroupName),
				),
			},
			importStep("boundary_managed_group.org1"),
			{
				// test update
				Config: testConfigWithRecovery(url, fooOrg, orgManagedGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckManagedGroupResourceExists(provider, "boundary_managed_group.org1"),
					resource.TestCheckResourceAttr("boundary_managed_group.org1", DescriptionKey, managedGroupDescription+managedGroupUpdate),
					resource.TestCheckResourceAttr("boundary_managed_group.org1", NameKey, managedGroupName+managedGroupUpdate),
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
		grps := managedgroups.NewClient(md.client)

		_, err := grps.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading group %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckManagedGroupResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}

		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_managed_group":
				grps := managedgroups.NewClient(md.client)

				id := rs.Primary.ID

				_, err := grps.Read(context.Background(), id)
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
