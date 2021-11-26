package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/hostsets"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccHostSetPlugin(t *testing.T) {
	t.Run("basic-crud", func(t *testing.T) {
		t.Parallel()
		testAccHostSetPluginCrud(t)
	})
}

func testAccHostSetPluginCrud(t *testing.T) {
	initialPreferredEndpoints := []string{"cidr:1.2.3.4/32", "dns:bar.foo.com"}
	initialPreferredEndpointsRaw, err := json.Marshal(initialPreferredEndpoints)
	require.NoError(t, err)
	initialPreferredEndpointsStr := string(initialPreferredEndpointsRaw)
	updatePreferredEndpoints := []string{"dns:bar.foo.com", "cidr:4.3.2.0/24"}
	updatePreferredEndpointsRaw, err := json.Marshal(updatePreferredEndpoints)
	require.NoError(t, err)
	updatePreferredEndpointsStr := string(updatePreferredEndpointsRaw)

	initialSyncIntervalSeconds := -1
	updateSyncIntervalSeconds := 60

	catalogBlock := `
	resource "boundary_host_catalog_plugin" "foo" {
		scope_id    = boundary_scope.proj1.id
		depends_on  = [boundary_role.proj1_admin]
		plugin_name = "loopback"
	}`

	hostSetBlock := `
	resource "boundary_host_set_plugin" "foo" {
		host_catalog_id    = boundary_host_catalog_plugin.foo.id
		name               = "test"
		description        = "test hostset"
		preferred_endpoints = %s
		sync_interval_seconds = %d
	}`

	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]
	fooSetName := "boundary_host_set_plugin.foo"

	hsBlock := fmt.Sprintf(hostSetBlock, initialPreferredEndpointsStr, initialSyncIntervalSeconds)
	hsUpdateBlock := fmt.Sprintf(hostSetBlock, updatePreferredEndpointsStr, updateSyncIntervalSeconds)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckHostSetPluginResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test project hostset create
				Config: testConfig(url, fooOrg, firstProjectFoo, catalogBlock, hsBlock),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostSetPluginResourceExists(provider, fooSetName),
					resource.TestCheckResourceAttr(fooSetName, NameKey, "test"),
					resource.TestCheckResourceAttr(fooSetName, DescriptionKey, "test hostset"),
					resource.TestCheckResourceAttr(fooSetName, SyncIntervalSecondsKey, fmt.Sprintf("%d", initialSyncIntervalSeconds)),
					testAccCheckHostSetPluginPreferredEndpoints(t, provider, fooSetName, initialPreferredEndpoints),
				),
			},
			importStep(fooSetName),
			{
				// test project hostset update
				Config: testConfig(url, fooOrg, firstProjectFoo, catalogBlock, hsUpdateBlock),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostSetPluginResourceExists(provider, fooSetName),
					resource.TestCheckResourceAttr(fooSetName, NameKey, "test"),
					resource.TestCheckResourceAttr(fooSetName, DescriptionKey, "test hostset"),
					resource.TestCheckResourceAttr(fooSetName, SyncIntervalSecondsKey, fmt.Sprintf("%d", updateSyncIntervalSeconds)),
					testAccCheckHostSetPluginPreferredEndpoints(t, provider, fooSetName, updatePreferredEndpoints),
				),
			},
			importStep(fooSetName),
		},
	})
}

func testAccCheckHostSetPluginResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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

		hsrr, err := hostsetsClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading hostset %q: %v", id, err)
		}

		fmt.Println("found in existence check:", hsrr.GetItem().(*hostsets.HostSet).PreferredEndpoints)

		return nil
	}
}

func testAccCheckHostSetPluginResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_host_set":

				id := rs.Primary.ID

				hostsetsClient := hostsets.NewClient(md.client)

				_, err := hostsetsClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed host set %q: %v", id, apiErr)
				}

			default:
				continue
			}
		}
		return nil
	}
}

func testAccCheckHostSetPluginPreferredEndpoints(t *testing.T, testProvider *schema.Provider, name string, wantPreferredEndpoints []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("host set resource not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("host set resource ID is not set")
		}

		// check boundary to ensure it matches
		md := testProvider.Meta().(*metaData)
		client := md.client.Clone()

		hstClient := hostsets.NewClient(client)

		hs, err := hstClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading hostset %q: %v", id, err)
		}

		assert.Equal(t, wantPreferredEndpoints, hs.Item.PreferredEndpoints)

		return nil
	}
}