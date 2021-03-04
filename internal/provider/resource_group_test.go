package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/groups"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
	depends_on = [boundary_role.org1_admin]
}`, fooGroupDescription)

	orgGroupUpdate = fmt.Sprintf(`
resource "boundary_group" "org1" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`, fooGroupDescriptionUpdate)

	orgGroupWithMembers = `
resource "boundary_user" "org1" {
	description = "org1"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}

resource "boundary_group" "with_members" {
	description = "with members"
	member_ids  = [boundary_user.org1.id]
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`

	orgGroupWithMembersUpdate = `
resource "boundary_user" "org1" {
	description = "org1"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}

resource "boundary_user" "bar" {
	description = "bar"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}

resource "boundary_group" "with_members" {
	description = "with members"
	member_ids  = [boundary_user.org1.id, boundary_user.bar.id]
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`

	orgToProjectGroupUpdate = `
resource "boundary_group" "org1" {
	name = "test-to-proj"
	description = "org1-test-to-proj"
	scope_id = boundary_scope.proj1.id
	depends_on = [boundary_role.org1_admin]
}`

	projGroup = `
resource "boundary_group" "proj1" {
	name = "test-proj"
	description = "desc-test-proj"
	scope_id = boundary_scope.proj1.id
	depends_on = [boundary_role.org1_admin]
}`

	projGroupUpdate = `
resource "boundary_group" "proj1" {
	name = "test-proj-up"
	description = "desc-test-proj-up"
	scope_id = boundary_scope.proj1.id
	depends_on = [boundary_role.org1_admin]
}`

	projNameRemoval = `
resource "boundary_group" "proj1" {
	name = ""
	description = "no-name"
	scope_id = boundary_scope.proj1.id
	depends_on = [boundary_role.org1_admin]
}`

	projToOrgGroupUpdate = `
resource "boundary_group" "proj1" {
	name = "test-back"
	description = "desc-back"
	scope_id = boundary_scope.org1.id
	depends_on = [boundary_role.org1_admin]
}`
)

// NOTE: this test also tests out the recovery KMS mechanism.
func TestAccGroup(t *testing.T) {
	wrapper := testWrapper(t, tcRecoveryKey)
	tc := controller.NewTestController(t, append(tcConfig, controller.WithRecoveryKms(wrapper))...)

	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckGroupResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfigWithRecovery(url, fooOrg, orgGroup),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists(provider, "boundary_group.org1"),
					resource.TestCheckResourceAttr("boundary_group.org1", DescriptionKey, fooGroupDescription),
					resource.TestCheckResourceAttr("boundary_group.org1", NameKey, "test"),
				),
			},
			importStep("boundary_group.org1"),
			{
				// test update
				Config: testConfigWithRecovery(url, fooOrg, orgGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists(provider, "boundary_group.org1"),
					resource.TestCheckResourceAttr("boundary_group.org1", DescriptionKey, fooGroupDescriptionUpdate),
				),
			},
			importStep("boundary_group.org1"),
			{
				// test update to project scope
				Config: testConfigWithRecovery(url, fooOrg, firstProjectFoo, orgToProjectGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists(provider, "boundary_group.org1"),
					resource.TestCheckResourceAttr("boundary_group.org1", DescriptionKey, "org1-test-to-proj"),
					testAccCheckGroupScope(provider, "boundary_group.org1", "p_"),
				),
			},
			importStep("boundary_group.org1"),
			{
				// test create
				Config: testConfigWithRecovery(url, fooOrg, firstProjectFoo, projGroup),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists(provider, "boundary_group.proj1"),
					resource.TestCheckResourceAttr("boundary_group.proj1", DescriptionKey, "desc-test-proj"),
					resource.TestCheckResourceAttr("boundary_group.proj1", NameKey, "test-proj"),
					testAccCheckGroupScope(provider, "boundary_group.proj1", "p_"),
				),
			},
			importStep("boundary_group.proj1"),
			{
				// test update
				Config: testConfigWithRecovery(url, fooOrg, firstProjectFoo, projGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists(provider, "boundary_group.proj1"),
					resource.TestCheckResourceAttr("boundary_group.proj1", DescriptionKey, "desc-test-proj-up"),
					testAccCheckGroupScope(provider, "boundary_group.proj1", "p_"),
				),
			},
			importStep("boundary_group.proj1"),
			{
				// test name removal
				Config: testConfigWithRecovery(url, fooOrg, firstProjectFoo, projNameRemoval),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists(provider, "boundary_group.proj1"),
					resource.TestCheckResourceAttr("boundary_group.proj1", NameKey, ""),
					testAccCheckGroupScope(provider, "boundary_group.proj1", "p_"),
				),
			},
			importStep("boundary_group.proj1"),
			{
				// test update to org scope
				Config: testConfigWithRecovery(url, fooOrg, firstProjectFoo, projToOrgGroupUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists(provider, "boundary_group.proj1"),
					resource.TestCheckResourceAttr("boundary_group.proj1", DescriptionKey, "desc-back"),
					testAccCheckGroupScope(provider, "boundary_group.proj1", "o_"),
				),
			},
			importStep("boundary_group.proj1"),
		},
	})
}

func TestAccGroupWithMembers(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)

	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),

		CheckDestroy: testAccCheckGroupResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, orgGroupWithMembers),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists(provider, "boundary_group.with_members"),
					testAccCheckUserResourceExists(provider, "boundary_user.org1"),
					resource.TestCheckResourceAttr("boundary_group.with_members", DescriptionKey, "with members"),
					testAccCheckGroupResourceMembersSet(provider, "boundary_group.with_members", []string{"boundary_user.org1"}),
				),
			},
			importStep("boundary_group.with_members"),
			{
				// test update
				Config: testConfig(url, fooOrg, orgGroupWithMembersUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupResourceExists(provider, "boundary_group.with_members"),
					testAccCheckUserResourceExists(provider, "boundary_user.org1"),
					resource.TestCheckResourceAttr("boundary_group.with_members", DescriptionKey, "with members"),
					testAccCheckGroupResourceMembersSet(provider, "boundary_group.with_members", []string{"boundary_user.org1", "boundary_user.bar"}),
				),
			},
			importStep("boundary_group.with_members"),
			importStep("boundary_user.org1"),
		},
	})
}

func testAccCheckGroupResourceMembersSet(testProvider *schema.Provider, name string, members []string) resource.TestCheckFunc {
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

		g, err := grpsClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading group %q: %v", id, err)
		}

		// for every member set as a member on the group in the state, ensure
		// each group in boundary has the same setings
		if len(g.Item.MemberIds) == 0 {
			return fmt.Errorf("no members found on group")
		}

		for _, stateMember := range g.Item.MemberIds {
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

func testAccCheckGroupScope(testProvider *schema.Provider, name, prefix string) resource.TestCheckFunc {
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

		g, err := grps.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("could not read resource state %q: %v", id, err)
		}

		if !strings.HasPrefix(g.Item.ScopeId, prefix) {
			return fmt.Errorf("Scope ID in state does not have prefix: %s != %s", g.Item.ScopeId, prefix)
		}

		return nil
	}
}

func testAccCheckGroupResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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

		_, err := grps.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading group %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckGroupResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}

		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_group":
				grps := groups.NewClient(md.client)

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
