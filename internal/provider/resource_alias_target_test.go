// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/aliases"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

type aliasDestroyType string

const (
	targetAliasType aliasDestroyType = "boundary_alias_target"

	targetAliasResc         = "boundary_alias_target.example"
	targetAliasName         = "foo"
	targetAliasDesc         = "the foo"
	targetAliasValue        = "value.example"
	targetAliasValueUpdated = "value.example.updated"
	targetAliasNamespace    = "target"
	targetAliasUpdate       = "_random"
)

var fooBarTarget = `
resource "boundary_target" "foo" {
	type         = "tcp"
	name         = "test"
	description  = "test target"
	default_port = 22
	address      = "127.0.0.1"
	scope_id     = boundary_scope.proj1.id
	depends_on   = [boundary_role.proj1_admin]
}`

var aliasId string

func targetAliasResource(name string, description string, value string) string {
	return fmt.Sprintf(`
resource "boundary_alias_target" "example" {
	name  = "%s"
	description = "%s"
	value = "%s"
	scope_id = "global"
	destination_id = boundary_target.foo.id
	authorize_session_host_id = "hst_1234567890"
	depends_on = [boundary_target.foo]
}`, name, description, value)
}

func TestAccAliasTarget(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	res := targetAliasResource(targetAliasName, targetAliasDesc, targetAliasValue)

	resUpdate := targetAliasResource(targetAliasName+targetAliasUpdate,
		targetAliasDesc+targetAliasUpdate, targetAliasValueUpdated)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAliasResourceDestroy(t, provider, targetAliasType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, fooBarTarget, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(targetAliasResc, NameKey, targetAliasName),
					resource.TestCheckResourceAttr(targetAliasResc, DescriptionKey, targetAliasDesc),
					resource.TestCheckResourceAttr(targetAliasResc, aliasTargetBaseValueKey, targetAliasValue),
					resource.TestCheckResourceAttrSet(targetAliasResc, ValueKey),

					testAccCheckAliasResourceExists(provider, targetAliasResc),
				),
			},
			importStep(targetAliasResc),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, fooBarTarget, resUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(targetAliasResc, NameKey, targetAliasName+targetAliasUpdate),
					resource.TestCheckResourceAttr(targetAliasResc, DescriptionKey, targetAliasDesc+targetAliasUpdate),
					resource.TestCheckResourceAttr(targetAliasResc, aliasTargetBaseValueKey, targetAliasValueUpdated),
					resource.TestCheckResourceAttrSet(targetAliasResc, ValueKey),

					testAccCheckAliasResourceExists(provider, targetAliasResc),
				),
			},
			importStep(targetAliasResc),
			{
				// Run a plan only update and verify no changes
				PlanOnly: true,
				Config:   testConfig(url, fooOrg, firstProjectFoo, fooBarTarget, resUpdate),
			},
			importStep(targetAliasResc),
			{
				// update again but apply a preConfig to externally update resource
				// TODO: Boundary currently causes an error on moving back to a previously
				// used token, for now verify that a plan only step had changes
				PreConfig:          func() { targetAliasExternalUpdate(t, provider) },
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Config:             testConfig(url, fooOrg, firstProjectFoo, fooBarTarget, resUpdate),
			},
			importStep(targetAliasResc),
		},
	})
}

func targetAliasExternalUpdate(t *testing.T, testProvider *schema.Provider) {
	if aliasId == "" {
		t.Fatal("aliasId must be set before testing an external update")
	}

	md := testProvider.Meta().(*metaData)
	ac := aliases.NewClient(md.client)
	ar, err := ac.Read(context.Background(), aliasId)
	if err != nil {
		t.Fatal(fmt.Errorf("got an error reading %q: %w", storeId, err))
	}

	// update alias options
	var opts []aliases.Option
	opts = append(opts, aliases.WithDescription("this is an updated description, my guy"))

	_, err = ac.Update(context.Background(), ar.Item.Id, ar.Item.Version, opts...)
	if err != nil {
		t.Fatal(fmt.Errorf("got an error updating %q: %w", ar.Item.Id, err))
	}
}

func testAccCheckAliasResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("no ID is set")
		}
		aliasId = id

		md := testProvider.Meta().(*metaData)
		c := aliases.NewClient(md.client)
		if _, err := c.Read(context.Background(), id); err != nil {
			return fmt.Errorf("got an error reading %q: %w", id, err)
		}

		return nil
	}
}

func testAccCheckAliasResourceDestroy(t *testing.T, testProvider *schema.Provider, typ aliasDestroyType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case string(typ):
				id := rs.Primary.ID

				c := aliases.NewClient(md.client)
				_, err := c.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed target alias %q: %v", id, err)
				}
			default:
				continue
			}
		}
		return nil
	}
}

// buildAliasRaw returns a minimal raw API response map suitable for use in
// setFromTargetAliasResponseMap tests.
func buildAliasRaw(id, scopeId, value string) map[string]interface{} {
	return map[string]interface{}{
		"id":       id,
		"scope_id": scopeId,
		"value":    value,
		"type":     "target",
	}
}

// TestSetFromTargetAliasResponseMap_Create verifies that on an initial create
// the supplied configuredBaseValue is stored in state as base_value.
func TestSetFromTargetAliasResponseMap_Create(t *testing.T) {
	r := resourceAliasTarget()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		ValueKey:   "foo.example",
		ScopeIdKey: "global",
		TypeKey:    "target",
	})

	raw := buildAliasRaw("alt_test", "global", "foo.example")
	if err := setFromTargetAliasResponseMap(d, raw, "foo.example"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := d.Get(aliasTargetBaseValueKey).(string); got != "foo.example" {
		t.Errorf("base_value = %q, want %q", got, "foo.example")
	}
	if got := d.Get(ValueKey).(string); got != "foo.example" {
		t.Errorf("value = %q, want %q", got, "foo.example")
	}
}

// TestSetFromTargetAliasResponseMap_ReadPreservesBaseValue verifies that on a
// Read (configuredBaseValue == "") the existing base_value is not overwritten
// by the server-appended suffix stored in the value field.
func TestSetFromTargetAliasResponseMap_ReadPreservesBaseValue(t *testing.T) {
	r := resourceAliasTarget()
	// Simulate post-create state: base_value = "foo.example",
	// value = "foo.example/p_proj123" (server appended suffix).
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		ValueKey:                "foo.example/p_proj123",
		aliasTargetBaseValueKey: "foo.example",
		ScopeIdKey:              "p_proj123",
		TypeKey:                 "target",
	})

	// API Read returns the suffixed value; no base_value in the response.
	raw := buildAliasRaw("alt_test", "p_proj123", "foo.example/p_proj123")

	// configuredBaseValue is "" because Read does not know the original config value.
	if err := setFromTargetAliasResponseMap(d, raw, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// base_value must still reflect the un-suffixed original, not the API value.
	if got := d.Get(aliasTargetBaseValueKey).(string); got != "foo.example" {
		t.Errorf("base_value = %q, want %q (existing base_value must be preserved on Read)", got, "foo.example")
	}
	if got := d.Get(ValueKey).(string); got != "foo.example/p_proj123" {
		t.Errorf("value = %q, want %q", got, "foo.example/p_proj123")
	}
}

// TestSetFromTargetAliasResponseMap_ImportFallsBackToRawValue verifies that on
// an import (no existing state, configuredBaseValue == "") base_value is
// initialised from the raw API value.
func TestSetFromTargetAliasResponseMap_ImportFallsBackToRawValue(t *testing.T) {
	r := resourceAliasTarget()
	// Empty state – simulates a freshly imported resource.
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{})

	raw := buildAliasRaw("alt_test", "p_proj123", "foo.example/p_proj123")
	if err := setFromTargetAliasResponseMap(d, raw, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With no existing state the function must fall back to the raw API value.
	if got := d.Get(aliasTargetBaseValueKey).(string); got != "foo.example/p_proj123" {
		t.Errorf("base_value = %q, want %q (should fall back to raw value on import)", got, "foo.example/p_proj123")
	}
}

// TestSetFromTargetAliasResponseMap_UpdateChangedValue verifies that when the
// user deliberately changes value, base_value is updated to the new configured
// value (not the server-returned suffixed form).
func TestSetFromTargetAliasResponseMap_UpdateChangedValue(t *testing.T) {
	r := resourceAliasTarget()
	// Pre-update state: base_value="foo.example", value="foo.example/p_proj123".
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		ValueKey:                "foo.example/p_proj123",
		aliasTargetBaseValueKey: "foo.example",
		ScopeIdKey:              "p_proj123",
		TypeKey:                 "target",
	})

	// After the update the server returns a new suffixed value.
	raw := buildAliasRaw("alt_test", "p_proj123", "bar.example/p_proj123")

	// User changed value to "bar.example".
	if err := setFromTargetAliasResponseMap(d, raw, "bar.example"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := d.Get(aliasTargetBaseValueKey).(string); got != "bar.example" {
		t.Errorf("base_value = %q, want %q (base_value must update when value changes)", got, "bar.example")
	}
	if got := d.Get(ValueKey).(string); got != "bar.example/p_proj123" {
		t.Errorf("value = %q, want %q", got, "bar.example/p_proj123")
	}
}

// ---------------------------------------------------------------------------
// Acceptance tests
// ---------------------------------------------------------------------------

// targetAliasProjectResource builds HCL for a boundary_alias_target that
// targets the project-scoped test target and uses the project scope as its
// scope_id.  This exercises the server-suffix code path.
func targetAliasProjectResource(name, description, value string) string {
	return fmt.Sprintf(`
resource "boundary_alias_target" "example" {
	name        = "%s"
	description = "%s"
	value       = "%s"
	scope_id    = boundary_scope.proj1.id
	destination_id = boundary_target.foo.id
	depends_on  = [boundary_target.foo]
}`, name, description, value)
}

// TestAccAliasTargetProjectScoped tests create / read / update / import for a
// project-scoped alias.  It also verifies that a plan-only step after apply
// produces no diff, confirming that the diff-suppression mechanism keeps
// base_value and value in sync across refreshes.
func TestAccAliasTargetProjectScoped(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	res := targetAliasProjectResource(targetAliasName, targetAliasDesc, targetAliasValue)
	resUpdate := targetAliasProjectResource(
		targetAliasName+targetAliasUpdate,
		targetAliasDesc+targetAliasUpdate,
		targetAliasValueUpdated,
	)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAliasResourceDestroy(t, provider, targetAliasType),
		Steps: []resource.TestStep{
			{
				// create project-scoped alias
				Config: testConfig(url, fooOrg, firstProjectFoo, fooBarTarget, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(targetAliasResc, NameKey, targetAliasName),
					resource.TestCheckResourceAttr(targetAliasResc, DescriptionKey, targetAliasDesc),
					resource.TestCheckResourceAttr(targetAliasResc, aliasTargetBaseValueKey, targetAliasValue),
					resource.TestCheckResourceAttrSet(targetAliasResc, ValueKey),
					testAccCheckAliasResourceExists(provider, targetAliasResc),
				),
			},
			importStep(targetAliasResc),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, fooBarTarget, resUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(targetAliasResc, NameKey, targetAliasName+targetAliasUpdate),
					resource.TestCheckResourceAttr(targetAliasResc, DescriptionKey, targetAliasDesc+targetAliasUpdate),
					resource.TestCheckResourceAttr(targetAliasResc, aliasTargetBaseValueKey, targetAliasValueUpdated),
					resource.TestCheckResourceAttrSet(targetAliasResc, ValueKey),
					testAccCheckAliasResourceExists(provider, targetAliasResc),
				),
			},
			importStep(targetAliasResc),
			{
				// plan-only: must produce no diff (diff suppression is working)
				PlanOnly: true,
				Config:   testConfig(url, fooOrg, firstProjectFoo, fooBarTarget, resUpdate),
			},
		},
	})
}

// TestAccAliasTargetGlobalRegression is a regression guard for global-scoped
// aliases.  Global aliases do not receive a server-appended suffix, so value
// and base_value must be identical after create and must remain stable across
// refresh and import.
func TestAccAliasTargetGlobalRegression(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	res := targetAliasResource(targetAliasName, targetAliasDesc, targetAliasValue)
	resUpdate := targetAliasResource(
		targetAliasName+targetAliasUpdate,
		targetAliasDesc+targetAliasUpdate,
		targetAliasValueUpdated,
	)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAliasResourceDestroy(t, provider, targetAliasType),
		Steps: []resource.TestStep{
			{
				// create global alias
				Config: testConfig(url, fooOrg, firstProjectFoo, fooBarTarget, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(targetAliasResc, NameKey, targetAliasName),
					resource.TestCheckResourceAttr(targetAliasResc, DescriptionKey, targetAliasDesc),
					resource.TestCheckResourceAttr(targetAliasResc, aliasTargetBaseValueKey, targetAliasValue),
					resource.TestCheckResourceAttrSet(targetAliasResc, ValueKey),
					testAccCheckAliasResourceExists(provider, targetAliasResc),
				),
			},
			importStep(targetAliasResc),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, fooBarTarget, resUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(targetAliasResc, NameKey, targetAliasName+targetAliasUpdate),
					resource.TestCheckResourceAttr(targetAliasResc, DescriptionKey, targetAliasDesc+targetAliasUpdate),
					resource.TestCheckResourceAttr(targetAliasResc, aliasTargetBaseValueKey, targetAliasValueUpdated),
					resource.TestCheckResourceAttrSet(targetAliasResc, ValueKey),
					testAccCheckAliasResourceExists(provider, targetAliasResc),
				),
			},
			importStep(targetAliasResc),
			{
				// plan-only: no diff after refresh
				PlanOnly: true,
				Config:   testConfig(url, fooOrg, firstProjectFoo, fooBarTarget, resUpdate),
			},
		},
	})
}
