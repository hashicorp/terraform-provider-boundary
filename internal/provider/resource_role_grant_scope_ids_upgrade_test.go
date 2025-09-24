// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/boundary/api/roles"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	orgRoleWithGrantScopes = `
	resource "boundary_role" "with_grant_scopes" {
		name            = "with_grant_scopes"
		description     = "with grant scopes"
		principal_ids   = [boundary_user.foo.id]
		scope_id        = boundary_scope.org1.id
		depends_on      = [boundary_role.org1_admin]
		grant_scope_ids = ["this", boundary_scope.proj1.id]
	}`

	orgRoleWithGrantScopesUpdate = `
	resource "boundary_role" "with_grant_scopes" {
		name            = "with_grant_scopes_update"
		description     = "with grant scopes update"
		principal_ids   = [boundary_user.foo.id]
		scope_id        = boundary_scope.org1.id
		depends_on      = [boundary_role.org1_admin]
		grant_scope_ids = ["this", "children"]
	}`

	orgRoleWithInvalidGrantScopesUpdate = `
	resource "boundary_role" "with_grant_scopes" {
		name            = "with_grant_scopes_update"
		description     = "with grant scopes update"
		principal_ids   = [boundary_user.foo.id]
		scope_id        = boundary_scope.org1.id
		depends_on      = [boundary_role.org1_admin]
		grant_scope_ids = ["this", "children", "p_foobar1234"]
	}`

	conversionOrgRoleWithGrantScopeIdsConversion = `
resource "boundary_role" "with_grant_scope_id" {
	name             = "grant scope id role"
	scope_id         = boundary_scope.org1.id
	depends_on       = [boundary_role.org1_admin]
	grant_scope_ids  = ["this", boundary_scope.proj1.id]
}`

	conversionOrgRoleWithGrantScopeIdsUpdate = `
resource "boundary_role" "with_grant_scope_id" {
	name             = "grant scope id role"
	scope_id         = boundary_scope.org1.id
	depends_on       = [boundary_role.org1_admin]
	grant_scope_ids  = ["this", "children"]
	}`
)

// TestAccRoleWithGrantScopes exercises creation and update with valid and
// invalid grant scopes
func TestAccRoleWithGrantScopes(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	t.Cleanup(tc.Shutdown)
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckRoleResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// Create with valid grant scopes should create role
				Config: testConfig(url, fooOrg, firstProjectFoo, fooUser, orgRoleWithGrantScopes),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists(provider, "boundary_scope.proj1"),
					testAccCheckRoleResourceExists(provider, "boundary_role.with_grant_scopes"),
					testAccCheckUserResourceExists(provider, "boundary_user.foo"),
					testAccCheckRoleResourceGrantScopesSet(provider, "boundary_role.with_grant_scopes", []string{"this", "boundary_scope.proj1"}),
					resource.TestCheckResourceAttr("boundary_role.with_grant_scopes", DescriptionKey, "with grant scopes"),
					resource.TestCheckResourceAttr("boundary_role.with_grant_scopes", NameKey, "with_grant_scopes"),
				),
			},
			importStep("boundary_role.with_grant_scopes"),
			{
				// Update with invalid grant scopes should update role but return empty plan
				// since grant scopes were not set correctly.
				Config: testConfig(url, fooOrg, firstProjectFoo, fooUser, orgRoleWithInvalidGrantScopesUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists(provider, "boundary_role.with_grant_scopes"),
					resource.TestCheckResourceAttr("boundary_role.with_grant_scopes", DescriptionKey, "with grant scopes update"),
					resource.TestCheckResourceAttr("boundary_role.with_grant_scopes", NameKey, "with_grant_scopes_update"),
				),
				ExpectError: regexp.MustCompile(`Unable to set grant scopes on role`),
			},
			{
				// Update again without invalid principal should produce empty plan
				Config: testConfig(url, fooOrg, firstProjectFoo, fooUser, orgRoleWithGrantScopesUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleResourceExists(provider, "boundary_role.with_grant_scopes"),
					testAccCheckRoleResourceGrantScopesSet(provider, "boundary_role.with_grant_scopes", []string{"this", "children"}),
					resource.TestCheckResourceAttr("boundary_role.with_grant_scopes", DescriptionKey, "with grant scopes update"),
					resource.TestCheckResourceAttr("boundary_role.with_grant_scopes", NameKey, "with_grant_scopes_update"),
				),
			},
			importStep("boundary_role.with_grant_scopes"),
		},
	})
}

func testAccCheckRoleResourceGrantScopesSet(testProvider *schema.Provider, name string, grantScopeIds []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("role resource not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("role resource ID is not set")
		}

		scopeIds := []string{}
		for _, grantScope := range grantScopeIds {
			if grantScope == "children" || grantScope == "this" || grantScope == "descendants" {
				scopeIds = append(scopeIds, grantScope)
				continue
			}

			scope, ok := s.RootModule().Resources[grantScope]
			if !ok {
				return fmt.Errorf("scope resource not found: %s", grantScope)
			}

			scopeId := scope.Primary.ID
			if scopeId == "" {
				return fmt.Errorf("scope resource ID not set")
			}

			scopeIds = append(scopeIds, scopeId)
		}

		md := testProvider.Meta().(*metaData)
		rolesClient := roles.NewClient(md.client)

		rr, err := rolesClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading role %q: %v", id, err)
		}

		// for every grant scope set on the role in the state, ensure each role
		// in boundary has the same setings
		if len(rr.Item.GrantScopeIds) == 0 {
			return fmt.Errorf("no grant scope ids found in boundary")
		}

		for _, stateGrantScopeId := range rr.Item.GrantScopeIds {
			ok := false
			for _, gotScopeId := range scopeIds {
				if gotScopeId == stateGrantScopeId {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("grant scope id in state not set in boundary: %s", stateGrantScopeId)
			}
		}

		return nil
	}
}
