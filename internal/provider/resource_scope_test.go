// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	fooOrg = `
resource "boundary_scope" "global" {
	global_scope = true
	name = "global"
	description = "Global Scope"
	scope_id = "global"
}

resource "boundary_scope" "org1" {
	name = "org1"
	scope_id = boundary_scope.global.id
}

resource "boundary_role" "org1_admin" {
	scope_id = boundary_scope.global.id
	grant_scope_id = boundary_scope.org1.id
	grant_strings = ["ids=*;type=*;actions=*"]
	principal_ids = ["u_auth"]
}
`

	firstProjectFoo = `
resource "boundary_scope" "proj1" {
	name = "proj1"
	scope_id    = boundary_scope.org1.id
	description = "foo"
	depends_on = [boundary_role.org1_admin]
}

resource "boundary_role" "proj1_admin" {
	scope_id = boundary_scope.org1.id
	grant_scope_id = boundary_scope.proj1.id
	grant_strings = ["ids=*;type=*;actions=*"]
	principal_ids = ["u_auth"]
}
`

	firstProjectBar = `
resource "boundary_scope" "proj1" {
	name = "proj1"
	scope_id    = boundary_scope.org1.id
	description = "bar"
	depends_on = [boundary_role.org1_admin]
}

resource "boundary_role" "proj1_admin" {
	scope_id = boundary_scope.org1.id
	grant_scope_id = boundary_scope.proj1.id
	grant_strings = ["ids=*;type=*;actions=*"]
	principal_ids = ["u_auth"]
}
`
	secondProject = `
resource "boundary_scope" "proj2" {
	name = "proj2"
	scope_id    = boundary_scope.org1.id
	description = "project2"
	depends_on = [boundary_role.org1_admin]
}

resource "boundary_role" "proj2_admin" {
	scope_id = boundary_scope.org1.id
	grant_scope_id = boundary_scope.proj2.id
	grant_strings = ["ids=*;type=*;actions=*"]
	principal_ids = ["u_auth"]
}
`
)

func TestAccScopeCreation(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckScopeResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg, firstProjectFoo, secondProject),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists(provider, "boundary_scope.org1"),
					testAccCheckScopeResourceExists(provider, "boundary_scope.proj1"),
					resource.TestCheckResourceAttr("boundary_scope.proj1", DescriptionKey, "foo"),
					resource.TestCheckResourceAttr("boundary_scope.proj2", DescriptionKey, "project2"),
				),
			},
			importStep("boundary_scope.org1"),
			importStep("boundary_scope.proj1"),
			// Updates the first project to have description bar
			{
				Config: testConfig(url, fooOrg, firstProjectBar, secondProject),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists(provider, "boundary_scope.proj1"),
					resource.TestCheckResourceAttr("boundary_scope.proj1", DescriptionKey, "bar"),
					resource.TestCheckResourceAttr("boundary_scope.proj2", DescriptionKey, "project2"),
				),
			},
			importStep("boundary_scope.proj1"),
			// Remove second project
			{
				Config: testConfig(url, fooOrg, firstProjectBar),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists(provider, "boundary_scope.proj1"),
					resource.TestCheckResourceAttr("boundary_scope.proj1", DescriptionKey, "bar"),
				),
			},
			importStep("boundary_scope.proj1"),
		},
	})
}

func testAccCheckScopeResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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
		scp := scopes.NewClient(md.client)

		_, err := scp.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading scope %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckScopeResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// retrieve the connection established in Provider configuration
		md := testProvider.Meta().(*metaData)
		scp := scopes.NewClient(md.client)

		for _, rs := range s.RootModule().Resources {
			id := rs.Primary.ID
			switch rs.Type {
			case "boundary_scope":
				if rs.Primary.Attributes["global_scope"] == "true" {
					// skip resource, its the global scope
					continue
				}
				_, err := scp.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed resource %q: %w", id, err)
				}
			default:
				continue
			}
		}
		return nil
	}
}
