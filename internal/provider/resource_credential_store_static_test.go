// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/boundary/api/credentialstores"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	staticCredStoreResc      = "boundary_credential_store_static.example"
	staticCredStoreName      = "foo"
	staticCredStoreDesc      = "the foo"
	staticCredStoreNamespace = "static"
	staticCredStoreUpdate    = "_random"
)

func staticCredStoreResource(name string, description string) string {
	return fmt.Sprintf(`
resource "boundary_credential_store_static" "example" {
	name  = "%s"
	description = "%s"
	scope_id = boundary_scope.proj1.id
	depends_on = [boundary_role.proj1_admin]
}`, name,
		description)
}

func TestAccCredentialStoreStatic(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	res := staticCredStoreResource(staticCredStoreName,
		staticCredStoreDesc)

	resUpdate := staticCredStoreResource(staticCredStoreName+staticCredStoreUpdate,
		staticCredStoreDesc+staticCredStoreUpdate)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialStoreResourceDestroy(t, provider, staticStoreCredentialStoreType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(staticCredStoreResc, NameKey, staticCredStoreName),
					resource.TestCheckResourceAttr(staticCredStoreResc, DescriptionKey, staticCredStoreDesc),

					testAccCheckCredentialStoreResourceExists(provider, staticCredStoreResc),
				),
			},
			importStep(staticCredStoreResc),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, resUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(staticCredStoreResc, NameKey, staticCredStoreName+staticCredStoreUpdate),
					resource.TestCheckResourceAttr(staticCredStoreResc, DescriptionKey, staticCredStoreDesc+staticCredStoreUpdate),

					testAccCheckCredentialStoreResourceExists(provider, staticCredStoreResc),
				),
			},
			importStep(staticCredStoreResc),
			{
				// Run a plan only update and verify no changes
				PlanOnly: true,
				Config:   testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(staticCredStoreResc),
			{
				// update again but apply a preConfig to externally update resource
				// TODO: Boundary currently causes an error on moving back to a previously
				// used token, for now verify that a plan only step had changes
				PreConfig:          func() { staticCredentialStoreExternalUpdate(t, provider) },
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Config:             testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(staticCredStoreResc),
		},
	})
}

func staticCredentialStoreExternalUpdate(t *testing.T, testProvider *schema.Provider) {
	if storeId == "" {
		t.Fatal("storeId must be set before testing an external update")
	}

	md := testProvider.Meta().(*metaData)
	c := credentialstores.NewClient(md.client)
	cr, err := c.Read(context.Background(), storeId)
	if err != nil {
		t.Fatal(fmt.Errorf("got an error reading %q: %w", storeId, err))
	}

	// update credential store options
	var opts []credentialstores.Option
	opts = append(opts, credentialstores.WithDescription("this is an updated description, my guy"))

	_, err = c.Update(context.Background(), cr.Item.Id, cr.Item.Version, opts...)
	if err != nil {
		t.Fatal(fmt.Errorf("got an error updating %q: %w", cr.Item.Id, err))
	}
}
