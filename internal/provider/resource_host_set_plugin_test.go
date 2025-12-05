// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/boundary/api/hostsets"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var currentPluginHostSetAttributesValue string

func TestAccHostSetPlugin(t *testing.T) {
	t.Parallel()

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
		%s
	}`

	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]
	fooSetName := "boundary_host_set_plugin.foo"

	attrBlock1 := `attributes_json = jsonencode({
		foo = "bar"
		zip = "zap"
	})`
	attrBlock2 := `attributes_json = jsonencode({
		foo = "bar"
		zip = "zoop"
	})`
	attrBlock3 := `attributes_json = jsonencode({
		foo = "bar"
		zip = "zoop"
	})`
	attrBlock4 := ``
	attrBlock5 := `attributes_json = jsonencode({
		foo = "bar"
		zip = "zoop"
	})`
	attrBlock6 := `attributes_json = "null"`

	hsBlock := fmt.Sprintf(hostSetBlock, initialPreferredEndpointsStr, initialSyncIntervalSeconds, attrBlock1)
	hsUpdate1Block := fmt.Sprintf(hostSetBlock, updatePreferredEndpointsStr, updateSyncIntervalSeconds, attrBlock2)
	hsUpdate2Block := fmt.Sprintf(hostSetBlock, updatePreferredEndpointsStr, updateSyncIntervalSeconds, attrBlock3)
	hsUpdate3Block := fmt.Sprintf(hostSetBlock, updatePreferredEndpointsStr, updateSyncIntervalSeconds, attrBlock4)
	hsUpdate4Block := fmt.Sprintf(hostSetBlock, updatePreferredEndpointsStr, updateSyncIntervalSeconds, attrBlock5)
	hsUpdate5Block := fmt.Sprintf(hostSetBlock, updatePreferredEndpointsStr, updateSyncIntervalSeconds, attrBlock6)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckHostSetResourceDestroy(t, provider, baseHostSetType),
		Steps: []resource.TestStep{
			{
				// test project hostset create
				Config: testConfig(url, fooOrg, firstProjectFoo, catalogBlock, hsBlock),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostSetPluginResourceExists(provider, fooSetName, expectedAttributesStatePreviouslyEmptyNowSet),
					resource.TestCheckResourceAttr(fooSetName, NameKey, "test"),
					resource.TestCheckResourceAttr(fooSetName, DescriptionKey, "test hostset"),
					resource.TestCheckResourceAttr(fooSetName, SyncIntervalSecondsKey, fmt.Sprintf("%d", initialSyncIntervalSeconds)),
					testAccCheckHostSetPluginPreferredEndpoints(t, provider, fooSetName, initialPreferredEndpoints),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(fooSetName),
			{
				// test project hostset update
				Config: testConfig(url, fooOrg, firstProjectFoo, catalogBlock, hsUpdate1Block),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostSetPluginResourceExists(provider, fooSetName, expectedAttributesStatePreviouslySetButChanged),
					resource.TestCheckResourceAttr(fooSetName, NameKey, "test"),
					resource.TestCheckResourceAttr(fooSetName, DescriptionKey, "test hostset"),
					resource.TestCheckResourceAttr(fooSetName, SyncIntervalSecondsKey, fmt.Sprintf("%d", updateSyncIntervalSeconds)),
					testAccCheckHostSetPluginPreferredEndpoints(t, provider, fooSetName, updatePreferredEndpoints),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(fooSetName),
			{
				// test project hostset update
				Config: testConfig(url, fooOrg, firstProjectFoo, catalogBlock, hsUpdate2Block),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostSetPluginResourceExists(provider, fooSetName, expectedAttributesStatePreviouslySetNoChange),
					resource.TestCheckResourceAttr(fooSetName, NameKey, "test"),
					resource.TestCheckResourceAttr(fooSetName, DescriptionKey, "test hostset"),
					resource.TestCheckResourceAttr(fooSetName, SyncIntervalSecondsKey, fmt.Sprintf("%d", updateSyncIntervalSeconds)),
					testAccCheckHostSetPluginPreferredEndpoints(t, provider, fooSetName, updatePreferredEndpoints),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(fooSetName),
			{
				// test project hostset update
				Config: testConfig(url, fooOrg, firstProjectFoo, catalogBlock, hsUpdate3Block),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostSetPluginResourceExists(provider, fooSetName, expectedAttributesStatePreviouslySetNowEmpty),
					resource.TestCheckResourceAttr(fooSetName, NameKey, "test"),
					resource.TestCheckResourceAttr(fooSetName, DescriptionKey, "test hostset"),
					resource.TestCheckResourceAttr(fooSetName, SyncIntervalSecondsKey, fmt.Sprintf("%d", updateSyncIntervalSeconds)),
					testAccCheckHostSetPluginPreferredEndpoints(t, provider, fooSetName, updatePreferredEndpoints),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(fooSetName),
			{
				// test project hostset update
				Config: testConfig(url, fooOrg, firstProjectFoo, catalogBlock, hsUpdate4Block),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostSetPluginResourceExists(provider, fooSetName, expectedAttributesStatePreviouslyEmptyNowSet),
					resource.TestCheckResourceAttr(fooSetName, NameKey, "test"),
					resource.TestCheckResourceAttr(fooSetName, DescriptionKey, "test hostset"),
					resource.TestCheckResourceAttr(fooSetName, SyncIntervalSecondsKey, fmt.Sprintf("%d", updateSyncIntervalSeconds)),
					testAccCheckHostSetPluginPreferredEndpoints(t, provider, fooSetName, updatePreferredEndpoints),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(fooSetName),
			{
				// test project hostset update
				Config: testConfig(url, fooOrg, firstProjectFoo, catalogBlock, hsUpdate5Block),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostSetPluginResourceExists(provider, fooSetName, expectedAttributesStatePreviouslySetNowEmpty),
					resource.TestCheckResourceAttr(fooSetName, NameKey, "test"),
					resource.TestCheckResourceAttr(fooSetName, DescriptionKey, "test hostset"),
					resource.TestCheckResourceAttr(fooSetName, SyncIntervalSecondsKey, fmt.Sprintf("%d", updateSyncIntervalSeconds)),
					testAccCheckHostSetPluginPreferredEndpoints(t, provider, fooSetName, updatePreferredEndpoints),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(fooSetName),
		},
	})
}

func testAccCheckHostSetPluginResourceExists(testProvider *schema.Provider, name string, expAttrs expectedAttributesState) resource.TestCheckFunc {
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

		attrs := hsrr.GetResponse().Map["attributes"]
		switch expAttrs {
		case expectedAttributesStatePreviouslyEmptyNowSet:
			if currentPluginHostSetAttributesValue != "" {
				return fmt.Errorf("expected no previous attributes value, got %s", currentPluginHostSetAttributesValue)
			}
			if attrs == nil {
				return errors.New("expected non-empty new attributes value")
			}
			attrsVal, err := json.Marshal(attrs)
			if err != nil {
				return fmt.Errorf("error marshaling attrs: %w", err)
			}
			currentPluginHostSetAttributesValue = string(attrsVal)

		case expectedAttributesStatePreviouslySetButChanged:
			if currentPluginHostSetAttributesValue == "" {
				return errors.New("expected previous attributes value")
			}
			if attrs == nil {
				return errors.New("expected non-empty new attributes value")
			}
			attrsVal, err := json.Marshal(attrs)
			if err != nil {
				return fmt.Errorf("error marshaling attrs: %w", err)
			}
			if string(attrsVal) == currentPluginHostSetAttributesValue {
				return errors.New("expected changed attrs value")
			}
			currentPluginHostSetAttributesValue = string(attrsVal)

		case expectedAttributesStatePreviouslySetNoChange:
			if currentPluginHostSetAttributesValue == "" {
				return errors.New("expected previous attributes value")
			}
			if attrs == nil {
				return errors.New("expected non-empty new attributes value")
			}
			attrsVal, err := json.Marshal(attrs)
			if err != nil {
				return fmt.Errorf("error marshaling attrs: %w", err)
			}
			if string(attrsVal) == "" {
				return errors.New("expected non-empty new attributes value")
			}
			if string(attrsVal) != currentPluginHostSetAttributesValue {
				return errors.New("expected same attributes value")
			}

		case expectedAttributesStatePreviouslySetNowEmpty:
			if currentPluginHostSetAttributesValue == "" {
				return errors.New("expected previous attributes value")
			}
			if attrs != nil {
				return fmt.Errorf("expected empty new attributes value, got %s", attrs)
			}
			currentPluginHostSetAttributesValue = ""
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
