package provider

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api/targets"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const (
	fooTargetDescription       = "bar"
	fooTargetDescriptionUpdate = "foo bar"
)

var (
	fooHostSet = `
resource "boundary_host_catalog" "foo" {
  name        = "test"
	description = "test catalog"
  scope_id    = boundary_project.foo.id
	type        = "static"
}

resource "boundary_host" "foo" {
  name            = "foo"
	host_catalog_id = boundary_host_catalog.foo.id
	scope_id        = boundary_project.foo.id
	address         = "10.0.0.1:80"
}

resource "boundary_host" "bar" {
  name            = "bar"
	host_catalog_id = boundary_host_catalog.foo.id
	scope_id        = boundary_project.foo.id
	address         = "10.0.0.1:80"
}

resource "boundary_host_set" "foo" {
  name            = "foo"
  host_catalog_id = boundary_host_catalog.foo.id

  host_ids = [
    boundary_host.foo.id,
		boundary_host.bar.id,
	]
}`

	fooTarget = fmt.Sprintf(`
resource "boundary_target" "foo" {
  name         = "test"
	description  = "%s"
	scope_id     = boundary_project.foo.id
	host_set_ids = [
    boundary_host_set.foo.id
	]
}`, fooTargetDescription)

	fooTargetUpdate = fmt.Sprintf(`
resource "boundary_target" "foo" {
  name         = "test"
	description  = "%s"
	scope_id     = boundary_project.foo.id
	host_set_ids = [
    boundary_host_set.foo.id
	]
}`, fooTargetDescriptionUpdate)
)

func TestAccTarget(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)

	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckTargetResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, fooProject, fooHostSet, fooTarget),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetResourceExists("boundary_target.foo"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetDescriptionKey, fooTargetDescription),
					resource.TestCheckResourceAttr("boundary_target.foo", targetNameKey, "test"),
				),
			},
			{
				// test update
				Config: testConfig(url, fooOrg, fooProject, fooHostSet, fooTargetUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetResourceExists("boundary_target.foo"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetDescriptionKey, fooTargetDescriptionUpdate),
				),
			},
		},
	})
}

func testAccCheckTargetResourceMembersSet(name string, hostSets []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("target resource not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("target resource ID is not set")
		}

		// ensure host sets are declared in state
		hostSetIDs := []string{}
		for _, hostSetResourceID := range hostSets {
			hs, ok := s.RootModule().Resources[hostSetResourceID]
			if !ok {
				return fmt.Errorf("host set resource not found: %s", hostSetResourceID)
			}

			hostSetID := hs.Primary.ID
			if id == "" {
				return fmt.Errorf("host set resource ID not set")
			}

			hostSetIDs = append(hostSetIDs, hostSetID)
		}

		// check boundary to ensure it matches
		md := testProvider.Meta().(*metaData)
		client := md.client.Clone()

		projID, ok := rs.Primary.Attributes["scope_id"]
		if ok {
			client.SetScopeId(projID)
		}
		tgtsClient := targets.NewClient(client)

		t, _, err := tgtsClient.Read(md.ctx, id)
		if err != nil {
			return fmt.Errorf("Got an error when reading target %q: %v", id, err)
		}

		if len(t.HostSetIds) == 0 {
			return fmt.Errorf("no hostSets found on target")
		}

		for _, stateHostSet := range t.HostSetIds {
			ok := false
			for _, gotHostSetID := range hostSetIDs {
				if gotHostSetID == stateHostSet {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("host set in state not set in boundary: %s", stateHostSet)
			}
		}

		return nil
	}
}

func testAccCheckTargetResourceExists(name string) resource.TestCheckFunc {
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
		projClient := md.client.Clone()
		projID, ok := rs.Primary.Attributes["scope_id"]
		if ok && projID != "" {
			projClient.SetScopeId(projID)
		}
		tgts := targets.NewClient(projClient)

		if _, _, err := tgts.Read(md.ctx, id); err != nil {
			return fmt.Errorf("Got an error when reading target %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckTargetResourceDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}

		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_target":
				projClient := md.client.Clone()
				projID, ok := rs.Primary.Attributes["scope_id"]
				if ok {
					projClient.SetScopeId(projID)
				}
				tgts := targets.NewClient(projClient)

				id := rs.Primary.ID

				_, apiErr, _ := tgts.Read(md.ctx, id)
				if apiErr == nil || apiErr.Status != http.StatusForbidden && apiErr.Status != http.StatusNotFound {
					return fmt.Errorf("Didn't get a 403 or 404 when reading destroyed resource %q: %v", id, apiErr)
				}

			default:
				continue
			}
		}
		return nil
	}
}
