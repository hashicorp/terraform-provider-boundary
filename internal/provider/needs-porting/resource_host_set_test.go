package provider

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api/hostsets"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	fooHostset = ` 
resource "boundary_host_catalog" "foo" {
	scope_id    = boundary_project.foo.id
	type        = "static"
}

resource "boundary_host" "foo" {
	scope_id        = boundary_project.foo.id
	host_catalog_id = boundary_host_catalog.foo.id
	address         = "10.0.0.1:80"
}

resource "boundary_host_set" "foo" {
	name               = "test"
	description        = "test hostset"
	host_catalog_id    = boundary_host_catalog.foo.id
	host_ids           = [boundary_host.foo.id]
}`

	fooHostsetUpdate = `
resource "boundary_host_catalog" "foo" {
	scope_id    = boundary_project.foo.id
	type        = "static"
}

resource "boundary_host" "foo" {
	scope_id        = boundary_project.foo.id
	host_catalog_id = boundary_host_catalog.foo.id
	address         = "10.0.0.1:80"
}

resource "boundary_host" "bar" {
	scope_id        = boundary_project.foo.id
	host_catalog_id = boundary_host_catalog.foo.id
	address         = "10.0.0.2:80"
}

resource "boundary_host_set" "foo" {
	name               = "test"
	description        = "test hostset"
	host_catalog_id    = boundary_host_catalog.foo.id
	host_ids           = [
	  	boundary_host.foo.id, 
		boundary_host.bar.id,
	]
}`
)

func TestAccHostset(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	//	org := iam.TestOrg(t, tc.IamRepo())
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckHostsetResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test project hostset create
				Config: testConfig(url, fooOrg, fooProject, fooHostset),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostsetResourceExists("boundary_host_set.foo"),
					testAccCheckHostsetHostIDsSet("boundary_host_set.foo", []string{"boundary_host.foo"}),
					resource.TestCheckResourceAttr("boundary_host_set.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_host_set.foo", "description", "test hostset"),
				),
			},
			{
				// test project hostset update
				Config: testConfig(url, fooOrg, fooProject, fooHostsetUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostsetResourceExists("boundary_host_set.foo"),
					testAccCheckHostsetHostIDsSet("boundary_host_set.foo", []string{"boundary_host.foo", "boundary_host.bar"}),
					resource.TestCheckResourceAttr("boundary_host_set.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_host_set.foo", "description", "test hostset"),
				),
			},
		},
	})
}

func testAccCheckHostsetResourceExists(name string) resource.TestCheckFunc {
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

		hostsetsClient := hostsets.NewClient(md.client)

		if _, _, err := hostsetsClient.Read2(md.ctx, id); err != nil {
			return fmt.Errorf("Got an error when reading hostset %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckHostsetHostIDsSet(name string, wantHostIDs []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("role resource not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("host resource ID is not set")
		}

		gotHostIDs := []string{}
		for _, hostResourceName := range wantHostIDs {
			ur, ok := s.RootModule().Resources[hostResourceName]
			if !ok {
				return fmt.Errorf("host resource not found: %s", hostResourceName)
			}

			hostID := ur.Primary.ID
			if id == "" {
				return fmt.Errorf("host resource ID not set")
			}

			gotHostIDs = append(gotHostIDs, hostID)
		}

		// check boundary to ensure it matches
		md := testProvider.Meta().(*metaData)
		client := md.client.Clone()

		hstClient := hostsets.NewClient(client)

		hs, _, err := hstClient.Read2(md.ctx, id)
		if err != nil {
			return fmt.Errorf("Got an error when reading hostset %q: %v", id, err)
		}

		if len(hs.HostIds) == 0 {
			return fmt.Errorf("no hosts found on hostset")
		}

		for _, stateHost := range hs.HostIds {
			ok := false
			for _, gotHost := range gotHostIDs {
				if gotHost == stateHost {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("host in state not set in boundary: %s", stateHost)
			}
		}

		return nil
	}
}

func testAccCheckHostsetResourceDestroy(t *testing.T) resource.TestCheckFunc {
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
			case "boundary_host_set":

				id := rs.Primary.ID

				hostsetsClient := hostsets.NewClient(md.client)

				_, apiErr, _ := hostsetsClient.Read2(md.ctx, id)
				if apiErr == nil || apiErr.Status != http.StatusNotFound && apiErr.Status != http.StatusForbidden {
					return fmt.Errorf("Didn't get a 404 or 403 when reading destroyed hostset %q: %v", id, apiErr)
				}

			default:
				continue
			}
		}
		return nil
	}
}
