package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	fooOrg = `
resource "boundary_scope" "org1" {
	name     = "test"
	scope_id = "global"
}`

	firstProjectFoo = `
resource "boundary_scope" "proj1" {
	scope_id    = boundary_scope.org1.id
	description = "foo"
}`

	firstProjectBar = `
resource "boundary_scope" "proj1" {
	scope_id    = boundary_scope.org1.id
	description = "bar"
}`

	secondProject = `
resource "boundary_scope" "proj2" {
	scope_id    = boundary_scope.org1.id
	description = "project2"
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

		_, apiErr, err := scp.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading scope %q: %v", id, err)
		}
		if apiErr != nil {
			return fmt.Errorf("Got an API error when reading scope %q: %v", id, apiErr.Message)
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
				_, apiErr, err := scp.Read(context.Background(), id)
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
