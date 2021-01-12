package provider

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/hosts"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"google.golang.org/grpc/codes"
)

const (
	fooHostAddress       = "10.0.0.1"
	fooHostAddressUpdate = "10.10.0.0"
)

var (
	projHost = fmt.Sprintf(`
resource "boundary_host_catalog" "foo" {
	name        = "test"
	description = "test catalog"
	scope_id    = boundary_scope.proj1.id
	type        = "static"
	depends_on  = [boundary_role.proj1_admin]
}

resource "boundary_host" "foo" {
	type            = "static"
	host_catalog_id = boundary_host_catalog.foo.id
	name            = "test"
	description     = "test host"
	address         = "%s"
}`, fooHostAddress)

	projHostUpdate = fmt.Sprintf(`
resource "boundary_host_catalog" "foo" {
	name        = "test"
	description = "test catalog"
	scope_id    = boundary_scope.proj1.id
	type        = "static"
	depends_on  = [boundary_role.proj1_admin]
}

resource "boundary_host" "foo" {
	type            = "static"
	host_catalog_id = boundary_host_catalog.foo.id
	name            = "test"
	description     = "test host"
	address         = "%s"
}`, fooHostAddressUpdate)
)

func TestAccHost(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	//	org := iam.TestOrg(t, tc.IamRepo())
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckHostResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test project host create
				Config: testConfig(url, fooOrg, firstProjectFoo, projHost),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostAttributeSet(provider, "boundary_host.foo", "address", fooHostAddress),
					testAccCheckHostResourceExists(provider, "boundary_host.foo"),
					resource.TestCheckResourceAttr("boundary_host.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_host.foo", "description", "test host"),
					resource.TestCheckResourceAttr("boundary_host.foo", "address", fooHostAddress),
				),
			},
			importStep("boundary_host.foo"),
			{
				// test project host update
				Config: testConfig(url, fooOrg, firstProjectFoo, projHostUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostAttributeSet(provider, "boundary_host.foo", "address", fooHostAddressUpdate),
					testAccCheckHostResourceExists(provider, "boundary_host.foo"),
					resource.TestCheckResourceAttr("boundary_host.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_host.foo", "description", "test host"),
					resource.TestCheckResourceAttr("boundary_host.foo", "address", fooHostAddressUpdate),
				),
			},
			importStep("boundary_host.foo"),
		},
	})
}

func testAccCheckHostResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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

		hostsClient := hosts.NewClient(md.client)

		if _, err := hostsClient.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error when reading host %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckHostAttributeSet(testProvider *schema.Provider, name, attrKey, wantAttrVal string) resource.TestCheckFunc {
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
		hostsClient := hosts.NewClient(md.client)

		h, err := hostsClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading host %q: %v", id, err)
		}

		if len(h.Item.Attributes) == 0 {
			return errors.New("no host attributes found")
		}

		gotAttrVal, ok := h.Item.Attributes[attrKey]
		if !ok {
			return fmt.Errorf("attribute not found on host: '%s'", attrKey)
		}

		if gotAttrVal != wantAttrVal {
			return fmt.Errorf("got incorrect value for '%s': got '%s', want '%s'", attrKey, gotAttrVal, wantAttrVal)
		}

		return nil
	}
}

func testAccCheckHostResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_scope":
				continue
			case "boundary_host":
				id := rs.Primary.ID

				hostsClient := hosts.NewClient(md.client)

				_, err := hostsClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Kind != codes.NotFound.String() {
					return fmt.Errorf("didn't get a 404 when reading destroyed host %q: %v", id, apiErr)
				}

			default:
				continue
			}
		}
		return nil
	}
}
