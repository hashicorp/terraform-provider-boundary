package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	fooOrg = `
resource "boundary_scope" "global" {
	global_scope = true
	name = "global"
	description = "Global Scope"
	scope_id = "global"
}

resource "boundary_role" "default" {
	default_role = true
	description = "Default role created on first instantiation of Boundary. It is meant to provide enough permissions for users to successfully authenticate via various client types."
	grant_scope_id = "global"
	name = "default"
	scope_id = boundary_scope.global.id
	principal_ids = ["u_auth", "u_anon"]
	grant_strings = [
		"type=scope;actions=list",
		"type=auth-method;actions=authenticate,list"
	]
}

resource "boundary_scope" "org1" {
	name = "org1"
	scope_id = boundary_scope.global.id
}

resource "boundary_role" "org1_admin" {
	scope_id = boundary_scope.global.id
	grant_scope_id = boundary_scope.org1.id
	grant_strings = ["id=*;actions=*"]
	principal_ids = ["u_auth"]
}
`

	firstProjectFoo = `
resource "boundary_scope" "proj1" {
	name = "proj1"
	scope_id    = boundary_scope.org1.id
	description = "foo"
	depends_on = [boundary_role.org1_admin]
}

resource "boundary_role" "proj1_admin" {
	scope_id = boundary_scope.org1.id
	grant_scope_id = boundary_scope.proj1.id
	grant_strings = ["id=*;actions=*"]
	principal_ids = ["u_auth"]
}
`

	firstProjectBar = `
resource "boundary_scope" "proj1" {
	name = "proj1"
	scope_id    = boundary_scope.org1.id
	description = "bar"
	depends_on = [boundary_role.org1_admin]
}

resource "boundary_role" "proj1_admin" {
	scope_id = boundary_scope.org1.id
	grant_scope_id = boundary_scope.proj1.id
	grant_strings = ["id=*;actions=*"]
	principal_ids = ["u_auth"]
}
`
	secondProject = `
resource "boundary_scope" "proj2" {
	name = "proj2"
	scope_id    = boundary_scope.org1.id
	description = "project2"
	depends_on = [boundary_role.org1_admin]
}

resource "boundary_role" "proj2_admin" {
	scope_id = boundary_scope.org1.id
	grant_scope_id = boundary_scope.proj2.id
	grant_strings = ["id=*;actions=*"]
	principal_ids = ["u_auth"]
}
`
)

func TestAccScopeCreation(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckScopeResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg, firstProjectFoo, secondProject),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists("boundary_scope.org1"),
					testAccCheckScopeResourceExists("boundary_scope.proj1"),
					resource.TestCheckResourceAttr("boundary_scope.proj1", DescriptionKey, "foo"),
					resource.TestCheckResourceAttr("boundary_scope.proj2", DescriptionKey, "project2"),
				),
			},
			// Updates the first project to have description bar
			{
				Config: testConfig(url, fooOrg, firstProjectBar, secondProject),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists("boundary_scope.proj1"),
					resource.TestCheckResourceAttr("boundary_scope.proj1", DescriptionKey, "bar"),
					resource.TestCheckResourceAttr("boundary_scope.proj2", DescriptionKey, "project2"),
				),
			},
			// Remove second project
			{
				Config: testConfig(url, fooOrg, firstProjectBar),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists("boundary_scope.proj1"),
					resource.TestCheckResourceAttr("boundary_scope.proj1", DescriptionKey, "bar"),
				),
			},
		},
	})
}

func testAccCheckScopeResourceExists(name string) resource.TestCheckFunc {
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
		scp := scopes.NewClient(md.client)

		_, err := scp.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading scope %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckScopeResourceDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// retrieve the connection established in Provider configuration
		md := testProvider.Meta().(*metaData)
		scp := scopes.NewClient(md.client)

		for _, rs := range s.RootModule().Resources {
			id := rs.Primary.ID
			switch rs.Type {
			case "boundary_scope":
				_, err := scp.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Status != http.StatusNotFound {
					return fmt.Errorf("Didn't get a 404 when reading destroyed resource %q: %w", id, err)
				}
			default:
				t.Logf("Got unknown resource type %q", rs.Type)
			}
		}
		return nil
	}
}
