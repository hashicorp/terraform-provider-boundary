package provider

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const (
	orgFooName              = "foo"
	orgFooDescription       = "foo description"
	orgFooDescriptionUpdate = "foo bar description"
)

var (
	fooOrg = `
resource "boundary_organization" "foo" {}`

	orgFoo = fmt.Sprintf(`
resource "boundary_organization" "foo" {
	name        = "%s"
  description = "%s"
}`, orgFooName, orgFooDescription)

	orgFooUpdate = fmt.Sprintf(`
resource "boundary_organization" "foo" {
	name        = "%s"
  description = "%s"
}`, orgFooName, orgFooDescriptionUpdate)
)

func TestAccOrganizationCreation(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckOrganizationResourceDestroy(t),
		Steps: []resource.TestStep{
			// test create
			{
				Config: testConfig(url, orgFoo),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrganizationResourceExists("boundary_organization.foo"),
					resource.TestCheckResourceAttr("boundary_organization.foo", organizationNameKey, orgFooName),
					resource.TestCheckResourceAttr("boundary_organization.foo", organizationDescriptionKey, orgFooDescription),
				),
			},
			// test update
			{
				Config: testConfig(url, orgFooUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrganizationResourceExists("boundary_organization.foo"),
					resource.TestCheckResourceAttr("boundary_organization.foo", organizationNameKey, orgFooName),
					resource.TestCheckResourceAttr("boundary_organization.foo", organizationDescriptionKey, orgFooDescriptionUpdate),
				),
			},
		},
	})
}

func testAccCheckOrganizationResourceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("No ID is set")
		}

		if !strings.HasPrefix(id, "o_") {
			return fmt.Errorf("ID not formatted as expected, expected prefix 'o_', got %s", id)
		}

		md := testProvider.Meta().(*metaData)
		scp := scopes.NewClient(md.client)

		if _, _, err := scp.Read(md.ctx, id); err != nil {
			return fmt.Errorf("Got an error when reading organization %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckOrganizationResourceDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// retrieve the connection established in Provider configuration
		md := testProvider.Meta().(*metaData)
		scp := scopes.NewClient(md.client)

		for _, rs := range s.RootModule().Resources {
			id := rs.Primary.ID
			switch rs.Type {
			case "boundary_organization":
				if _, apiErr, _ := scp.Read(md.ctx, id); apiErr == nil || apiErr.Status != http.StatusNotFound && apiErr.Status != http.StatusForbidden {
					return fmt.Errorf("Didn't get a 404 or 403 when reading destroyed organization %q: %v", id, apiErr)
				}
			default:
				t.Logf("Got unknown resource type %q", rs.Type)
			}
		}
		return nil
	}
}
