package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/boundary/testing/vault"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	fooTargetsDataMissingScope = `
data "boundary_targets" "foo" {}
`
	fooTargetsData = `
data "boundary_targets" "foo" {
	scope_id = boundary_target.foo.scope_id
}
`
)

func TestAccDataSourceTargets(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	vc := vault.NewTestVaultServer(t)
	_, token := vc.CreateToken(t)
	credStoreRes := vaultCredStoreResource(vc,
		vaultCredStoreName,
		vaultCredStoreDesc,
		vaultCredStoreNamespace,
		"www.original.com",
		token,
		true)

	var provider *schema.Provider

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config:      testConfig(url, fooTargetsDataMissingScope),
				ExpectError: regexp.MustCompile("scope_id: This field must be a valid project scope ID or the list operation.*\n.*must be recursive."),
			},
			{
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, fooBarCredLibs, fooBarHostSet, fooTarget, fooTargetsData),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "id", "boundary-targets"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.%", "21"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.application_credential_libraries.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.application_credential_library_ids.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.application_credential_source_ids.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.application_credential_sources.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.authorized_actions.#", "11"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.authorized_actions.0", "no-op"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.authorized_actions.1", "read"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.authorized_actions.10", "authorize-session"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.authorized_actions.2", "update"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.authorized_actions.3", "delete"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.authorized_actions.4", "add-host-sets"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.authorized_actions.5", "set-host-sets"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.authorized_actions.6", "remove-host-sets"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.authorized_actions.7", "add-credential-libraries"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.authorized_actions.8", "set-credential-libraries"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.authorized_actions.9", "remove-credential-libraries"),
					resource.TestCheckResourceAttrSet("data.boundary_targets.foo", "items.0.created_time"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.description", "bar"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.host_set_ids.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.host_sets.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.host_source_ids.#", "0"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.host_sources.#", "0"),
					resource.TestCheckResourceAttrSet("data.boundary_targets.foo", "items.0.id"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.name", "test"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.scope.#", "1"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.scope.0.%", "5"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.scope.0.description", "foo"),
					resource.TestCheckResourceAttrSet("data.boundary_targets.foo", "items.0.scope.0.id"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.scope.0.name", "proj1"),
					resource.TestCheckResourceAttrSet("data.boundary_targets.foo", "items.0.scope.0.parent_scope_id"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.scope.0.type", "project"),
					resource.TestCheckResourceAttrSet("data.boundary_targets.foo", "items.0.scope_id"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.session_connection_limit", "6"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.session_max_seconds", "6000"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.type", "tcp"),
					resource.TestCheckResourceAttrSet("data.boundary_targets.foo", "items.0.updated_time"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.version", "3"),
					resource.TestCheckResourceAttr("data.boundary_targets.foo", "items.0.worker_filter", "type == \"foo\""),
					resource.TestCheckResourceAttrSet("data.boundary_targets.foo", "scope_id"),
				),
			},
		},
	})
}
