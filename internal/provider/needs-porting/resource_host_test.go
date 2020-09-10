package provider

import (
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
  scope_id    = boundary_project.foo.id
	type        = "static"
}

resource "boundary_host" "foo" {
  name            = "test"
	description     = "test host"
	host_catalog_id = boundary_host_catalog.foo.id
	address         = "%s"
	scope_id        = boundary_project.foo.id
}`, fooHostAddress)

	projHostUpdate = fmt.Sprintf(`
resource "boundary_host_catalog" "foo" {
  name        = "test"
	description = "test catalog"
  scope_id    = boundary_project.foo.id
	type        = "static"
}

resource "boundary_host" "foo" {
  name            = "test"
  description     = "test host"
	host_catalog_id = boundary_host_catalog.foo.id
	address         = "%s"
	scope_id        = boundary_project.foo.id
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
				Config: testConfig(url, fooOrg, fooProject, projHost),
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
				Config: testConfig(url, fooOrg, fooProject, projHostUpdate),
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

		hostCatalogID, ok := rs.Primary.Attributes["host_catalog_id"]
		if !ok {
			return errors.New("host_catalog_id is not set")
		}

		projID, ok := rs.Primary.Attributes["scope_id"]
		if !ok {
			return fmt.Errorf("scope_id is not set")
		}

		projClient := md.client.Clone()
		projClient.SetScopeId(projID)
		hostsClient := hosts.NewClient(projClient)

		if _, _, err := hostsClient.Read(md.ctx, hostCatalogID, id); err != nil {
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

		hostCatalogID, ok := rs.Primary.Attributes["host_catalog_id"]
		if !ok {
			return errors.New("host_catalog_id is not set")
		}

		md := testProvider.Meta().(*metaData)
		projID, ok := rs.Primary.Attributes["scope_id"]
		if !ok {
			return fmt.Errorf("scope_id is not set")
		}
		projClient := md.client.Clone()
		projClient.SetScopeId(projID)
		hostsClient := hosts.NewClient(projClient)

		h, _, err := hostsClient.Read(md.ctx, hostCatalogID, id)
		if err != nil {
			return fmt.Errorf("Got an error when reading host %q: %v", id, err)
		}

		if len(h.Attributes) == 0 {
			return errors.New("no host attributes found")
		}

		gotAttrVal, ok := h.Attributes[attrKey]
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
			case "boundary_project":
				continue
			case "boundary_organization":
				continue
			case "boundary_host":

				id := rs.Primary.ID
				hostCatalogID, ok := rs.Primary.Attributes["host_catalog_id"]
				if !ok {
					return errors.New("host_catalog_id is not set")
				}

				projID, ok := rs.Primary.Attributes["scope_id"]
				if !ok {
					return fmt.Errorf("scope_id is not set")
				}

				projClient := md.client.Clone()
				projClient.SetScopeId(projID)
				hostsClient := hosts.NewClient(projClient)

				_, apiErr, _ := hostsClient.Read(md.ctx, hostCatalogID, id)
				if apiErr == nil || apiErr.Status != http.StatusNotFound && apiErr.Status != http.StatusForbidden {
					return fmt.Errorf("Didn't get a 404 or 403 when reading destroyed host %q: %v", id, apiErr)
				}

			default:
				continue
			}
		}
		return nil
	}
}
