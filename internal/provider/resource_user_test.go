package provider

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api/users"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const (
	fooUserDescription       = "bar"
	fooUserDescriptionUpdate = "foo bar"
	fooUserDescriptionUnset  = ""
)

var (
	orgUser = fmt.Sprintf(`
resource "boundary_user" "foo" {
  name        = "test"
	description = "%s"
  scope_id    = boundary_organization.foo.id
}`, fooUserDescription)

	orgUserUpdate = fmt.Sprintf(`
resource "boundary_user" "foo" {
  name        = "test"
	description = "%s"
  scope_id    = boundary_organization.foo.id
}`, fooUserDescriptionUpdate)
)

func TestAccUser(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckUserResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, orgUser),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists("boundary_user.foo"),
					resource.TestCheckResourceAttr("boundary_user.foo", userDescriptionKey, fooUserDescription),
					resource.TestCheckResourceAttr("boundary_user.foo", userNameKey, "test"),
				),
			},
			{
				// test update description
				Config: testConfig(url, fooOrg, orgUserUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists("boundary_user.foo"),
					resource.TestCheckResourceAttr("boundary_user.foo", userDescriptionKey, fooUserDescriptionUpdate),
					resource.TestCheckResourceAttr("boundary_user.foo", userNameKey, "test"),
				),
			},
		},
	})
}

func testAccCheckUserResourceExists(name string) resource.TestCheckFunc {
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
		usrs := users.NewClient(projClient)

		if _, _, err := usrs.Read(md.ctx, id); err != nil {
			return fmt.Errorf("Got an error when reading user %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckUserResourceDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_user":

				id := rs.Primary.ID
				projID, ok := rs.Primary.Attributes["scope_id"]
				if !ok {
					return fmt.Errorf("scope_id is not set")
				}
				projClient := md.client.Clone()
				projClient.SetScopeId(projID)
				usrs := users.NewClient(projClient)

				_, apiErr, _ := usrs.Read(md.ctx, id)
				if apiErr == nil || apiErr.Status != http.StatusNotFound && apiErr.Status != http.StatusForbidden {
					return fmt.Errorf("Didn't get a 404 or 403 when reading destroyed user %q: %v", id, apiErr)
				}

			default:
				t.Logf("Got unknown resource type %q", rs.Type)
			}
		}
		return nil
	}
}
