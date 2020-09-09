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

var (
	tcLoginName = "user"
	tcPassword  = "passpass"
	tcPAUM      = "ampw_0000000000"
	tcScope     = "global"
	tcConfig    = []controller.Option{
		controller.WithDefaultAuthMethodId(tcPAUM),
		controller.WithDefaultLoginName(tcLoginName),
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
	auth_method_id       = "%s"
	password_auth_method_login_name = "%s"
	password_auth_method_password = "%s"
}`, url, tcPAUM, tcLoginName, tcPassword)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func setGrantScopeIdOnProject(projId string, principalId string, client *api.Client) error {
	roleClient := roles.NewClient(client)
	_, err, _ := roleClient.Create(
		context.Background(),
		tcScope,
		roles.WithName(fmt.Sprintf("TestRole_%s", projId)),
		roles.WithDescription(fmt.Sprintf("Terraform test management role for %s", projId)),
		roles.WithGrantScopeId(projId))

	return fmt.Errorf("%s", err.Message)
}

func TestProvider(t *testing.T) {
	if err := New().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
