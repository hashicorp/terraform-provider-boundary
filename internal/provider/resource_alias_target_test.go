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
					resource.TestCheckResourceAttr(targetAliasResc, ValueKey, targetAliasValue),

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
					resource.TestCheckResourceAttr(targetAliasResc, ValueKey, targetAliasValueUpdated),

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
