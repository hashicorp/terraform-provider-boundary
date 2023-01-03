// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/hostcatalogs"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	fooHostCatalogDescription       = "bar"
	fooHostCatalogDescriptionUpdate = "foo bar"
)

var (
	projHostCatalog = `
resource "%s" "foo" {
	name        = "foo"
	description = "bar"
	scope_id    = boundary_scope.proj1.id 
	%s
	depends_on  = [boundary_role.proj1_admin]
}`

	projHostCatalogUpdate = `
resource "%s" "foo" {
	name        = "foo"
	description = "foo bar"
	scope_id    = boundary_scope.proj1.id 
	%s
	depends_on  = [boundary_role.proj1_admin]
}`
)

func TestAccHostCatalogCreate(t *testing.T) {
	t.Run("non-static", func(t *testing.T) {
		t.Parallel()
		testAccHostCatalog(t, false)
	})
	t.Run("static", func(t *testing.T) {
		t.Parallel()
		testAccHostCatalog(t, true)
	})
}

func testAccHostCatalog(t *testing.T, static bool) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resName := "boundary_host_catalog"
	typeStr := `type = "static"`
	if static {
		resName = "boundary_host_catalog_static"
		typeStr = ""
	}
	hc, hcu := fmt.Sprintf(projHostCatalog, resName, typeStr), fmt.Sprintf(projHostCatalogUpdate, resName, typeStr)
	fooName := fmt.Sprintf("%s.foo", resName)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckHostCatalogResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, firstProjectFoo, hc),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists(provider, "boundary_scope.org1"),
					testAccCheckScopeResourceExists(provider, "boundary_scope.proj1"),
					testAccCheckHostCatalogResourceExists(provider, fooName),
					resource.TestCheckResourceAttr(fooName, DescriptionKey, fooHostCatalogDescription),
				),
			},
			importStep(fooName),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, hcu),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostCatalogResourceExists(provider, fooName),
					resource.TestCheckResourceAttr(fooName, DescriptionKey, fooHostCatalogDescriptionUpdate),
				),
			},
			importStep(fooName),
		},
	})
}

func testAccCheckHostCatalogResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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
		hcClient := hostcatalogs.NewClient(md.client)

		if _, err := hcClient.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error when reading host catalog %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckHostCatalogResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_host_catalog", "boundary_host_catalog_static":

				id := rs.Primary.ID
				hcClient := hostcatalogs.NewClient(md.client)

				_, err := hcClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed host catalog %q: %v", id, err)
				}

			default:
				continue
			}
		}
		return nil
	}
}
