package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/hashicorp/watchtower/testing/controller"
)

var testProvider *schema.Provider
var testProviders map[string]terraform.ResourceProvider

func init() {
	testProvider = New().(*schema.Provider)
	testProviders = map[string]terraform.ResourceProvider{
		"watchtower": testProvider,
	}
}

func testConfig(url string, res ...string) string {
	provider := fmt.Sprintf(`
provider "watchtower" {
  base_url             = "%s"
  default_organization = "o_0000000000"
	auth_method_id       = "am_1234567890"
	auth_method_username = "foo"
	auth_method_password = "bar"
}`, url)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func TestProvider(t *testing.T) {
	if err := New().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func getControllerOptions() []controller.Option {
	var databaseURL = os.Getenv("WATCHTOWER_DATABASE_URL")
	options := []controller.Option{controller.WithDefaultOrgId("o_0000000000")}

	if databaseURL != "" {
		options = append(options, controller.DisableDatabaseCreation(), controller.WithDatabaseURL(databaseURL))
	}

	return options
}
