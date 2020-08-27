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
	orgRole = fmt.Sprintf(`
resource "boundary_role" "foo" {
  name        = "test"
	description = "%s"
	scope_id    = boundary_organization.foo.id
}`, fooRoleDescription)

	orgRoleUpdate = fmt.Sprintf(`
resource "boundary_role" "foo" {
  name        = "test"
	description = "%s"
  scope_id    = boundary_organization.foo.id
}`, fooRoleDescriptionUpdate)

	ProjToOrgRole = fmt.Sprintf(`
resource "boundary_role" "foo" {
  name        = "test"
	description = "%s"
	scope_id    = "%s"
}`, fooRoleDescription, tcOrg)

	projRole = fmt.Sprintf(`
resource "boundary_role" "foo" {
  name        = "test"
	description = "%s"
	scope_id    = boundary_project.foo.id
}`, fooRoleDescription)

	projRoleUpdate = fmt.Sprintf(`
resource "boundary_role" "foo" {
  name        = "test"
	description = "%s"
	scope_id    = boundary_project.foo.id
}`, fooRoleDescriptionUpdate)

	projRoleWithPrincipal = `
resource "boundary_user" "foo" {
  name     = "foo"
}

resource "boundary_role" "with_principal" {
  name        = "with_principal"
	description = "with principal"
	principals  = [boundary_user.foo.id]
	scope_id    = boundary_project.foo.id
}`

	projRoleWithPrincipalUpdate = `
resource "boundary_user" "foo" {
  name     = "foo"
}

resource "boundary_user" "bar" {
  name     = "bar"
}

resource "boundary_role" "with_principal" {
  name        = "with_principal"
	description = "with principal"
	principals  = [boundary_user.foo.id, boundary_user.bar.id]
	scope_id    = boundary_project.foo.id
}`

	projRoleWithGroups = `
resource "boundary_group" "foo" {
  name     = "foo"
	scope_id = boundary_project.foo.id
}

resource "boundary_role" "with_groups" {
  name        = "with_groups"
	description = "with groups"
	principals  = [boundary_group.foo.id]
	scope_id    = boundary_project.foo.id
}`

	projRoleWithGroupsUpdate = `
resource "boundary_group" "foo" {
  name     = "foo"
	scope_id = boundary_project.foo.id
}

resource "boundary_group" "bar" {
  name     = "bar"
	scope_id = boundary_project.foo.id
}

resource "boundary_role" "with_groups" {
  name        = "with_groups"
	description = "with groups"
	principals  = [boundary_group.foo.id, boundary_group.bar.id]
	scope_id    = boundary_project.foo.id
}`

	readonlyGrant       = "id=*;actions=read"
	readonlyGrantUpdate = "id=*;actions=read,create"

	projRoleWithGrants = fmt.Sprintf(`
resource "boundary_role" "with_grants" {
  name        = "with_grants"
	description = "with grants"
	grants      = ["%s"]
	scope_id    = boundary_project.foo.id
}`, readonlyGrant)

	projRoleWithGrantsUpdate = fmt.Sprintf(`
resource "boundary_role" "with_grants" {
  name        = "with_grants"
	description = "with grants"
	grants      = ["%s", "%s"]
	scope_id    = boundary_project.foo.id
}`, readonlyGrant, readonlyGrantUpdate)
)

func TestAccRoleToOrgToProject(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckRoleResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test org role create
				Config: testConfig(url, fooOrg, orgRole),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.foo"),
					resource.TestCheckResourceAttr("boundary_role.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_role.foo", "description", fooRoleDescription),
				),
			},
			{
				// test org role update
				Config: testConfig(url, fooOrg, orgRoleUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.foo"),
					resource.TestCheckResourceAttr("boundary_role.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_role.foo", "description", fooRoleDescriptionUpdate),
				),
			},
			{
				// test org to project role create
				Config: testConfig(url, fooOrg, fooProject, projRole),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.foo"),
					resource.TestCheckResourceAttr("boundary_role.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_role.foo", "description", fooRoleDescription),
				),
			},
			{
				// test project role update
				Config: testConfig(url, fooOrg, fooProject, projRoleUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.foo"),
					resource.TestCheckResourceAttr("boundary_role.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_role.foo", "description", fooRoleDescriptionUpdate),
				),
			},
		},
	})
}

func TestAccRoleWithGrants(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckRoleResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test project role create with grants
				Config: testConfig(url, fooOrg, fooProject, projRoleWithGrants),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.with_grants"),
					testAccCheckRoleResourceGrantsSet("boundary_role.with_grants", []string{readonlyGrant}),
					resource.TestCheckResourceAttr("boundary_role.with_grants", "name", "with_grants"),
					resource.TestCheckResourceAttr("boundary_role.with_grants", "description", "with grants"),
				),
			},
			{
				// test project role update with grants
				Config: testConfig(url, fooOrg, fooProject, projRoleWithGrantsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.with_grants"),

					testAccCheckRoleResourceGrantsSet("boundary_role.with_grants", []string{readonlyGrant, readonlyGrantUpdate}),
					resource.TestCheckResourceAttr("boundary_role.with_grants", "name", "with_grants"),
					resource.TestCheckResourceAttr("boundary_role.with_grants", "description", "with grants"),
				),
			},
		},
	})
}

func TestAccRoleWithPrincipals(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckRoleResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, fooProject, projRoleWithPrincipal),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.with_principal"),
					testAccCheckRoleResourcePrincipalsSet("boundary_role.with_principal", []string{"boundary_user.foo"}),
					testAccCheckUserResourceExists("boundary_user.foo"),
					resource.TestCheckResourceAttr("boundary_role.with_principal", roleDescriptionKey, "with principal"),
					resource.TestCheckResourceAttr("boundary_role.with_principal", roleNameKey, "with_principal"),
					resource.TestCheckResourceAttr("boundary_user.foo", "name", "foo"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooOrg, fooProject, projRoleWithPrincipalUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.with_principal"),
					testAccCheckUserResourceExists("boundary_user.foo"),
					testAccCheckUserResourceExists("boundary_user.bar"),
					testAccCheckRoleResourcePrincipalsSet("boundary_role.with_principal", []string{"boundary_user.foo", "boundary_user.bar"}),
				),
			},
		},
	})
}

func TestAccRoleWithGroups(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckRoleResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, fooProject, projRoleWithGroups),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.with_groups"),
					testAccCheckRoleResourceGroupsSet("boundary_role.with_groups", []string{"boundary_group.foo"}),
					testAccCheckUserResourceExists("boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_role.with_groups", roleDescriptionKey, "with groups"),
					resource.TestCheckResourceAttr("boundary_role.with_groups", roleNameKey, "with_groups"),
					resource.TestCheckResourceAttr("boundary_group.foo", "name", "foo"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooOrg, fooProject, projRoleWithGroupsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("boundary_role.with_groups"),
					testAccCheckUserResourceExists("boundary_group.foo"),
					testAccCheckUserResourceExists("boundary_group.bar"),
					testAccCheckRoleResourceGroupsSet("boundary_role.with_groups", []string{"boundary_group.foo", "boundary_group.bar"}),
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
		projID, ok := rs.Primary.Attributes["scope_id"]
		if !ok {
			return fmt.Errorf("scope_id is not set")
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
		projID, ok := rs.Primary.Attributes["scope_id"]
		if !ok {
			return fmt.Errorf("scope_id is not set")
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

func testAccCheckRoleResourcePrincipalsSet(name string, principals []string) resource.TestCheckFunc {
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
		for _, userResourceName := range principals {
			ur, ok := s.RootModule().Resources[userResourceName]
			if !ok {
				return fmt.Errorf("user resource not found: %s", userResourceName)
			}

			userID := ur.Primary.ID
			if id == "" {
				return fmt.Errorf("principal resource ID not set")
			}

			userIDs = append(userIDs, userID)
		}

		md := testProvider.Meta().(*metaData)
		projID, ok := rs.Primary.Attributes["scope_id"]
		if !ok {
			return fmt.Errorf("scope_id is not set")
		}
		projClient := md.client.Clone()
		projClient.SetScopeId(projID)
		rolesClient := roles.NewRolesClient(projClient)

		r, _, err := rolesClient.Read(md.ctx, id)
		if err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		// for every principal set as a principal on the role in the state, ensure
		// each role in boundary has the same setings
		if len(r.PrincipalIds) == 0 {
			return fmt.Errorf("no principals found in boundary")
		}

		for _, statePrincipal := range r.PrincipalIds {
			ok := false
			for _, gotPrincipal := range userIDs {
				if gotPrincipal == statePrincipal {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("principal in state not set in boundary: %s", statePrincipal)
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
		projID, ok := rs.Primary.Attributes["scope_id"]
		if !ok {
			return fmt.Errorf("scope_id is not set")
		}
		projClient := md.client.Clone()
		projClient.SetScopeId(projID)
		rolesClient := roles.NewRolesClient(projClient)

		r, _, err := rolesClient.Read(md.ctx, id)
		if err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		// for every principal set as a principal on the role in the state, ensure
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
		projID, ok := rs.Primary.Attributes["scope_id"]
		if !ok {
			return fmt.Errorf("scope_id is not set")
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
				return fmt.Errorf("expected grant not found on role, %s: %s\n  Have: %#v\n", r.Name, grant, r)
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
			case "boundary_project":
				continue
			case "boundary_user":
				continue
			case "boundary_group":
				continue
			case "boundary_role":

				id := rs.Primary.ID
				projID, ok := rs.Primary.Attributes["scope_id"]
				if !ok {
					return fmt.Errorf("scope_id is not set")
				}
				projClient := md.client.Clone()
				projClient.SetScopeId(projID)
				rolesClient := roles.NewRolesClient(projClient)

				_, apiErr, _ := rolesClient.Read(md.ctx, id)
				if apiErr == nil || apiErr.Status != http.StatusNotFound && apiErr.Status != http.StatusForbidden {
					return fmt.Errorf("Didn't get a 404 or 403 when reading destroyed role %q: %v", id, apiErr)
				}

			default:
				continue
			}
		}
		return nil
	}
}
