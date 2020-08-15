package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/roles"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var testProvider *schema.Provider
var testProviders map[string]terraform.ResourceProvider

var fooProject = `
resource "boundary_project" "foo" {
  name = "test"
}`

var (
	tcUsername = "user"
	tcPassword = "passpass"
	tcPAUM     = "paum_0000000000"
	tcOrg      = "o_0000000000"
	tcConfig   = []controller.Option{
		controller.WithDefaultOrgId(tcOrg),
		controller.WithDefaultAuthMethodId(tcPAUM),
		controller.WithDefaultLoginName(tcUsername),
		controller.WithDefaultPassword(tcPassword),
	}
)

func init() {
	testProvider = New().(*schema.Provider)
	testProviders = map[string]terraform.ResourceProvider{
		"boundary": testProvider,
	}
}

func testConfig(url string, res ...string) string {
	provider := fmt.Sprintf(`
provider "boundary" {
  base_url             = "%s"
  default_scope        = "%s"
	auth_method_id       = "%s"
	auth_method_username = "%s"
	auth_method_password = "%s"
}`, url, tcOrg, tcPAUM, tcUsername, tcPassword)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func setGrantScopeIDonProject(projID string, principalID string, client *api.Client) error {
	roleClient := roles.NewRolesClient(client)
	_, err, _ := roleClient.Create(
		context.Background(),
		roles.WithName(fmt.Sprintf("TestRole_%s", projID)),
		roles.WithDescription(fmt.Sprintf("Terraform test management role for %s", projID)),
		roles.WithGrantScopeId(projID))

	return fmt.Errorf("%s", err.Message)
}

func TestProvider(t *testing.T) {
	if err := New().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
