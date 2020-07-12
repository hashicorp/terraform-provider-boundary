package provider

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/hashicorp/watchtower/api/scopes"
	"github.com/hashicorp/watchtower/api/users"
	"github.com/hashicorp/watchtower/testing/controller"
)

const (
	fooUserDescription       = "bar"
	fooUserDescriptionUpdate = "foo bar"
	fooUserDescriptionUnset  = ""
)

var (
	fooUser = fmt.Sprintf(`
resource "watchtower_user" "foo" {
  name = "test"
	description = "%s"
}`, fooUserDescription)

	fooUserUpdate = fmt.Sprintf(`
resource "watchtower_user" "foo" {
  name = "test"
	description = "%s"
}`, fooUserDescriptionUpdate)
	fooUserUnset = fmt.Sprintf(`
resource "watchtower_user" "foo" {
  name = "test"
	description = "%s"
}`, fooUserDescriptionUnset)
)

func TestAccUser(t *testing.T) {
	tc := controller.NewTestController(t, controller.WithDefaultOrgId("o_0000000000"))
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckUserResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooUser),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists("watchtower_user.foo"),
					resource.TestCheckResourceAttr("watchtower_user.foo", userDescriptionKey, fooUserDescription),
					resource.TestCheckResourceAttr("watchtower_user.foo", userNameKey, "test"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooUserUnset),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists("watchtower_user.foo"),
					resource.TestCheckResourceAttr("watchtower_user.foo", userDescriptionKey, fooUserDescriptionUnset),
				),
			},
			{
				// test unset
				Config: testConfig(url, fooUserUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists("watchtower_user.foo"),
					resource.TestCheckResourceAttr("watchtower_user.foo", userDescriptionKey, fooUserDescriptionUpdate),
				),
			},

			{
				// test destroy
				Config: testConfig(url),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserDestroyed("watchtower_user.foo"),
				),
			},
		},
	})
}

// testAccCheckUserDestroyed checks the terraform state for the host
// catalog and returns an error if found.
//
// TODO(malnick) This method falls short of checking the Watchtower API for
// the resource if the resource is not found in state. This is due to us not
// having the host catalog ID, but it doesn't guarantee that the resource was
// successfully removed.
//
// It does check Watchtower if the resource is found in state to point out any
// misalignment between what is in state and the actual configuration.
func testAccCheckUserDestroyed(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			// If it's not in state, it's destroyed in TF but not guaranteed to be destroyed
			// in Watchtower. Need to find a way to get the host catalog ID here so we can
			// form a lookup to the WT API to check this.
			return nil
		}
		errs := []string{}
		errs = append(errs, fmt.Sprintf("Found user resource in state: %s", name))

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("No ID is set")
		}

		md := testProvider.Meta().(*metaData)

		u := users.User{Id: id}

		o := &scopes.Org{
			Client: md.client,
		}
		if _, apiErr, _ := o.ReadUser(md.ctx, &u); apiErr == nil || apiErr.Status != http.StatusNotFound {
			errs = append(errs, fmt.Sprintf("User not destroyed %q: %v", id, apiErr))
		}

		return errors.New(strings.Join(errs, ","))
	}
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

		u := users.User{Id: id}

		o := &scopes.Org{
			Client: md.client,
		}
		if _, _, err := o.ReadUser(md.ctx, &u); err != nil {
			return fmt.Errorf("Got an error when reading user %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckUserResourceDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		md := testProvider.Meta().(*metaData)
		client := md.client

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "watchtower_user":

				id := rs.Primary.ID

				u := users.User{Id: id}

				o := &scopes.Org{
					Client: client,
				}

				_, apiErr, _ := o.ReadUser(md.ctx, &u)
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
