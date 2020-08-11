package provider

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/hashicorp/boundary/api/hosts"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/stretchr/testify/assert"
)

const (
	fooHostCatalogDescription       = "bar"
	fooHostCatalogDescriptionUpdate = "foo bar"
)

var (
	fooHostCatalog = fmt.Sprintf(`
resource "boundary_host_catalog" "foo" {
	description = "%s"
	project_id = boundary_project.project1.id 
	type = "Static"
}`, fooHostCatalogDescription)

	fooHostCatalogUpdate = fmt.Sprintf(`
resource "boundary_host_catalog" "foo" {
	description = "%s"
	project_id = boundary_project.project1.id 
	type = "Static"
}`, fooHostCatalogDescriptionUpdate)
)

func TestAccHostCatalogCreate(t *testing.T) {
	tc := controller.NewTestController(t, controller.WithDefaultOrgId("o_0000000000"))
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckHostCatalogResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, firstProjectBar, fooHostCatalog),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectResourceExists("boundary_project.project1"),
					testAccCheckHostCatalogResourceExists("boundary_host_catalog.foo"),
					resource.TestCheckResourceAttr("boundary_host_catalog.foo", hostCatalogDescriptionKey, fooHostCatalogDescription),
				),
			},
			{
				// test update
				Config: testConfig(url, firstProjectBar, fooHostCatalogUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostCatalogResourceExists("boundary_host_catalog.foo"),
					resource.TestCheckResourceAttr("boundary_host_catalog.foo", hostCatalogDescriptionKey, fooHostCatalogDescriptionUpdate),
				),
			},
			{
				// test destroy
				Config: testConfig(url, firstProjectBar),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectResourceExists("boundary_project.project1"),
					testAccCheckHostCatalogDestroyed("boundary_host_catalog.foo"),
				),
			},
		},
	})
}

// testAccCheckHostCatalogDestroyed checks the terraform state for the host
// catalog and returns an error if found.
//
// TODO(malnick) This method falls short of checking the Boundary API for
// the resource if the resource is not found in state. This is due to us not
// having the host catalog ID, but it doesn't guarantee that the resource was
// successfully removed.
//
// It does check Boundary if the resource is found in state to point out any
// misalignment between what is in state and the actual configuration.
func testAccCheckHostCatalogDestroyed(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			// If it's not in state, it's destroyed in TF but not guaranteed to be destroyed
			// in Boundary. Need to find a way to get the host catalog ID here so we can
			// form a lookup to the WT API to check this.
			return nil
		}
		errs := []string{}
		errs = append(errs, fmt.Sprintf("Found host catalog resource in state: %s", name))

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("No ID is set")
		}

		md := testProvider.Meta().(*metaData)

		h := hosts.HostCatalog{Id: id}

		p := &scopes.Project{
			Client: md.client,
			Id:     rs.Primary.Attributes["project_id"],
		}

		if _, apiErr, _ := p.ReadHostCatalog(md.ctx, &h); apiErr == nil || apiErr.Status != http.StatusNotFound {
			errs = append(errs, fmt.Sprintf("Host catalog not destroyed %q: %v", id, apiErr))
		}

		return errors.New(strings.Join(errs, ","))
	}
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

		h := hosts.HostCatalog{Id: id}

		p := &scopes.Project{
			Client: md.client,
			Id:     rs.Primary.Attributes["project_id"],
		}

		if _, _, err := p.ReadHostCatalog(md.ctx, &h); err != nil {
			return fmt.Errorf("Got an error when reading host catalog %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckHostCatalogResourceDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		md := testProvider.Meta().(*metaData)
		client := md.client

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_project":
				continue
			case "boundary_host_catalog":

				id := rs.Primary.ID
				h := hosts.HostCatalog{Id: id}

				p := &scopes.Project{
					Client: client,
					Id:     rs.Primary.Attributes["project_id"],
				}

				_, apiErr, _ := p.ReadHostCatalog(md.ctx, &h)
				if apiErr == nil || apiErr.Status != http.StatusNotFound {
					return fmt.Errorf("Didn't get a 404 when reading destroyed host catalog %q: %v", id, apiErr)
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
