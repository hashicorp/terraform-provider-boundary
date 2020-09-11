package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api/users"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	fooUserDescription       = "bar"
	fooUserDescriptionUpdate = "foo bar"
	fooUserDescriptionUnset  = ""
)

var (
	orgUser = fmt.Sprintf(`
resource "boundary_user" "org1" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.org1.id
}`, fooUserDescription)

	orgUserUpdate = fmt.Sprintf(`
resource "boundary_user" "org1" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.org1.id
}`, fooUserDescriptionUpdate)
)

// NOTE: this test also tests out the direct token auth mechanism.

func TestAccUser(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]
	token := tc.Token().Token

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckUserResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfigWithToken(url, token, fooOrg, orgUser),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists("boundary_user.org1"),
					resource.TestCheckResourceAttr("boundary_user.org1", userDescriptionKey, fooUserDescription),
					resource.TestCheckResourceAttr("boundary_user.org1", userNameKey, "test"),
				),
			},
			{
				// test update description
				Config: testConfigWithToken(url, token, fooOrg, orgUserUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists("boundary_user.org1"),
					resource.TestCheckResourceAttr("boundary_user.org1", userDescriptionKey, fooUserDescriptionUpdate),
					resource.TestCheckResourceAttr("boundary_user.org1", userNameKey, "test"),
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
		usrs := users.NewClient(md.client)

		_, apiErr, err := usrs.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading user %q: %v", id, err)
		}
		if apiErr != nil {
			return fmt.Errorf("Got an api error when reading user %q: %v", id, apiErr.Message)
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
				usrs := users.NewClient(md.client)

				_, apiErr, err := usrs.Read(context.Background(), id)
				if err != nil {
					return fmt.Errorf("Error when reading destroyed user %q: %v", id, err)
				}
				if apiErr == nil || apiErr.Status != http.StatusNotFound {
					return fmt.Errorf("Didn't get a 404 when reading destroyed user %q: %v", id, apiErr)
				}

			default:
				t.Logf("Got unknown resource type %q", rs.Type)
			}
		}
		return nil
	}
}
