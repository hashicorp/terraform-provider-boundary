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
	fooBaseAuthMethodDesc       = "test auth method"
	fooBaseAuthMethodDescUpdate = "test auth method update"
)

// TODO(malnick) - remove these tests after deprecating use of password params in
// favor of attributes map
var (
	fooBaseAuthMethod = fmt.Sprintf(`
resource "boundary_auth_method" "foo" {
	name        = "test"
	description = "%s"
	type        = "password"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`, fooBaseAuthMethodDesc)

	fooBaseAuthMethodUpdate = fmt.Sprintf(`
resource "boundary_auth_method" "foo" {
	name        = "test"
	description = "%s"
	type        = "password"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`, fooBaseAuthMethodDescUpdate)
)

var (
	fooBaseAuthMethodAttrs = fmt.Sprintf(`
resource "boundary_auth_method" "foo" {
	name        = "test"
	description = "%s"
	type        = "password"
	scope_id    = boundary_scope.org1.id

	attributes = {
    "min_password_length" = 5
    "min_login_name_length" = 5
	}

	depends_on  = [boundary_role.org1_admin]
}`, fooBaseAuthMethodDesc)

	fooBaseAuthMethodUpdateAttrs = fmt.Sprintf(`
resource "boundary_auth_method" "foo" {
	name        = "test"
	description = "%s"
	type        = "password"
	scope_id    = boundary_scope.org1.id

  attributes = {
    "min_password_length" = 10 
    "min_login_name_length" = 10
	}

	depends_on  = [boundary_role.org1_admin]
}`, fooBaseAuthMethodDescUpdate)
)

func TestAccBaseAuthMethodPassword(t *testing.T) {
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
				Config: testConfig(url, fooOrg, fooBaseAuthMethod),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method.foo", "description", fooBaseAuthMethodDesc),
					resource.TestCheckResourceAttr("boundary_auth_method.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method.foo", "type", "password"),
					testAccCheckBaseAuthMethodResourceExists(provider, "boundary_auth_method.foo"),
				),
			},
			importStep("boundary_auth_method.foo"),

			{
				// update
				Config: testConfig(url, fooOrg, fooBaseAuthMethodUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method.foo", "description", fooBaseAuthMethodDescUpdate),
					resource.TestCheckResourceAttr("boundary_auth_method.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method.foo", "type", "password"),
					testAccCheckBaseAuthMethodResourceExists(provider, "boundary_auth_method.foo"),
				),
			},
			importStep("boundary_auth_method.foo"),
		},
	})
}

func TestAccBaseAuthMethodAttrs(t *testing.T) {
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
				Config: testConfig(url, fooOrg, fooBaseAuthMethodAttrs),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method.foo", "description", fooBaseAuthMethodDesc),
					resource.TestCheckResourceAttr("boundary_auth_method.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method.foo", "type", "password"),
					testAccCheckAuthMethodAttrSet(
						provider,
						"boundary_auth_method.foo",
						map[string]interface{}{
							"min_password_length":   5,
							"min_login_name_length": 5}),
					testAccCheckBaseAuthMethodResourceExists(provider, "boundary_auth_method.foo"),
				),
			},
			importStep("boundary_auth_method.foo"),

			{
				// update
				Config: testConfig(url, fooOrg, fooBaseAuthMethodUpdateAttrs),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method.foo", "description", fooBaseAuthMethodDescUpdate),
					resource.TestCheckResourceAttr("boundary_auth_method.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method.foo", "type", "password"),
					testAccCheckAuthMethodAttrSet(
						provider,
						"boundary_auth_method.foo",
						map[string]interface{}{
							"min_password_length":   10,
							"min_login_name_length": 10}),

					testAccCheckBaseAuthMethodResourceExists(provider, "boundary_auth_method.foo"),
				),
			},
			importStep("boundary_auth_method.foo"),
		},
	})
}

func testAccCheckBaseAuthMethodResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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

func testAccCheckBaseAuthMethodResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_auth_method":
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

func testAccCheckAuthMethodAttrSet(testProvider *schema.Provider, resourceName string, attrs map[string]interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("auth method resource not found: %s", resourceName)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("auth method resource ID is not set")
		}

		md := testProvider.Meta().(*metaData)
		amClient := authmethods.NewClient(md.client)

		amr, err := amClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading auth method %q: %v", id, err)
		}

		if fmt.Sprint(amr.Item.Attributes) == fmt.Sprint(attrs) {
			return nil
		}

		return fmt.Errorf("attrs not equal, got %+v, want %+v\n", amr.Item.Attributes, attrs)
	}
}
