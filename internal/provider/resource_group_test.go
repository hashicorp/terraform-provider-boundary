package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/api/groups"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	fooGroupDescription       = "bar"
	fooGroupDescriptionUpdate = "foo bar"
)

var (
	orgGroup = fmt.Sprintf(`
resource "boundary_group" "org1" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.org1.id
}`, fooGroupDescription)

	orgGroupUpdate = fmt.Sprintf(`
resource "boundary_group" "org1" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.org1.id
}`, fooGroupDescriptionUpdate)

	orgGroupWithMembers = `
resource "boundary_user" "org1" {
	description = "org1"
	scope_id    = boundary_scope.org1.id
}

resource "boundary_group" "with_members" {
	description = "with members"
	member_ids  = [boundary_user.org1.id]
	scope_id    = boundary_scope.org1.id
}`

	orgGroupWithMembersUpdate = `
resource "boundary_user" "org1" {
	description = "org1"
	scope_id    = boundary_scope.org1.id
}

resource "boundary_user" "bar" {
	description = "bar"
	scope_id    = boundary_scope.org1.id
}

resource "boundary_group" "with_members" {
	description = "with members"
	member_ids  = [boundary_user.org1.id, boundary_user.bar.id]
	scope_id    = boundary_scope.org1.id
}`

	orgToProjectGroupUpdate = `
resource "boundary_group" "org1" {
	name = "test-to-proj"
	description = "org1-test-to-proj"
	scope_id = boundary_scope.proj1.id
}`

	projGroup = `
resource "boundary_group" "proj1" {
	name = "test-proj"
	description = "desc-test-proj"
	scope_id = boundary_scope.proj1.id
}`

	projGroupUpdate = `
resource "boundary_group" "proj1" {
	name = "test-proj-up"
	description = "desc-test-proj-up"
	scope_id = boundary_scope.proj1.id
}`

	projToOrgGroupUpdate = `
resource "boundary_group" "proj1" {
	name = "test-back"
	description = "desc-back"
	scope_id = boundary_scope.org1.id
}`
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
					testAccCheckGroupResourceExists("boundary_group.org1"),
					resource.TestCheckResourceAttr("boundary_group.org1", groupDescriptionKey, fooGroupDescription),
					resource.TestCheckResourceAttr("boundary_group.org1", groupNameKey, "test"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooOrg, orgGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.org1"),
					resource.TestCheckResourceAttr("boundary_group.org1", groupDescriptionKey, fooGroupDescriptionUpdate),
				),
			},
			{
				// test update to project scope
				Config: testConfig(url, fooOrg, firstProjectFoo, orgToProjectGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.org1"),
					resource.TestCheckResourceAttr("boundary_group.org1", groupDescriptionKey, "org1-test-to-proj"),
					testAccCheckGroupScope("boundary_group.org1", "p_"),
				),
			},
			{
				// test create
				Config: testConfig(url, fooOrg, firstProjectFoo, projGroup),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.proj1"),
					resource.TestCheckResourceAttr("boundary_group.proj1", groupDescriptionKey, "desc-test-proj"),
					resource.TestCheckResourceAttr("boundary_group.proj1", groupNameKey, "test-proj"),
					testAccCheckGroupScope("boundary_group.proj1", "p_"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, projGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.proj1"),
					resource.TestCheckResourceAttr("boundary_group.proj1", groupDescriptionKey, "desc-test-proj-up"),
					testAccCheckGroupScope("boundary_group.proj1", "p_"),
				),
			},
			{
				// test update to org scope
				Config: testConfig(url, fooOrg, firstProjectFoo, projToOrgGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.proj1"),
					resource.TestCheckResourceAttr("boundary_group.proj1", groupDescriptionKey, "desc-back"),
					testAccCheckGroupScope("boundary_group.proj1", "o_"),
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
					testAccCheckUserResourceExists("boundary_user.org1"),
					resource.TestCheckResourceAttr("boundary_group.with_members", groupDescriptionKey, "with members"),
					testAccCheckGroupResourceMembersSet("boundary_group.with_members", []string{"boundary_user.org1"}),
				),
			},
			{
				// test update
				Config: testConfig(url, fooOrg, orgGroupWithMembersUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists("boundary_group.with_members"),
					testAccCheckUserResourceExists("boundary_user.org1"),
					resource.TestCheckResourceAttr("boundary_group.with_members", groupDescriptionKey, "with members"),
					testAccCheckGroupResourceMembersSet("boundary_group.with_members", []string{"boundary_user.org1", "boundary_user.bar"}),
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
		grpsClient := groups.NewClient(md.client)

		g, apiErr, err := grpsClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading group %q: %v", id, err)
		}
		if apiErr != nil {
			return fmt.Errorf("Got an API error when reading group %q: %v", id, apiErr.Message)
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

func testAccCheckGroupScope(name, prefix string) resource.TestCheckFunc {
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
		grps := groups.NewClient(md.client)

		g, apiErr, err := grps.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("could not read resource state %q: %v", id, err)
		}
		if apiErr != nil {
			return fmt.Errorf("Got an API error when reading group %q: %v", id, apiErr.Message)
		}

		if !strings.HasPrefix(g.ScopeId, prefix) {
			return fmt.Errorf("Scope ID in state does not have prefix: %s != %s", g.ScopeId, prefix)
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
		grps := groups.NewClient(md.client)

		_, apiErr, err := grps.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading group %q: %v", id, err)
		}
		if apiErr != nil {
			return fmt.Errorf("Got an API error when reading group %q: %v", id, apiErr.Message)
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
			case "boundary_scope":
				continue
			case "boundary_user":
				continue
			case "boundary_group":
				grps := groups.NewClient(md.client)

				id := rs.Primary.ID

				_, apiErr, err := grps.Read(context.Background(), id)
				if err != nil {
					return err
				}
				if apiErr == nil || apiErr.Status != http.StatusNotFound {
					return fmt.Errorf("Didn't get a 404 when reading destroyed resource %q: %v", id, apiErr)
				}

			default:
				t.Logf("Got unknown resource type %q", rs.Type)
			}
		}
		return nil
	}
}