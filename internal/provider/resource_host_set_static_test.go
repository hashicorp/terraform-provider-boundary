// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/hostsets"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccHostSetStatic(t *testing.T) {
	t.Run("non-static", func(t *testing.T) {
		t.Parallel()
		testAccHostSetStatic(t, false)
	})
	t.Run("static", func(t *testing.T) {
		t.Parallel()
		testAccHostSetStatic(t, true)
	})
}

func testAccHostSetStatic(t *testing.T, static bool) {
	catalogBlock := ` 
	resource "%s" "foo" {
		%s
		scope_id    = boundary_scope.proj1.id
		depends_on  = [boundary_role.proj1_admin]
	}`

	hostBlock := `
	resource "%s" "foo" {
		host_catalog_id = %s.foo.id
		%s
		address         = "10.0.0.1"
	}`

	hostSetBlock := `
	resource "%s" "foo" {
		host_catalog_id    = %s.foo.id
		name               = "test"
		description        = "test hostset"
		%s
		host_ids           = [%s.foo.id%s]
	}`

	host2Block := `
	resource "%s" "bar" {
		host_catalog_id = %s.foo.id
		%s
		address         = "10.0.0.2"
	}`

	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	//	org := iam.TestOrg(t, tc.IamRepo())
	url := tc.ApiAddrs()[0]

	catalogName := "boundary_host_catalog"
	hostName := "boundary_host"
	setName := "boundary_host_set"
	typeStr := `type = "static"`
	if static {
		catalogName = "boundary_host_catalog_static"
		hostName = "boundary_host_static"
		setName = "boundary_host_set_static"
		typeStr = ""
	}
	fooSetName := fmt.Sprintf("%s.foo", setName)
	fooHostName := fmt.Sprintf("%s.foo", hostName)
	barHostName := fmt.Sprintf("%s.bar", hostName)
	hcBlock := fmt.Sprintf(catalogBlock, catalogName, typeStr)
	hBlock := fmt.Sprintf(hostBlock, hostName, catalogName, typeStr)
	h2Block := fmt.Sprintf(host2Block, hostName, catalogName, typeStr)
	hsBlock := fmt.Sprintf(hostSetBlock, setName, catalogName, typeStr, hostName, "")
	hsUpdateBlock := fmt.Sprintf(hostSetBlock, setName, catalogName, typeStr, hostName, fmt.Sprintf(`, %s.bar.id`, hostName))

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckHostSetStaticResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test project hostset create
				Config: testConfig(url, fooOrg, firstProjectFoo, hcBlock, hBlock, hsBlock),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostSetStaticResourceExists(provider, fooSetName),
					testAccCheckHostSetStaticHostIDsSet(provider, fooSetName, []string{fooHostName}),
					resource.TestCheckResourceAttr(fooSetName, "name", "test"),
					resource.TestCheckResourceAttr(fooSetName, "description", "test hostset"),
				),
			},
			importStep(fooSetName),
			{
				// test project hostset update
				Config: testConfig(url, fooOrg, firstProjectFoo, hcBlock, hBlock, h2Block, hsUpdateBlock),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostSetStaticResourceExists(provider, fooSetName),
					testAccCheckHostSetStaticHostIDsSet(provider, fooSetName, []string{fooHostName, barHostName}),
					resource.TestCheckResourceAttr(fooSetName, "name", "test"),
					resource.TestCheckResourceAttr(fooSetName, "description", "test hostset"),
				),
			},
			importStep(fooSetName),
		},
	})
}

func testAccCheckHostSetStaticResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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

		if _, err := hostsetsClient.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error when reading hostset %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckHostSetStaticHostIDsSet(testProvider *schema.Provider, name string, wantHostIDs []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("host set resource not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("host set resource ID is not set")
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

		hs, err := hstClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading hostset %q: %v", id, err)
		}

		if len(hs.Item.HostIds) == 0 {
			return fmt.Errorf("no hosts found on hostset %v; %v found in state; %#v in hs map", id, gotHostIDs, hs)
		}

		for _, stateHost := range hs.Item.HostIds {
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

func testAccCheckHostSetStaticResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
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
