package provider

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/api/groups"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const (
	fooGroupDescription       = "bar"
	fooGroupDescriptionUpdate = "foo bar"
)

var (
	orgGroup = fmt.Sprintf(`
resource "boundary_group" "foo" {
  name = "test"
	description = "%s"
}`, fooGroupDescription)

	orgGroupUpdate = fmt.Sprintf(`
resource "boundary_group" "foo" {
  name = "test"
	description = "%s"
}`, fooGroupDescriptionUpdate)

	projGroup = fmt.Sprintf(`
resource "boundary_group" "foo" {
  name = "test"
	description = "%s"
	project_id = boundary_project.foo.id
}`, fooGroupDescription)

	projGroupUpdate = fmt.Sprintf(`
resource "boundary_group" "foo" {
  name = "test"
	description = "%s"
	project_id = boundary_project.foo.id
}`, fooGroupDescriptionUpdate)
)

func TestAccGroup(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)

	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckGroupResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooProject, projGroup),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescription),
					resource.TestCheckResourceAttr("boundary_group.foo", groupNameKey, "test"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooProject, projGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescriptionUpdate),
				),
			},
			{
				// test create
				Config: testConfig(url, orgGroup),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescription),
					resource.TestCheckResourceAttr("boundary_group.foo", groupNameKey, "test"),
				),
			},
			{
				// test update
				Config: testConfig(url, orgGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescriptionUpdate),
				),
			},
			{
				// test destroy
				Config: testConfig(url),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupDestroyed("boundary_group.foo"),
				),
			},
		},
	})
}

// testAccCheckGroupDestroyed checks the terraform state for the host
// catalog and returns an error if found.
//
// TODO(malnick) This method falls short of checking the Boundary API for
// the resource if the resource is not found in state. This is due to us not
// having the host catalog ID, but it doesn't guarantee that the resource was
// successfully removed.
//
// It does check Boundary if the resource is found in state to point out any
// misalignment between what is in state and the actual configuration.
func testAccCheckGroupDestroyed(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			// If it's not in state, it's destroyed in TF but not guaranteed to be destroyed
			// in Boundary. Need to find a way to get the host catalog ID here so we can
			// form a lookup to the WT API to check this.
			return nil
		}
		errs := []string{}
		errs = append(errs, fmt.Sprintf("Found group resource in state: %s", name))

		expectedGroupID := rs.Primary.ID
		if expectedGroupID == "" {
			return fmt.Errorf("No ID is set")
		}

		md := testProvider.Meta().(*metaData)
		projID, ok := rs.Primary.Attributes["project_id"]
		projClient := md.client.Clone()
		if ok {
			projClient.SetScopeId(projID)
		}
		grps := groups.NewGroupsClient(projClient)

		if _, apiErr, _ := grps.Read(md.ctx, expectedGroupID); apiErr == nil || apiErr.Status != http.StatusNotFound {
			errs = append(errs, fmt.Sprintf("Group not destroyed %q: %v", expectedGroupID, apiErr))
		}

		return errors.New(strings.Join(errs, ","))
	}
}

func testAccCheckGroupResourceExists(name string) resource.TestCheckFunc {
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
		projClient := md.client.Clone()
		projID, ok := rs.Primary.Attributes["project_id"]
		if ok {
			projClient.SetScopeId(projID)
		}
		grps := groups.NewGroupsClient(projClient)

		if _, _, err := grps.Read(md.ctx, id); err != nil {
			return fmt.Errorf("Got an error when reading group %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckGroupResourceDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}

		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_project":
				continue
			case "boundary_group":
				projClient := md.client.Clone()
				projID, ok := rs.Primary.Attributes["project_id"]
				if ok {
					projClient.SetScopeId(projID)
				}
				grps := groups.NewGroupsClient(projClient)

				id := rs.Primary.ID

				_, apiErr, _ := grps.Read(md.ctx, id)
				if apiErr == nil || apiErr.Status != http.StatusNotFound {
					return fmt.Errorf("Didn't get a 404 when reading destroyed group %q: %v", id, apiErr)
				}

			default:
				t.Logf("Got unknown resource type %q", rs.Type)
			}
		}
		return nil
	}
}
