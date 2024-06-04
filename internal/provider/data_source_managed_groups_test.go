package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/cap/oidc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	fooManagedGroupsDataMissingAuthMethodId = `
data "boundary_managed_groups" "foo" {}
`
	fooManagedGroupsData = `
data "boundary_managed_groups" "foo" {
	auth_method_id = boundary_auth_method_oidc.foo.id
}
`
)

func TestAccDataSourceManagedGroups(t *testing.T) {
	tp := oidc.StartTestProvider(t)
	defer tp.Stop()
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	tpCert := strings.TrimSpace(tp.CACert())
	createConfig := fmt.Sprintf(fooAuthMethodOidc, fooAuthMethodOidcDesc, tp.Addr(), tpCert)

	var provider *schema.Provider

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooManagedGroupsDataMissingAuthMethodId),
				ExpectError: regexp.MustCompile("auth_method_id: Invalid formatted identifier."),
			},
			{
				Config: testConfig(url, fooOrg, createConfig, fooManagedGroupsData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.boundary_managed_groups.foo", "auth_method_id"),
				),
			},
		},
	})
}
