package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	fooAuthMethodDesc       = "test auth method"
	fooAuthMethodDescUpdate = "test auth method update"
)

var (
	fooAuthMethod = fmt.Sprintf(`
resource "boundary_auth_method_password" "foo" {
	name        = "test"
	description = "%s"
	type        = "password"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`, fooAuthMethodDesc)

	fooAuthMethodUpdate = fmt.Sprintf(`
resource "boundary_auth_method_password" "foo" {
	name        = "test"
	description = "%s"
	type        = "password"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`, fooAuthMethodDescUpdate)
)

func TestAccAuthMethodPassword(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAuthMethodResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, fooAuthMethod),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "description", fooAuthMethodDesc),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "type", "password"),
					testAccCheckAuthMethodResourceExists(provider, "boundary_auth_method_password.foo"),
				),
			},
			importStep("boundary_auth_method_password.foo"),
			{
				// update
				Config: testConfig(url, fooOrg, fooAuthMethodUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "description", fooAuthMethodDescUpdate),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method_password.foo", "type", "password"),
					testAccCheckAuthMethodResourceExists(provider, "boundary_auth_method_password.foo"),
				),
			},
			importStep("boundary_auth_method_password.foo"),
		},
	})
}

func testAccCheckAuthMethodResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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

		amClient := authmethods.NewClient(md.client)

		if _, err := amClient.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error when reading auth method %q: %w", id, err)
		}

		return nil
	}
}

func testAccCheckAuthMethodResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_auth_method_password":
				id := rs.Primary.ID

				amClient := authmethods.NewClient(md.client)

				_, err := amClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed auth method %q: %v", id, err)
				}

			default:
				continue
			}
		}
		return nil
	}
}
