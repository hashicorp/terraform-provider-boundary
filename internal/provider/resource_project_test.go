package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/hashicorp/watchtower/api"
	"github.com/hashicorp/watchtower/api/scopes"
	"github.com/hashicorp/watchtower/testing/controller"
	"github.com/stretchr/testify/assert"
)

func TestAccProjectCreation(t *testing.T) {
	url, cancel := controller.NewTestController(t, controller.WithDefaultOrgId("o_0000000000"))
	defer cancel()

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckProjectResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testSingleProjectConfig(url, firstProjectBar),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectResourceExists("watchtower_project.project1"),
					resource.TestCheckResourceAttr("watchtower_project.project1", "description", "bar"),
				),
			},
			{
				Config: testSingleProjectConfig(url, firstProjectFoo),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectResourceExists("watchtower_project.project1"),
					resource.TestCheckResourceAttr("watchtower_project.project1", "description", "foo"),
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
		o := scopes.Organization{
			Client: md.client,
		}
		if _, _, err := o.ReadProject(md.ctx, &scopes.Project{Id: id}); err != nil {
			return fmt.Errorf("Didn't receive a 404 when checking for cleaned up project %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckProjectResourceDestroy(s *terraform.State) error {
	// retrieve the connection established in Provider configuration
	md := testProvider.Meta().(*metaData)
	o := scopes.Organization{
		Client: md.client,
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "watchtower_project" {
			continue
		}
		id := rs.Primary.ID
		if _, _, err := o.ReadProject(md.ctx, &scopes.Project{Id: id}); err == nil || !strings.Contains(err.Error(), "404") {
			return fmt.Errorf("Error when reading created project %q: %v", id, err)
		}
	}
	return nil
}

func TestResourceDataToProject(t *testing.T) {
	nameKey, descKey := "name", "description"

	rp := resourceProject()
	testCases := []struct {
		name     string
		rData    map[string]interface{}
		expected *scopes.Project
	}{
		{
			name: "Fully populated",
			rData: map[string]interface{}{
				nameKey: "name",
				descKey: "desc",
			},
			expected: &scopes.Project{
				Name:        api.String("name"),
				Description: api.String("desc"),
			},
		},
		{
			name: "Name populated",
			rData: map[string]interface{}{
				nameKey: "name",
			},
			expected: &scopes.Project{
				Name: api.String("name"),
			},
		},
		{
			name: "Description populated",
			rData: map[string]interface{}{
				descKey: "desc",
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
			assert.Equal(t, tc.expected, resourceDataToProject(rd))
		})
	}
}

func TestProjectToResourceData(t *testing.T) {
	nameKey, descKey := "name", "description"

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
				nameKey: "name",
				descKey: "desc",
			},
		},
		{
			name: "Name populated",
			proj: &scopes.Project{
				Id:   "someid",
				Name: api.String("name"),
			},
			expected: map[string]interface{}{
				nameKey: "name",
			},
		},
		{
			name: "Description populated",
			proj: &scopes.Project{
				Id:          "someid",
				Description: api.String("desc"),
			},
			expected: map[string]interface{}{
				descKey: "desc",
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
			err := projectToResourceData(tc.proj, actual)
			assert.NoError(t, err)
			assert.Equal(t, expectedRd, actual)
		})
	}
}

func testTwoProjectConfig(url string) string {
	return fmt.Sprintf(`
provider "watchtower" {
  base_url = "%s"
  default_organization = "o_0000000000"
}

resource "watchtower_project" "project1" {
  description = "my description1"
}

resource "watchtower_project" "project2" {
  description = "my description2"
}
`, url)
}

const (
	firstProjectFoo = `
resource "watchtower_project" "project1" {
  description = "foo"
}`

	firstProjectBar = `
resource "watchtower_project" "project1" {
  description = "bar"
}`

	secondProject = `
resource "watchtower_project" "project1" {
  description = "project2"
}`
)

func testSingleProjectConfig(url, res string) string {
	return fmt.Sprintf(`
provider "watchtower" {
  base_url = "%s"
  default_organization = "o_0000000000"
}

%s
`, url, res)
}
