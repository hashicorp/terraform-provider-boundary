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
	projHostCatalog = fmt.Sprintf(`
resource "boundary_host_catalog" "foo" {
	name        = "foo"
	description = "%s"
	scope_id    = boundary_scope.proj1.id 
	type        = "static"
	depends_on  = [boundary_role.proj1_admin]
}`, fooHostCatalogDescription)

	projHostCatalogUpdate = fmt.Sprintf(`
resource "boundary_host_catalog" "foo" {
	name        = "foo"
	description = "%s"
	scope_id    = boundary_scope.proj1.id 
	type        = "static"
	depends_on  = [boundary_role.proj1_admin]
}`, fooHostCatalogDescriptionUpdate)
)

func TestAccHostCatalogCreate(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckHostCatalogResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, firstProjectFoo, projHostCatalog),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists(provider, "boundary_scope.org1"),
					testAccCheckScopeResourceExists(provider, "boundary_scope.proj1"),
					testAccCheckHostCatalogResourceExists(provider, "boundary_host_catalog.foo"),
					resource.TestCheckResourceAttr("boundary_host_catalog.foo", DescriptionKey, fooHostCatalogDescription),
				),
			},
			importStep("boundary_host_catalog.foo"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, projHostCatalogUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostCatalogResourceExists(provider, "boundary_host_catalog.foo"),
					resource.TestCheckResourceAttr("boundary_host_catalog.foo", DescriptionKey, fooHostCatalogDescriptionUpdate),
				),
			},
			importStep("boundary_host_catalog.foo"),
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
			case "boundary_host_catalog":

				id := rs.Primary.ID
				hcClient := hostcatalogs.NewClient(md.client)

				_, err := hcClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.ResponseStatus() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed host catalog %q: %v", id, err)
				}

			default:
				continue
			}
		}
		return nil
	}
}
