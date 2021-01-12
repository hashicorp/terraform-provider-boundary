package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/targets"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"google.golang.org/grpc/codes"
)

const (
	fooTargetDescription       = "bar"
	fooTargetDescriptionUpdate = "foo bar"
)

var (
	fooHostSet = `
resource "boundary_host_catalog" "foo" {
	type        = "static"
	name        = "test"
	description = "test catalog"
	scope_id    = boundary_scope.proj1.id
	depends_on  = [boundary_role.proj1_admin]
}

resource "boundary_host" "foo" {
	name            = "foo"
	host_catalog_id = boundary_host_catalog.foo.id
	type            = "static"
	address         = "10.0.0.1"
}

resource "boundary_host" "bar" {
	name            = "bar"
	host_catalog_id = boundary_host_catalog.foo.id
	type            = "static"
	address         = "10.0.0.1"
}

resource "boundary_host_set" "foo" {
	name            = "foo"
	type            = "static"
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
	type         = "tcp"
	scope_id     = boundary_scope.proj1.id
	host_set_ids = [
		boundary_host_set.foo.id
	]
	default_port = 22
	depends_on  = [boundary_role.proj1_admin]
	session_max_seconds = 6000
	session_connection_limit = 6
}`, fooTargetDescription)

	fooTargetUpdate = fmt.Sprintf(`
resource "boundary_target" "foo" {
	name         = "test"
	description  = "%s"
	type         = "tcp"
	scope_id     = boundary_scope.proj1.id
	host_set_ids = [
		boundary_host_set.foo.id
	]
	default_port = 80
	depends_on  = [boundary_role.proj1_admin]
	session_max_seconds = 7000
	session_connection_limit = 7
}`, fooTargetDescriptionUpdate)
)

func TestAccTarget(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)

	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckTargetResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, firstProjectFoo, fooHostSet, fooTarget),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetResourceExists(provider, "boundary_target.foo"),
					resource.TestCheckResourceAttr("boundary_target.foo", DescriptionKey, fooTargetDescription),
					resource.TestCheckResourceAttr("boundary_target.foo", NameKey, "test"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetDefaultPortKey, "22"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetSessionMaxSecondsKey, "6000"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetSessionConnectionLimitKey, "6"),
				),
			},
			importStep("boundary_target.foo"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, fooHostSet, fooTargetUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetResourceExists(provider, "boundary_target.foo"),
					resource.TestCheckResourceAttr("boundary_target.foo", DescriptionKey, fooTargetDescriptionUpdate),
					resource.TestCheckResourceAttr("boundary_target.foo", targetDefaultPortKey, "80"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetSessionMaxSecondsKey, "7000"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetSessionConnectionLimitKey, "7"),
				),
			},
			importStep("boundary_target.foo"),
		},
	})
}

func testAccCheckTargetResourceMembersSet(testProvider *schema.Provider, name string, hostSets []string) resource.TestCheckFunc {
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
		tgtsClient := targets.NewClient(md.client)

		t, err := tgtsClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading target %q: %v", id, err)
		}

		if len(t.Item.HostSetIds) == 0 {
			return fmt.Errorf("no hostSets found on target")
		}

		for _, stateHostSet := range t.Item.HostSetIds {
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

func testAccCheckTargetResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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
		tgts := targets.NewClient(md.client)

		if _, err := tgts.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error when reading target %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckTargetResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}

		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_target":
				tgts := targets.NewClient(md.client)

				id := rs.Primary.ID

				_, err := tgts.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Kind != codes.NotFound.String() {
					return fmt.Errorf("didn't get a 404 when reading destroyed target %q: %v", id, apiErr)
				}

			default:
				continue
			}
		}
		return nil
	}
}
