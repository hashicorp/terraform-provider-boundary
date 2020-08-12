package provider

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/api/roles"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const (
	fooRoleDescription       = "bar"
	fooRoleDescriptionUpdate = "foo bar"
)

var (
	fooRole = fmt.Sprintf(`
resource "boundary_role" "foo" {
  name = "test"
	description = "%s"
	project_id = boundary_project.foo.id
}`, fooRoleDescription)

	fooRoleUpdate = fmt.Sprintf(`
resource "boundary_role" "foo" {
  name = "test"
	description = "%s"
	project_id = boundary_project.foo.id
}`, fooRoleDescriptionUpdate)

	fooRoleWithUser = `
resource "boundary_user" "foo" {
  name = "foo"
	project_id = boundary_project.foo.id
}

resource "boundary_role" "foo" {
  name = "test"
	description = "test description"
	users = [boundary_user.foo.id]
	project_id = boundary_project.foo.id
}`

	fooRoleWithUserUpdate = `
resource "boundary_user" "foo" {
  name = "foo"
  project_id = boundary_project.foo.id
}

resource "boundary_user" "bar" {
  name = "bar"
	project_id = boundary_project.foo.id
}

resource "boundary_role" "foo" {
  name = "test"
	description = "test description"
	users = [boundary_user.foo.id, boundary_user.bar.id]
	project_id = boundary_project.foo.id
}`

	fooRoleWithGroups = `
resource "boundary_group" "foo" {
  name = "foo"
	project_id = boundary_project.foo.id
}

resource "boundary_role" "foo" {
  name = "test"
	description = "test description"
	groups = [boundary_group.foo.id]
	project_id = boundary_project.foo.id
}`

	fooRoleWithGroupsUpdate = `
resource "boundary_group" "foo" {
  name = "foo"
	project_id = boundary_project.foo.id
}

resource "boundary_group" "bar" {
  name = "bar"
	project_id = boundary_project.foo.id
}

resource "boundary_role" "foo" {
  name = "test"
	description = "test description"
	groups = [boundary_group.foo.id, boundary_group.bar.id]
	project_id = boundary_project.foo.id
}`

	readonlyGrant       = "id=*;actions=read"
	readonlyGrantUpdate = "id=*;actions=read,create"

	fooRoleWithGrants = fmt.Sprintf(`
resource "boundary_role" "foo" {
  name = "readonly"
	description = "test description"
	grants = ["%s"]
	project_id = boundary_project.foo.id
}`, readonlyGrant)

	fooRoleWithGrantsUpdate = fmt.Sprintf(`
resource "boundary_role" "foo" {
  name = "readonly"
	description = "test description"
	grants = ["%s", "%s"]
	project_id = boundary_project.foo.id
}`, readonlyGrant, readonlyGrantUpdate)
)

func TestAccRoleWithGrants(t *testing.T) {
	tc := controller.NewTestController(t, controller.WithDefaultOrgId("o_0000000000"))
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckRoleResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooProject, fooRoleWithGrants),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.foo"),
					testAccCheckRoleResourceGrantsSet("boundary_role.foo", []string{readonlyGrant}),
					resource.TestCheckResourceAttr("boundary_role.foo", "name", "readonly"),
					resource.TestCheckResourceAttr("boundary_role.foo", "description", "test description"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooProject, fooRoleWithGrantsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.foo"),
					testAccCheckRoleResourceGrantsSet("boundary_role.foo", []string{readonlyGrant, readonlyGrantUpdate}),
					resource.TestCheckResourceAttr("boundary_role.foo", "name", "readonly"),
					resource.TestCheckResourceAttr("boundary_role.foo", "description", "test description"),
				),
			},
		},
	})
}

func TestAccRoleWithUsers(t *testing.T) {
	tc := controller.NewTestController(t, controller.WithDefaultOrgId("o_0000000000"))
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckRoleResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooProject, fooRoleWithUser),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.foo"),
					testAccCheckRoleResourceUsersSet("boundary_role.foo", []string{"boundary_user.foo"}),
					testAccCheckUserResourceExists("boundary_user.foo"),
					resource.TestCheckResourceAttr("boundary_role.foo", roleDescriptionKey, "test description"),
					resource.TestCheckResourceAttr("boundary_role.foo", roleNameKey, "test"),
					resource.TestCheckResourceAttr("boundary_user.foo", "name", "foo"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooProject, fooRoleWithUserUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.foo"),
					testAccCheckUserResourceExists("boundary_user.foo"),
					testAccCheckUserResourceExists("boundary_user.bar"),
					testAccCheckRoleResourceUsersSet("boundary_role.foo", []string{"boundary_user.foo", "boundary_user.bar"}),
				),
			},
		},
	})
}

func TestAccRoleWithGroups(t *testing.T) {
	tc := controller.NewTestController(t, controller.WithDefaultOrgId("o_0000000000"))
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckRoleResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooProject, fooRoleWithGroups),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.foo"),
					testAccCheckRoleResourceGroupsSet("boundary_role.foo", []string{"boundary_group.foo"}),
					testAccCheckUserResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_role.foo", roleDescriptionKey, "test description"),
					resource.TestCheckResourceAttr("boundary_role.foo", roleNameKey, "test"),
					resource.TestCheckResourceAttr("boundary_group.foo", "name", "foo"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooProject, fooRoleWithGroupsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.foo"),
					testAccCheckUserResourceExists("boundary_group.foo"),
					testAccCheckUserResourceExists("boundary_group.bar"),
					testAccCheckRoleResourceGroupsSet("boundary_role.foo", []string{"boundary_group.foo", "boundary_group.bar"}),
				),
			},
		},
	})
}

func TestAccRole(t *testing.T) {
	tc := controller.NewTestController(t, controller.WithDefaultOrgId("o_0000000000"))
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckRoleResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooProject, fooRole),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.foo"),
					resource.TestCheckResourceAttr("boundary_role.foo", roleDescriptionKey, fooRoleDescription),
					resource.TestCheckResourceAttr("boundary_role.foo", roleNameKey, "test"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooProject, fooRoleUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.foo"),
					resource.TestCheckResourceAttr("boundary_role.foo", roleDescriptionKey, fooRoleDescriptionUpdate),
				),
			},
			{
				// test destroy
				Config: testConfig(url),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleDestroyed("boundary_role.foo"),
				),
			},
		},
	})
}

// testAccCheckRoleDestroyed checks the terraform state for the host
// catalog and returns an error if found.
//
// TODO(malnick) This method falls short of checking the Boundary API for
// the resource if the resource is not found in state. This is due to us not
// having the host catalog ID, but it doesn't guarantee that the resource was
// successfully removed.
//
// It does check Boundary if the resource is found in state to point out any
// misalignment between what is in state and the actual configuration.
func testAccCheckRoleDestroyed(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			// If it's not in state, it's destroyed in TF but not guaranteed to be destroyed
			// in Boundary. Need to find a way to get the host catalog ID here so we can
			// form a lookup to the WT API to check this.
			return nil
		}
		errs := []string{}
		errs = append(errs, fmt.Sprintf("Found role resource in state: %s", name))

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("No ID is set")
		}

		md := testProvider.Meta().(*metaData)
		projID, ok := rs.Primary.Attributes["project_id"]
		if !ok {
			return fmt.Errorf("project_id is not set")
		}
		projClient := md.client.Clone()
		projClient.SetScopeId(projID)
		rolesClient := roles.NewRolesClient(projClient)

		if _, apiErr, _ := rolesClient.Read(md.ctx, id); apiErr == nil || apiErr.Status != http.StatusNotFound {
			errs = append(errs, fmt.Sprintf("Role not destroyed %q: %v", id, apiErr))
		}

		return errors.New(strings.Join(errs, ","))
	}
}

func testAccCheckRoleResourceExists(name string) resource.TestCheckFunc {
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
		projID, ok := rs.Primary.Attributes["project_id"]
		if !ok {
			return fmt.Errorf("project_id is not set")
		}
		projClient := md.client.Clone()
		projClient.SetScopeId(projID)
		rolesClient := roles.NewRolesClient(projClient)

		if _, _, err := rolesClient.Read(md.ctx, id); err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckRoleResourceUsersSet(name string, users []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("role resource not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("role resource ID is not set")
		}

		userIDs := []string{}
		for _, userResourceName := range users {
			ur, ok := s.RootModule().Resources[userResourceName]
			if !ok {
				return fmt.Errorf("user resource not found: %s", userResourceName)
			}

			userID := ur.Primary.ID
			if id == "" {
				return fmt.Errorf("user resource ID not set")
			}

			userIDs = append(userIDs, userID)
		}

		md := testProvider.Meta().(*metaData)
		projID, ok := rs.Primary.Attributes["project_id"]
		if !ok {
			return fmt.Errorf("project_id is not set")
		}
		projClient := md.client.Clone()
		projClient.SetScopeId(projID)
		rolesClient := roles.NewRolesClient(projClient)

		r, _, err := rolesClient.Read(md.ctx, id)
		if err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		// for every user set as a principal on the role in the state, ensure
		// each role in boundary has the same setings
		if len(r.PrincipalIds) == 0 {
			return fmt.Errorf("no users found in boundary")
		}

		for _, stateUser := range r.PrincipalIds {
			ok := false
			for _, gotUser := range userIDs {
				if gotUser == stateUser {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("user in state not set in boundary: %s", stateUser)
			}
		}

		return nil
	}
}

func testAccCheckRoleResourceGroupsSet(name string, groups []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("role resource not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("role resource ID is not set")
		}

		groupIDs := []string{}
		for _, groupResourceName := range groups {
			gr, ok := s.RootModule().Resources[groupResourceName]
			if !ok {
				return fmt.Errorf("group resource not found: %s", groupResourceName)
			}

			groupID := gr.Primary.ID
			if id == "" {
				return fmt.Errorf("group resource ID not set")
			}

			groupIDs = append(groupIDs, groupID)
		}

		md := testProvider.Meta().(*metaData)
		projID, ok := rs.Primary.Attributes["project_id"]
		if !ok {
			return fmt.Errorf("project_id is not set")
		}
		projClient := md.client.Clone()
		projClient.SetScopeId(projID)
		rolesClient := roles.NewRolesClient(projClient)

		r, _, err := rolesClient.Read(md.ctx, id)
		if err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		// for every user set as a principal on the role in the state, ensure
		// each role in boundary has the same setings
		if len(r.PrincipalIds) == 0 {
			return fmt.Errorf("no groups found in boundary")
		}

		for _, stateGroup := range r.PrincipalIds {
			ok := false
			for _, gotGroup := range groupIDs {
				if gotGroup == stateGroup {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("group in state not set in boundary: %s", stateGroup)
			}
		}

		return nil
	}
}

func testAccCheckRoleResourceGrantsSet(name string, expectedGrants []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("role resource not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("role resource ID is not set")
		}

		md := testProvider.Meta().(*metaData)
		projID, ok := rs.Primary.Attributes["project_id"]
		if !ok {
			return fmt.Errorf("project_id is not set")
		}
		projClient := md.client.Clone()
		projClient.SetScopeId(projID)
		rolesClient := roles.NewRolesClient(projClient)

		r, _, err := rolesClient.Read(md.ctx, id)
		if err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		// for each expected grant, ensure they're set on the role
		if len(r.Grants) == 0 {
			return fmt.Errorf("no grants found on role, %+v\n", r)
		}

		for _, grant := range expectedGrants {
			ok = false
			for _, gotGrant := range r.Grants {
				if gotGrant.Raw == grant {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("expected grant not found on role, %s: %s\n  Have: %v\n", r.Name, grant, r)
			}
		}

		return nil
	}
}

func testAccCheckRoleResourceDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_user":
				continue
			case "boundary_group":
				continue
			case "boundary_role":

				id := rs.Primary.ID
				projID, ok := rs.Primary.Attributes["project_id"]
				if !ok {
					return fmt.Errorf("project_id is not set")
				}
				projClient := md.client.Clone()
				projClient.SetScopeId(projID)
				rolesClient := roles.NewRolesClient(projClient)

				_, apiErr, _ := rolesClient.Read(md.ctx, id)
				if apiErr == nil || apiErr.Status != http.StatusNotFound {
					return fmt.Errorf("Didn't get a 404 when reading destroyed role %q: %v", id, apiErr)
				}

			default:
				t.Logf("Got unknown resource type %q", rs.Type)
			}
		}
		return nil
	}
}
