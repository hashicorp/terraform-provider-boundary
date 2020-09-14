package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api/hosts"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckHostResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test project host create
				Config: testConfig(url, fooOrg, firstProjectFoo, projHost),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostAttributeSet("boundary_host.foo", "address", fooHostAddress),
					testAccCheckHostResourceExists("boundary_host.foo"),
					resource.TestCheckResourceAttr("boundary_host.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_host.foo", "description", "test host"),
					resource.TestCheckResourceAttr("boundary_host.foo", "address", fooHostAddress),
				),
			},
			{
				// test project host update
				Config: testConfig(url, fooOrg, firstProjectFoo, projHostUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostAttributeSet("boundary_host.foo", "address", fooHostAddressUpdate),
					testAccCheckHostResourceExists("boundary_host.foo"),
					resource.TestCheckResourceAttr("boundary_host.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_host.foo", "description", "test host"),
					resource.TestCheckResourceAttr("boundary_host.foo", "address", fooHostAddressUpdate),
				),
			},
		},
	})
}

func testAccCheckHostResourceExists(name string) resource.TestCheckFunc {
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

		if _, _, err := hostsClient.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error when reading host %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckHostAttributeSet(name, attrKey, wantAttrVal string) resource.TestCheckFunc {
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

		h, _, err := hostsClient.Read(context.Background(), id)
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

func testAccCheckHostResourceDestroy(t *testing.T) resource.TestCheckFunc {
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

				_, apiErr, err := hostsClient.Read(context.Background(), id)
				if err != nil {
					return fmt.Errorf("Error when reading destroyed host %q: %v", id, err)
				}
				if apiErr == nil || apiErr.Status != http.StatusNotFound {
					return fmt.Errorf("Didn't get a 404 when reading destroyed host %q: %v", id, apiErr)
				}

			default:
				continue
			}
		}
		return nil
	}
}
