package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/roles"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"google.golang.org/grpc/codes"
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
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`, fooRoleDescription)

	orgRoleUpdate = fmt.Sprintf(`
resource "boundary_role" "foo" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`, fooRoleDescriptionUpdate)

	projRole = fmt.Sprintf(`
resource "boundary_role" "foo" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.proj1.id
	depends_on  = [boundary_role.proj1_admin]
}`, fooRoleDescription)

	projRoleUpdate = fmt.Sprintf(`
resource "boundary_role" "foo" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.proj1.id
	depends_on  = [boundary_role.proj1_admin]
}`, fooRoleDescriptionUpdate)

	projRoleWithPrincipal = `
resource "boundary_user" "foo" {
	name       = "foo"
	scope_id   = boundary_scope.org1.id
	depends_on = [boundary_role.org1_admin]
}

resource "boundary_role" "with_principal" {
	name           = "with_principal"
	description    = "with principal"
	principal_ids  = [boundary_user.foo.id]
	scope_id       = boundary_scope.proj1.id
	depends_on     = [boundary_role.proj1_admin]
}`

	projRoleWithPrincipalUpdate = `
resource "boundary_user" "foo" {
	name       = "foo"
	scope_id   = boundary_scope.org1.id
	depends_on = [boundary_role.org1_admin]
}

resource "boundary_user" "bar" {
	name       = "bar"
	scope_id   = boundary_scope.org1.id
	depends_on = [boundary_role.org1_admin]
}

resource "boundary_role" "with_principal" {
	name           = "with_principal"
	description    = "with principal"
	principal_ids  = [boundary_user.foo.id, boundary_user.bar.id]
	scope_id       = boundary_scope.proj1.id
	depends_on     = [boundary_role.proj1_admin]
}`

	projRoleWithGroups = `
resource "boundary_group" "foo" {
	name       = "foo"
	scope_id   = boundary_scope.proj1.id
	depends_on = [boundary_role.proj1_admin]
}

resource "boundary_role" "with_groups" {
	name           = "with_groups"
	description    = "with groups"
	principal_ids  = [boundary_group.foo.id]
	scope_id       = boundary_scope.proj1.id
	depends_on     = [boundary_role.proj1_admin]
}`

	projRoleWithGroupsUpdate = `
resource "boundary_group" "foo" {
	name       = "foo"
	scope_id   = boundary_scope.proj1.id
	depends_on = [boundary_role.proj1_admin]
}

resource "boundary_group" "bar" {
	name       = "bar"
	scope_id   = boundary_scope.proj1.id
	depends_on = [boundary_role.proj1_admin]
}

resource "boundary_role" "with_groups" {
	name           = "with_groups"
	description    = "with groups"
	principal_ids  = [boundary_group.foo.id, boundary_group.bar.id]
	scope_id       = boundary_scope.proj1.id
	depends_on     = [boundary_role.proj1_admin]
}`

	readonlyGrant       = "id=*;type=*;actions=read"
	readonlyGrantUpdate = "id=*;type=*;actions=read,create"

	projRoleWithGrants = fmt.Sprintf(`
resource "boundary_role" "with_grants" {
	name          = "with_grants"
	description   = "with grants"
	grant_strings = ["%s"]
	scope_id      = boundary_scope.proj1.id
	depends_on    = [boundary_role.proj1_admin]
}`, readonlyGrant)

	projRoleWithGrantsUpdate = fmt.Sprintf(`
resource "boundary_role" "with_grants" {
	name          = "with_grants"
	description   = "with grants"
	grant_strings = ["%s", "%s"]
	scope_id      = boundary_scope.proj1.id
	depends_on    = [boundary_role.proj1_admin]
}`, readonlyGrant, readonlyGrantUpdate)
)

func TestAccRoleToOrgToProject(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckRoleResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test org role create
				Config: testConfig(url, fooOrg, orgRole),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists(provider, "boundary_role.foo"),
					resource.TestCheckResourceAttr("boundary_role.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_role.foo", "description", fooRoleDescription),
				),
			},
			importStep("boundary_role.foo"),
			{
				// test org role update
				Config: testConfig(url, fooOrg, orgRoleUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists(provider, "boundary_role.foo"),
					resource.TestCheckResourceAttr("boundary_role.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_role.foo", "description", fooRoleDescriptionUpdate),
				),
			},
			importStep("boundary_role.foo"),
			{
				// test org to project role create
				Config: testConfig(url, fooOrg, firstProjectFoo, projRole),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists(provider, "boundary_role.foo"),
					resource.TestCheckResourceAttr("boundary_role.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_role.foo", "description", fooRoleDescription),
				),
			},
			importStep("boundary_role.foo"),
			{
				// test project role update
				Config: testConfig(url, fooOrg, firstProjectFoo, projRoleUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists(provider, "boundary_role.foo"),
					resource.TestCheckResourceAttr("boundary_role.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_role.foo", "description", fooRoleDescriptionUpdate),
				),
			},
			importStep("boundary_role.foo"),
		},
	})
}

func TestAccRoleWithGrants(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckRoleResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test project role create with grants
				Config: testConfig(url, fooOrg, firstProjectFoo, projRoleWithGrants),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists(provider, "boundary_role.with_grants"),
					testAccCheckRoleResourceGrantsSet(provider, "boundary_role.with_grants", []string{readonlyGrant}),
					resource.TestCheckResourceAttr("boundary_role.with_grants", "name", "with_grants"),
					resource.TestCheckResourceAttr("boundary_role.with_grants", "description", "with grants"),
				),
			},
			importStep("boundary_role.with_grants"),
			{
				// test project role update with grants
				Config: testConfig(url, fooOrg, firstProjectFoo, projRoleWithGrantsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists(provider, "boundary_role.with_grants"),
					testAccCheckRoleResourceGrantsSet(provider, "boundary_role.with_grants", []string{readonlyGrant, readonlyGrantUpdate}),
					resource.TestCheckResourceAttr("boundary_role.with_grants", "name", "with_grants"),
					resource.TestCheckResourceAttr("boundary_role.with_grants", "description", "with grants"),
				),
			},
			importStep("boundary_role.with_grants"),
		},
	})
}

func TestAccRoleWithPrincipals(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckRoleResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, firstProjectFoo, projRoleWithPrincipal),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists(provider, "boundary_role.with_principal"),
					testAccCheckRoleResourcePrincipalsSet(provider, "boundary_role.with_principal", []string{"boundary_user.foo"}),
					testAccCheckUserResourceExists(provider, "boundary_user.foo"),
					resource.TestCheckResourceAttr("boundary_role.with_principal", DescriptionKey, "with principal"),
					resource.TestCheckResourceAttr("boundary_role.with_principal", NameKey, "with_principal"),
					resource.TestCheckResourceAttr("boundary_user.foo", "name", "foo"),
				),
			},
			importStep("boundary_role.with_principal"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, projRoleWithPrincipalUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists(provider, "boundary_role.with_principal"),
					testAccCheckUserResourceExists(provider, "boundary_user.foo"),
					testAccCheckUserResourceExists(provider, "boundary_user.bar"),
					testAccCheckRoleResourcePrincipalsSet(provider, "boundary_role.with_principal", []string{"boundary_user.foo", "boundary_user.bar"}),
				),
			},
			importStep("boundary_role.with_principal"),
		},
	})
}

func TestAccRoleWithGroups(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckRoleResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, firstProjectFoo, projRoleWithGroups),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists(provider, "boundary_role.with_groups"),
					testAccCheckRoleResourceGroupsSet(provider, "boundary_role.with_groups", []string{"boundary_group.foo"}),
					testAccCheckGroupResourceExists(provider, "boundary_group.foo"),
					resource.TestCheckResourceAttr("boundary_role.with_groups", DescriptionKey, "with groups"),
					resource.TestCheckResourceAttr("boundary_role.with_groups", NameKey, "with_groups"),
					resource.TestCheckResourceAttr("boundary_group.foo", "name", "foo"),
				),
			},
			importStep("boundary_role.with_groups"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, projRoleWithGroupsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists(provider, "boundary_role.with_groups"),
					testAccCheckGroupResourceExists(provider, "boundary_group.foo"),
					testAccCheckGroupResourceExists(provider, "boundary_group.bar"),
					testAccCheckRoleResourceGroupsSet(provider, "boundary_role.with_groups", []string{"boundary_group.foo", "boundary_group.bar"}),
				),
			},
			importStep("boundary_role.with_groups"),
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
func testAccCheckRoleDestroyed(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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
		rolesClient := roles.NewClient(md.client)

		_, err := rolesClient.Read(context.Background(), id)
		if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Kind != codes.NotFound.String() {
			errs = append(errs, fmt.Sprintf("Role not destroyed %q: %v", id, apiErr))
		}

		return errors.New(strings.Join(errs, ","))
	}
}

func testAccCheckRoleResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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
		rolesClient := roles.NewClient(md.client)

		if _, err := rolesClient.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckRoleResourcePrincipalsSet(testProvider *schema.Provider, name string, principals []string) resource.TestCheckFunc {
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
		rolesClient := roles.NewClient(md.client)

		rr, err := rolesClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		// for every principal set as a principal on the role in the state, ensure
		// each role in boundary has the same setings
		if len(rr.Item.PrincipalIds) == 0 {
			return fmt.Errorf("no principals found in boundary")
		}

		for _, statePrincipal := range rr.Item.PrincipalIds {
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

func testAccCheckRoleResourceGroupsSet(testProvider *schema.Provider, name string, groups []string) resource.TestCheckFunc {
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
		rolesClient := roles.NewClient(md.client)

		rr, err := rolesClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		// for every principal set as a principal on the role in the state, ensure
		// each role in boundary has the same setings
		if len(rr.Item.PrincipalIds) == 0 {
			return fmt.Errorf("no groups found in boundary")
		}

		for _, stateGroup := range rr.Item.PrincipalIds {
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

func testAccCheckRoleResourceGrantsSet(testProvider *schema.Provider, name string, expectedGrants []string) resource.TestCheckFunc {
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
		rolesClient := roles.NewClient(md.client)

		rr, err := rolesClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		// for each expected grant, ensure they're set on the role
		if len(rr.Item.Grants) == 0 {
			return fmt.Errorf("no grants found on role, %+v\n", rr)
		}

		for _, grant := range expectedGrants {
			ok = false
			for _, gotGrant := range rr.Item.Grants {
				if gotGrant.Raw == grant {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("expected grant not found on role, %s: %s\n  Have: %#v\n", rr.Item.Name, grant, rr.Item)
			}
		}

		return nil
	}
}

func testAccCheckRoleResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
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
				continue
			case "boundary_role":

				id := rs.Primary.ID
				rolesClient := roles.NewClient(md.client)

				_, err := rolesClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Kind != codes.NotFound.String() {
					return fmt.Errorf("Didn't get a 404 when reading destroyed role %q: %v", id, err)
				}

			default:
				continue
			}
		}
		return nil
	}
}
