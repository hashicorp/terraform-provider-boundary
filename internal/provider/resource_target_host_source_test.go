// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/targets"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const fooTargetHostSourceTemplate = `
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

resource "boundary_host_set" "foo" {
	name            = "foo"
	type            = "static"
	host_catalog_id = boundary_host_catalog.foo.id
	host_ids = [
		boundary_host.foo.id,
	]
}

resource "boundary_target" "foo" {
	name                = "test-target"
	type                = "tcp"
	scope_id            = boundary_scope.proj1.id
	default_port        = 22
	default_client_port = 1022
	depends_on          = [boundary_role.proj1_admin]
	lifecycle {
		ignore_changes = [host_source_ids]
	}
}

%s
`

const fooTargetHostSourceResource = `resource "boundary_target_host_source" "foo" {
	target_id      = boundary_target.foo.id
	host_source_id = boundary_host_set.foo.id
}
`

var (
	fooTargetHostSourceAttachment = fmt.Sprintf(fooTargetHostSourceTemplate, fooTargetHostSourceResource)
	fooTargetHostSourceDetached   = fmt.Sprintf(fooTargetHostSourceTemplate, "")
)

func TestAccTargetHostSource(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	t.Cleanup(tc.Shutdown)

	url := tc.ApiAddrs()[0]
	suffix := id.UniqueId()

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckTargetHostSourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg(suffix), firstProjectFoo, fooTargetHostSourceAttachment),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetHostSourceExists(provider, "boundary_target_host_source.foo"),
					resource.TestCheckResourceAttrSet("boundary_target_host_source.foo", targetHostSourceTargetIdKey),
					resource.TestCheckResourceAttrSet("boundary_target_host_source.foo", targetHostSourceIdKey),
					testAccCheckTargetHostSourceAttached(provider, "boundary_target.foo", "boundary_host_set.foo"),
				),
			},
			importStep("boundary_target_host_source.foo"),
			{
				Config: testConfig(url, fooOrg(suffix), firstProjectFoo, fooTargetHostSourceDetached),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetResourceHostSource(provider, "boundary_target.foo", nil),
				),
			},
		},
	})
}

func testAccCheckTargetHostSourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("target host source resource not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("target host source resource ID is not set")
		}

		targetID := rs.Primary.Attributes[targetHostSourceTargetIdKey]
		hostSourceID := rs.Primary.Attributes[targetHostSourceIdKey]
		if targetID == "" {
			return fmt.Errorf("target_id is not set")
		}
		if hostSourceID == "" {
			return fmt.Errorf("host_source_id is not set")
		}

		md := testProvider.Meta().(*metaData)
		targetClient := targets.NewClient(md.client)

		targetResource, err := targetClient.Read(context.Background(), targetID)
		if err != nil {
			return fmt.Errorf("error reading target %q: %v", targetID, err)
		}
		if targetResource == nil {
			return fmt.Errorf("target %q nil after read", targetID)
		}

		for _, id := range targetResource.Item.HostSourceIds {
			if id == hostSourceID {
				return nil
			}
		}

		return fmt.Errorf("host source %q not attached to target %q", hostSourceID, targetID)
	}
}

func testAccCheckTargetHostSourceAttached(testProvider *schema.Provider, targetName, hostSourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		targetResource, ok := s.RootModule().Resources[targetName]
		if !ok {
			return fmt.Errorf("target resource not found: %s", targetName)
		}
		hostSourceResource, ok := s.RootModule().Resources[hostSourceName]
		if !ok {
			return fmt.Errorf("host source resource not found: %s", hostSourceName)
		}

		targetID := targetResource.Primary.ID
		hostSourceID := hostSourceResource.Primary.ID
		if targetID == "" {
			return fmt.Errorf("target resource ID is not set")
		}
		if hostSourceID == "" {
			return fmt.Errorf("host source resource ID is not set")
		}

		md := testProvider.Meta().(*metaData)
		targetClient := targets.NewClient(md.client)

		targetResourceState, err := targetClient.Read(context.Background(), targetID)
		if err != nil {
			return fmt.Errorf("error reading target %q: %v", targetID, err)
		}
		if targetResourceState == nil {
			return fmt.Errorf("target %q nil after read", targetID)
		}

		for _, id := range targetResourceState.Item.HostSourceIds {
			if id == hostSourceID {
				return nil
			}
		}

		return fmt.Errorf("host source %q not attached to target %q", hostSourceID, targetID)
	}
}

func testAccCheckTargetHostSourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "boundary_target_host_source" {
				continue
			}

			targetID := rs.Primary.Attributes[targetHostSourceTargetIdKey]
			hostSourceID := rs.Primary.Attributes[targetHostSourceIdKey]
			if targetID == "" || hostSourceID == "" {
				continue
			}

			md := testProvider.Meta().(*metaData)
			targetClient := targets.NewClient(md.client)

			targetResource, err := targetClient.Read(context.Background(), targetID)
			if err != nil {
				if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
					continue
				}
				return fmt.Errorf("error reading target %q while checking destroy: %v", targetID, err)
			}
			if targetResource == nil {
				continue
			}

			for _, id := range targetResource.Item.HostSourceIds {
				if id == hostSourceID {
					return fmt.Errorf("host source %q still attached to target %q after destroy", hostSourceID, targetID)
				}
			}
		}

		return nil
	}
}
