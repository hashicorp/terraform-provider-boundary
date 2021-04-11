package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/accounts"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	fooAccountPasswordDesc       = "test account"
	fooAccountPasswordDescUpdate = "test account update"
)

var (
	fooAccountPassword = fmt.Sprintf(`
resource "boundary_auth_method" "foo" {
	name        = "test"
	description = "test account"
	type        = "password"
	scope_id    = boundary_scope.org1.id
	depends_on = [boundary_role.org1_admin]
}

resource "boundary_account_password" "foo" {
	name           = "test"
	description    = "%s"
	type           = "password"
	login_name     = "foo"
	password       = "foofoofoo"
	auth_method_id = boundary_auth_method.foo.id
}`, fooAccountPasswordDesc)

	fooAccountPasswordUpdate = fmt.Sprintf(`
resource "boundary_auth_method" "foo" {
	name        = "test"
	description = "test account"
	type        = "password"
	scope_id    = boundary_scope.org1.id
	depends_on = [boundary_role.org1_admin]
}

resource "boundary_account_password" "foo" {
	name           = "test"
	description    = "%s"
	type           = "password"
	login_name     = "foo"
	password       = "foofoofoo"
	auth_method_id = boundary_auth_method.foo.id
}`, fooAccountPasswordDescUpdate)
)

func TestAccAccount(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAccountPasswordResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, fooAccountPassword),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_account_password.foo", "description", fooAccountPasswordDesc),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "type", "password"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "login_name", "foo"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "password", "foofoofoo"),
					testAccCheckAccountPasswordResourceExists(provider, "boundary_account_password.foo"),
				),
			},
			importStep("boundary_account_password.foo", "password"),
			{
				// update
				Config: testConfig(url, fooOrg, fooAccountPasswordUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_account_password.foo", "description", fooAccountPasswordDescUpdate),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "type", "password"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "login_name", "foo"),
					resource.TestCheckResourceAttr("boundary_account_password.foo", "password", "foofoofoo"),
					testAccCheckAccountPasswordResourceExists(provider, "boundary_account_password.foo"),
				),
			},
			importStep("boundary_account_password.foo", "password"),
		},
	})
}

func testAccCheckAccountPasswordResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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

		amClient := accounts.NewClient(md.client)

		if _, err := amClient.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error when reading account %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckAccountPasswordResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_account_password":
				id := rs.Primary.ID

				amClient := accounts.NewClient(md.client)

				_, err := amClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed account %q: %v", id, err)
				}

			default:
				continue
			}
		}
		return nil
	}
}
