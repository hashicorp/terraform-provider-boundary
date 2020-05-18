package provider

import (
	"testing"

	"github.com/hashicorp/watchtower/api"
	"github.com/hashicorp/watchtower/api/scopes"
	"github.com/stretchr/testify/assert"
)

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
