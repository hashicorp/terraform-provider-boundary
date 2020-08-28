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
  name        = "test"
	description = "%s"
	scope_id    = boundary_organization.foo.id
}`, fooGroupDescription)

	orgGroupUpdate = fmt.Sprintf(`
resource "boundary_group" "foo" {
  name        = "test"
	description = "%s"
  scope_id    = boundary_organization.foo.id
}`, fooGroupDescriptionUpdate)

	orgGroupWithMembers = `
resource "boundary_user" "foo" {
  description = "foo"
  scope_id    = boundary_organization.foo.id
}

resource "boundary_group" "with_members" {
	description = "with members"
	member_ids  = [boundary_user.foo.id]
  scope_id    = boundary_organization.foo.id
}`

	orgGroupWithMembersUpdate = `
resource "boundary_user" "foo" {
  description = "foo"
  scope_id    = boundary_organization.foo.id
}

resource "boundary_user" "bar" {
  description = "bar"
  scope_id    = boundary_organization.foo.id
}

resource "boundary_group" "with_members" {
	description = "with members"
	member_ids  = [boundary_user.foo.id, boundary_user.bar.id]
  scope_id    = boundary_organization.foo.id
}`

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
				Config: testConfig(url, fooOrg, orgGroup),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescription),
					resource.TestCheckResourceAttr("boundary_group.foo", groupNameKey, "test"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooOrg, orgGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescriptionUpdate),
				),
			},
			{
				// test update to project scope
				Config: testConfig(url, fooOrg, fooProject, orgToProjectGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescriptionUpdate),
					testAccCheckGroupProjectScope("boundary_group.foo"),
				),
			},
			{
				// test create
				Config: testConfig(url, fooOrg, fooProject, projGroup),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescription),
					resource.TestCheckResourceAttr("boundary_group.foo", groupNameKey, "test"),
					testAccCheckGroupProjectScope("boundary_group.foo"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooOrg, fooProject, projGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescriptionUpdate),
					testAccCheckGroupProjectScope("boundary_group.foo"),
				),
			},
			{
				// test update to org scope
				Config: testConfig(url, fooOrg, fooProject, projToOrgGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_group.foo", groupDescriptionKey, fooGroupDescriptionUpdate),
					resource.TestCheckResourceAttr("boundary_group.foo", groupScopeIDKey, tcOrg),
				),
			},
		},
	})
}

func TestAccGroupWithMembers(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)

	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckGroupResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, orgGroupWithMembers),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.with_members"),
					testAccCheckGroupResourceExists("boundary_user.foo"),
					resource.TestCheckResourceAttr("boundary_group.with_members", groupDescriptionKey, "with members"),
					testAccCheckGroupResourceMembersSet("boundary_group.with_members", []string{"boundary_user.foo"}),
				),
			},
			{
				// test update
				Config: testConfig(url, fooOrg, orgGroupWithMembersUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.with_members"),
					testAccCheckGroupResourceExists("boundary_user.foo"),
					resource.TestCheckResourceAttr("boundary_group.with_members", groupDescriptionKey, "with members"),
					testAccCheckGroupResourceMembersSet("boundary_group.with_members", []string{"boundary_user.foo", "boundary_user.bar"}),
				),
			},
		},
	})
}

func testAccCheckGroupResourceMembersSet(name string, members []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("role resource not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("role resource ID is not set")
		}

		// ensure users are declared in state
		memberIDs := []string{}
		for _, userResourceName := range members {
			ur, ok := s.RootModule().Resources[userResourceName]
			if !ok {
				return fmt.Errorf("user resource not found: %s", userResourceName)
			}

			memberID := ur.Primary.ID
			if id == "" {
				return fmt.Errorf("principal resource ID not set")
			}

			memberIDs = append(memberIDs, memberID)
		}

		// check boundary to ensure it matches
		md := testProvider.Meta().(*metaData)
		client := md.client.Clone()

		projID, ok := rs.Primary.Attributes["scope_id"]
		if ok {
			client.SetScopeId(projID)
		}
		grpsClient := groups.NewClient(client)

		g, _, err := grpsClient.Read(md.ctx, id)
		if err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		// for every member set as a member on the group in the state, ensure
		// each group in boundary has the same setings
		if len(g.MemberIds) == 0 {
			return fmt.Errorf("no members found on group")
		}

		for _, stateMember := range g.MemberIds {
			ok := false
			for _, gotMember := range memberIDs {
				if gotMember == stateMember {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("member in state not set in boundary: %s", stateMember)
			}
		}

		return nil
	}
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
		grps := groups.NewClient(projClient)

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
		grps := groups.NewClient(projClient)

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
			case "boundary_organization":
				continue
			case "boundary_project":
				continue
			case "boundary_user":
				continue
			case "boundary_group":
				projClient := md.client.Clone()
				projID, ok := rs.Primary.Attributes["scope_id"]
				if ok {
					projClient.SetScopeId(projID)
				}
				grps := groups.NewClient(projClient)

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
