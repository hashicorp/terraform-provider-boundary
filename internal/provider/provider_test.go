package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/roles"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testProvider *schema.Provider
var testProviders map[string]*schema.Provider

var (
	tcLoginName = "user"
	tcPassword  = "passpass"
	tcPAUM      = "ampw_0000000000"
	tcConfig    = []controller.Option{
		controller.WithDefaultAuthMethodId(tcPAUM),
		controller.WithDefaultLoginName(tcLoginName),
		controller.WithDefaultPassword(tcPassword),
	}
)

func init() {
	testProvider = New()
	testProviders = map[string]*schema.Provider{
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

func setGrantScopeIdOnProject(scopeId, grantScopeId, principalId string, client *api.Client) error {
	roleClient := roles.NewClient(client)
	_, err, _ := roleClient.Create(
		context.Background(),
		scopeId,
		roles.WithName(fmt.Sprintf("TestRole_%s", grantScopeId)),
		roles.WithDescription(fmt.Sprintf("Terraform test management role for %s", grantScopeId)),
		roles.WithGrantScopeId(grantScopeId))

	return fmt.Errorf("%s", err.Message)
}

func TestProvider(t *testing.T) {
	if err := New().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
