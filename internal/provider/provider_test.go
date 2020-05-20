package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var testProvider *schema.Provider
var testProviders map[string]terraform.ResourceProvider

func init() {
	// Always run acceptance tests since our backend is in memory.
	os.Setenv("TF_ACC", "true")

	testProvider = New().(*schema.Provider)
	testProviders = map[string]terraform.ResourceProvider{
		"watchtower": testProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := New().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
