package provider

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/hashicorp/watchtower/api/roles"
	"github.com/hashicorp/watchtower/api/scopes"
	"github.com/hashicorp/watchtower/testing/controller"
)

const (
	fooRoleDescription       = "bar"
	fooRoleDescriptionUpdate = "foo bar"
)

var (
	fooRole = fmt.Sprintf(`
resource "watchtower_role" "foo" {
  name = "test"
	description = "%s"
}`, fooRoleDescription)

	fooRoleUpdate = fmt.Sprintf(`
resource "watchtower_role" "foo" {
  name = "test"
	description = "%s"
}`, fooRoleDescriptionUpdate)

	fooRoleWithUser = `
resource "watchtower_user" "foo" {
  name = "foo"
}

resource "watchtower_role" "foo" {
  name = "test"
	description = "test description"
	users = [watchtower_user.foo.id]
}`

	fooRoleWithUserUpdate = `
resource "watchtower_user" "foo" {
  name = "foo"
}

resource "watchtower_user" "bar" {
  name = "bar"
}

resource "watchtower_role" "foo" {
  name = "test"
	description = "test description"
	users = [watchtower_user.foo.id, watchtower_user.bar.id]
}`

	fooRoleWithGroups = `
resource "watchtower_group" "foo" {
  name = "foo"
}

resource "watchtower_role" "foo" {
  name = "test"
	description = "test description"
	groups = [watchtower_group.foo.id]
}`

	fooRoleWithGroupsUpdate = `
resource "watchtower_group" "foo" {
  name = "foo"
}

resource "watchtower_group" "bar" {
  name = "bar"
}

resource "watchtower_role" "foo" {
  name = "test"
	description = "test description"
	groups = [watchtower_group.foo.id, watchtower_group.bar.id]
}`

	readonlyGrant       = "id=*;actions=read"
	readonlyGrantUpdate = "id=*;actions=read,create"
	fooRoleWithGrants   = fmt.Sprintf(`
resource "watchtower_role" "foo" {
  name = "readonly"
	description = "test description"
	grants = ["%s"]
}`, readonlyGrant)
	fooRoleWithGrantsUpdate = fmt.Sprintf(`
resource "watchtower_role" "foo" {
  name = "readonly"
	description = "test description"
	grants = ["%s", "%s"]
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
				Config: testConfig(url, fooRoleWithGrants),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("watchtower_role.foo"),
					testAccCheckRoleResourceGrantsSet("watchtower_role.foo", []string{readonlyGrant}),
					resource.TestCheckResourceAttr("watchtower_role.foo", "name", "readonly"),
					resource.TestCheckResourceAttr("watchtower_role.foo", "description", "test description"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooRoleWithGrantsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("watchtower_role.foo"),
					testAccCheckRoleResourceGrantsSet("watchtower_role.foo", []string{readonlyGrant, readonlyGrantUpdate}),
					resource.TestCheckResourceAttr("watchtower_role.foo", "name", "readonly"),
					resource.TestCheckResourceAttr("watchtower_role.foo", "description", "test description"),
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
				Config: testConfig(url, fooRoleWithUser),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("watchtower_role.foo"),
					testAccCheckRoleResourceUsersSet("watchtower_role.foo", []string{"watchtower_user.foo"}),
					testAccCheckUserResourceExists("watchtower_user.foo"),
					resource.TestCheckResourceAttr("watchtower_role.foo", roleDescriptionKey, "test description"),
					resource.TestCheckResourceAttr("watchtower_role.foo", roleNameKey, "test"),
					resource.TestCheckResourceAttr("watchtower_user.foo", "name", "foo"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooRoleWithUserUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("watchtower_role.foo"),
					testAccCheckUserResourceExists("watchtower_user.foo"),
					testAccCheckUserResourceExists("watchtower_user.bar"),
					testAccCheckRoleResourceUsersSet("watchtower_role.foo", []string{"watchtower_user.foo", "watchtower_user.bar"}),
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
				Config: testConfig(url, fooRoleWithGroups),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("watchtower_role.foo"),
					testAccCheckRoleResourceGroupsSet("watchtower_role.foo", []string{"watchtower_group.foo"}),
					testAccCheckUserResourceExists("watchtower_group.foo"),
					resource.TestCheckResourceAttr("watchtower_role.foo", roleDescriptionKey, "test description"),
					resource.TestCheckResourceAttr("watchtower_role.foo", roleNameKey, "test"),
					resource.TestCheckResourceAttr("watchtower_group.foo", "name", "foo"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooRoleWithGroupsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("watchtower_role.foo"),
					testAccCheckUserResourceExists("watchtower_group.foo"),
					testAccCheckUserResourceExists("watchtower_group.bar"),
					testAccCheckRoleResourceGroupsSet("watchtower_role.foo", []string{"watchtower_group.foo", "watchtower_group.bar"}),
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
				Config: testConfig(url, fooRole),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("watchtower_role.foo"),
					resource.TestCheckResourceAttr("watchtower_role.foo", roleDescriptionKey, fooRoleDescription),
					resource.TestCheckResourceAttr("watchtower_role.foo", roleNameKey, "test"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooRoleUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists("watchtower_role.foo"),
					resource.TestCheckResourceAttr("watchtower_role.foo", roleDescriptionKey, fooRoleDescriptionUpdate),
				),
			},
			{
				// test destroy
				Config: testConfig(url),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleDestroyed("watchtower_role.foo"),
				),
			},
		},
	})
}

// testAccCheckRoleDestroyed checks the terraform state for the host
// catalog and returns an error if found.
//
// TODO(malnick) This method falls short of checking the Watchtower API for
// the resource if the resource is not found in state. This is due to us not
// having the host catalog ID, but it doesn't guarantee that the resource was
// successfully removed.
//
// It does check Watchtower if the resource is found in state to point out any
// misalignment between what is in state and the actual configuration.
func testAccCheckRoleDestroyed(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			// If it's not in state, it's destroyed in TF but not guaranteed to be destroyed
			// in Watchtower. Need to find a way to get the host catalog ID here so we can
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

		u := roles.Role{Id: id}

		o := &scopes.Org{
			Client: md.client,
		}
		if _, apiErr, _ := o.ReadRole(md.ctx, &u); apiErr == nil || apiErr.Status != http.StatusNotFound {
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

		u := roles.Role{Id: id}

		o := &scopes.Org{
			Client: md.client,
		}
		if _, _, err := o.ReadRole(md.ctx, &u); err != nil {
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

		u := roles.Role{Id: id}

		o := &scopes.Org{
			Client: md.client,
		}

		r, _, err := o.ReadRole(md.ctx, &u)
		if err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		// for every user set as a principle on the role in the state, ensure
		// each role in watchtower has the same setings
		if len(r.UserIds) == 0 {
			return fmt.Errorf("no users found in watchtower")
		}

		for _, stateUser := range r.UserIds {
			ok := false
			for _, gotUser := range userIDs {
				if gotUser == stateUser {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("user in state not set in watchtower: %s", stateUser)
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

		u := roles.Role{Id: id}

		o := &scopes.Org{
			Client: md.client,
		}

		r, _, err := o.ReadRole(md.ctx, &u)
		if err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		// for every user set as a principle on the role in the state, ensure
		// each role in watchtower has the same setings
		if len(r.GroupIds) == 0 {
			return fmt.Errorf("no groups found in watchtower")
		}

		for _, stateGroup := range r.GroupIds {
			ok := false
			for _, gotGroup := range groupIDs {
				if gotGroup == stateGroup {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("group in state not set in watchtower: %s", stateGroup)
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

		u := roles.Role{Id: id}

		o := &scopes.Org{
			Client: md.client,
		}

		r, _, err := o.ReadRole(md.ctx, &u)
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
				if gotGrant == grant {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("expected grant not found on role, %s: %s\n  Have: %v\n", *r.Name, grant, r)
			}
		}

		return nil
	}
}

func testAccCheckRoleResourceDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		md := testProvider.Meta().(*metaData)
		client := md.client

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "watchtower_user":
				continue
			case "watchtower_group":
				continue
			case "watchtower_role":

				id := rs.Primary.ID

				u := roles.Role{Id: id}

				o := &scopes.Org{
					Client: client,
				}

				_, apiErr, _ := o.ReadRole(md.ctx, &u)
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
