package provider

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/stretchr/testify/assert"
)

func TestAccProjectCreation(t *testing.T) {
	tc := controller.NewTestController(t, controller.WithDefaultOrgId("o_0000000000"))
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckProjectResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, firstProjectBar, secondProject),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectResourceExists("boundary_project.project1"),
					resource.TestCheckResourceAttr("boundary_project.project1", projectDescriptionKey, "bar"),
					resource.TestCheckResourceAttr("boundary_project.project2", projectDescriptionKey, "project2"),
				),
			},
			// Updates the first project to have description foo
			{
				Config: testConfig(url, firstProjectFoo, secondProject),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectResourceExists("boundary_project.project1"),
					resource.TestCheckResourceAttr("boundary_project.project1", projectDescriptionKey, "foo"),
					resource.TestCheckResourceAttr("boundary_project.project2", projectDescriptionKey, "project2"),
				),
			},
			// Remove second project
			{
				Config: testConfig(url, firstProjectFoo),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectResourceExists("boundary_project.project1"),
					resource.TestCheckResourceAttr("boundary_project.project1", projectDescriptionKey, "foo"),
				),
			},
		},
	})
}

func testAccCheckProjectResourceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("No ID is set")
		}

		if !strings.HasPrefix(id, "p_") {
			return fmt.Errorf("ID not formatted as expected")
		}
		md := testProvider.Meta().(*metaData)
		o := scopes.Org{
			Client: md.client,
		}
		if _, _, err := o.ReadProject(md.ctx, &scopes.Project{Id: id}); err != nil {
			return fmt.Errorf("Got an error when reading project %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckProjectResourceDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// retrieve the connection established in Provider configuration
		md := testProvider.Meta().(*metaData)
		o := scopes.Org{
			Client: md.client,
		}

		for _, rs := range s.RootModule().Resources {
			id := rs.Primary.ID
			switch rs.Type {
			case "boundary_project":
				if _, apiErr, _ := o.ReadProject(md.ctx, &scopes.Project{Id: id}); apiErr == nil || apiErr.Status != http.StatusNotFound {
					return fmt.Errorf("Didn't get a 404 when reading destroyed project %q: %v", id, apiErr)
				}
			default:
				t.Logf("Got unknown resource type %q", rs.Type)
			}
		}
		return nil
	}
}

func TestResourceDataToProject(t *testing.T) {
	rp := resourceProject()
	testCases := []struct {
		name     string
		rData    map[string]interface{}
		expected *scopes.Project
	}{
		{
			name: "Fully populated",
			rData: map[string]interface{}{
				projectNameKey:        "name",
				projectDescriptionKey: "desc",
			},
			expected: &scopes.Project{
				Name:        api.String("name"),
				Description: api.String("desc"),
			},
		},
		{
			name: "Name populated",
			rData: map[string]interface{}{
				projectNameKey: "name",
			},
			expected: &scopes.Project{
				Name: api.String("name"),
			},
		},
		{
			name: "Description populated",
			rData: map[string]interface{}{
				projectDescriptionKey: "desc",
			},
			expected: &scopes.Project{
				Description: api.String("desc"),
			},
		},
		{
			name:     "Not populated",
			rData:    map[string]interface{}{},
			expected: &scopes.Project{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := rp.TestResourceData()
			for k, v := range tc.rData {
				err := rd.Set(k, v)
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expected, convertResourceDataToProject(rd))
		})
	}
}

func TestProjectToResourceData(t *testing.T) {
	rp := resourceProject()
	testCases := []struct {
		name     string
		expected map[string]interface{}
		proj     *scopes.Project
	}{
		{
			name: "Fully populated",
			proj: &scopes.Project{
				Id:          "someid",
				Name:        api.String("name"),
				Description: api.String("desc"),
			},
			expected: map[string]interface{}{
				projectNameKey:        "name",
				projectDescriptionKey: "desc",
			},
		},
		{
			name: "Name populated",
			proj: &scopes.Project{
				Id:   "someid",
				Name: api.String("name"),
			},
			expected: map[string]interface{}{
				projectNameKey: "name",
			},
		},
		{
			name: "Description populated",
			proj: &scopes.Project{
				Id:          "someid",
				Description: api.String("desc"),
			},
			expected: map[string]interface{}{
				projectDescriptionKey: "desc",
			},
		},
		{
			name: "Not populated",
			proj: &scopes.Project{
				Id: "someid",
			},
			expected: map[string]interface{}{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expectedRd := rp.TestResourceData()
			expectedRd.SetId("someid")
			for k, v := range tc.expected {
				err := expectedRd.Set(k, v)
				assert.NoError(t, err)
			}

			actual := rp.TestResourceData()
			err := convertProjectToResourceData(tc.proj, actual)
			assert.NoError(t, err)
			assert.Equal(t, expectedRd, actual)
		})
	}
}

const (
	firstProjectFoo = `
resource "boundary_project" "project1" {
  description = "foo"
}`

	firstProjectBar = `
resource "boundary_project" "project1" {
  description = "bar"
}`

	secondProject = `
resource "boundary_project" "project2" {
  description = "project2"
}
`
)
