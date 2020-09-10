package provider

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api/hostcatalogs"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
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
	scope_id    = boundary_project.foo.id 
	type        = "static"
}`, fooHostCatalogDescription)

	projHostCatalogUpdate = fmt.Sprintf(`
resource "boundary_host_catalog" "foo" {
  name        = "foo"
	description = "%s"
	scope_id    = boundary_project.foo.id 
	type        = "static"
}`, fooHostCatalogDescriptionUpdate)
)

func TestAccHostCatalogCreate(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckHostCatalogResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, fooProject, projHostCatalog),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrganizationResourceExists("boundary_organization.foo"),
					testAccCheckProjectResourceExists("boundary_project.foo"),
					testAccCheckHostCatalogResourceExists("boundary_host_catalog.foo"),
					resource.TestCheckResourceAttr("boundary_host_catalog.foo", hostCatalogDescriptionKey, fooHostCatalogDescription),
				),
			},
			{
				// test update
				Config: testConfig(url, fooOrg, fooProject, projHostCatalogUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostCatalogResourceExists("boundary_host_catalog.foo"),
					resource.TestCheckResourceAttr("boundary_host_catalog.foo", hostCatalogDescriptionKey, fooHostCatalogDescriptionUpdate),
				),
			},
		},
	})
}

func testAccCheckHostCatalogResourceExists(name string) resource.TestCheckFunc {
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
		projID, ok := rs.Primary.Attributes["scope_id"]
		if !ok {
			return fmt.Errorf("scope_id is not set")
		}
		hcClient := hostcatalogs.NewClient(md.client)

		if _, _, err := hcClient.Read(md.ctx, id, hostcatalogs.WithScopeId(projID)); err != nil {
			return fmt.Errorf("Got an error when reading host catalog %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckHostCatalogResourceDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_project":
				continue
			case "boundary_host_catalog":

				id := rs.Primary.ID
				projID, ok := rs.Primary.Attributes["scope_id"]
				if !ok {
					return fmt.Errorf("scope_id is not set")
				}

				hcClient := hostcatalogs.NewClient(md.client)

				_, apiErr, _ := hcClient.Read(md.ctx, id, hostcatalogs.WithScopeId(projID))
				if apiErr == nil || apiErr.Status != http.StatusNotFound && apiErr.Status != http.StatusForbidden {
					return fmt.Errorf("Didn't get a 404 or 403 when reading destroyed host catalog %q: %v", id, apiErr)
				}

			default:
				t.Logf("Got unknown resource type %q", rs.Type)
			}
		}
		return nil
	}
}

func TestHostCatalogValidateType(t *testing.T) {
	cases := []struct {
		name   string
		config string
		numErr int
		numWrn int
	}{
		{
			name:   "should validate when correct values are passed",
			config: hostCatalogTypeStatic,
			numErr: 0,
			numWrn: 0,
		},
		{
			name:   "should error when incorrect values are passed",
			config: "nope",
			numErr: 1,
			numWrn: 0,
		}}

	for _, tCase := range cases {
		t.Run(tCase.name, func(t *testing.T) {
			wrn, err := validateHostCatalogType(tCase.config, "")
			assert.Len(t, wrn, tCase.numWrn)
			assert.Len(t, err, tCase.numErr)
		})
	}
}
