package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	fooAuthTokensData = `
data "boundary_auth_tokens" "foo" {}
`
)

func TestAccDataAuthTokens(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()

	url := tc.ApiAddrs()[0]

	var provider *schema.Provider

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooAuthTokensData),
				ExpectError: regexp.MustCompile("scope_id: This field must be 'global' or a valid org scope id."),
			},
		},
	})
}
