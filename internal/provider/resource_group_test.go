package provider

import (
	"fmt"
	"net/http"
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

	orgToProjectGroupUpdate = fmt.Sprintf(`
resource "boundary_group" "foo" {
  name = "test"
	description = "%s"
	scope_id = boundary_project.foo.id
}`, fooGroupDescriptionUpdate)

	projGroup = fmt.Sprintf(`
resource "boundary_group" "foo" {
  name = "test"
	description = "%s"
	scope_id = boundary_project.foo.id
}`, fooGroupDescription)

	projGroupUpdate = fmt.Sprintf(`
resource "boundary_group" "foo" {
  name = "test"
	description = "%s"
	scope_id = boundary_project.foo.id
}`, fooGroupDescriptionUpdate)

	// TODO When removing the scope_id the provider does not revert back to the provider
	// default scope. As a workaround, you can move the resource into the org scope by
	// manually setting the org scope ID as the scope_id field.
	projToOrgGroupUpdate = fmt.Sprintf(`
resource "boundary_group" "foo" {
  name = "test"
	description = "%s"
	scope_id = "%s"
}`, fooGroupDescriptionUpdate, tcOrg)
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
				Config: testConfig(url, orgGroup),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescription),
					resource.TestCheckResourceAttr("boundary_group.foo", groupNameKey, "test"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupScopeIDKey, tcOrg),
				),
			},
			{
				// test update
				Config: testConfig(url, orgGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescriptionUpdate),
					resource.TestCheckResourceAttr("boundary_group.foo", groupScopeIDKey, tcOrg),
				),
			},
			{
				// test update to project scope
				Config: testConfig(url, fooProject, orgToProjectGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescriptionUpdate),
					testAccCheckGroupProjectScope("boundary_group.foo"),
				),
			},
			{
				// test create
				Config: testConfig(url, fooProject, projGroup),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescription),
					resource.TestCheckResourceAttr("boundary_group.foo", groupNameKey, "test"),
					testAccCheckGroupProjectScope("boundary_group.foo"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooProject, projGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescriptionUpdate),
					testAccCheckGroupProjectScope("boundary_group.foo"),
				),
			},
			{
				// test update to org scope
				Config: testConfig(url, fooProject, projToOrgGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescriptionUpdate),
					resource.TestCheckResourceAttr("boundary_group.foo", groupScopeIDKey, tcOrg),
				),
			},
		},
	})
}

func testAccCheckGroupProjectScope(name string) resource.TestCheckFunc {
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

		stateProjID, ok := rs.Primary.Attributes["scope_id"]
		if ok {
			projClient.SetScopeId(stateProjID)
		}
		grps := groups.NewGroupsClient(projClient)

		g, _, err := grps.Read(md.ctx, id)
		if err != nil {
			return fmt.Errorf("could not read resource state %q: %v", id, err)
		}

		if g.Scope.Id != stateProjID {
			return fmt.Errorf("project ID in state does not match boundary state: %s != %s", g.Scope.Id, stateProjID)
		}

		return nil
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
		projID, ok := rs.Primary.Attributes["scope_id"]
		if ok && projID != "" {
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
		fmt.Printf("test check group resource destroyed\n")
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
				projID, ok := rs.Primary.Attributes["scope_id"]
				if ok {
					projClient.SetScopeId(projID)
				}
				grps := groups.NewGroupsClient(projClient)

				id := rs.Primary.ID

				_, apiErr, _ := grps.Read(md.ctx, id)
				if apiErr == nil || apiErr.Status != http.StatusForbidden && apiErr.Status != http.StatusNotFound {
					return fmt.Errorf("Didn't get a 403 or 404 when reading destroyed resource %q: %v", id, apiErr)
				}

			default:
				t.Logf("Got unknown resource type %q", rs.Type)
			}
		}
		return nil
	}
}
