// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/boundary/api/hostcatalogs"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	_ "github.com/kr/pretty" // So I don't have to keep adding to/removing from go.mod :-)
)

const (
	testPluginHostCatalogDescription        = "bar"
	testPluginHostCatalogDescriptionUpdate  = "bar foo"
	testPluginHostCatalogDescriptionUpdate2 = "bar foo foo"

	testPluginHostCatalogWorkerFilterUpdate  = "\"dev\" in \"/tags/type\""
	testPluginHostCatalogWorkerFilterUpdate2 = "\"pki\" in \"/tags/type\""
)

// expectedAttributesState is used here and in plugin host sets to control how
// we expect attributes to behave compared to the previous step.
type expectedAttributesState uint

const (
	expectedAttributesStatePreviouslyEmptyNowSet expectedAttributesState = iota
	expectedAttributesStatePreviouslySetNoChange
	expectedAttributesStatePreviouslySetButChanged
	expectedAttributesStatePreviouslySetNowEmpty
)

var projPluginHostCatalogBase = `
resource "boundary_host_catalog_plugin" "foo" {
	name        = "foo"
	scope_id    = boundary_scope.proj1.id
	plugin_name = "loopback"
%s
	depends_on  = [boundary_role.proj1_admin]
}`

var (
	currentPluginHostCatalogSecretsHmacValue string
	currentPluginHostCatalogAttributesValue  string
	testStep                                 = 1
)

// NOTE: In the test below, secrets and attributes change in the same manner at
// the same time; the eventual result is the same even if the JSON looks
// different. Thus expectedAttributesState also controls expectations for
// secrets.
func TestAccPluginHostCatalog(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resName := "boundary_host_catalog_plugin.foo"
	initialValuesStr := fmt.Sprintf(`
	description = "%s"
	attributes_json = jsonencode({
		foo = "bar"
		zip = "zap"
	})
	secrets_json = jsonencode({
		hush = "puppies"
	})
	`,
		testPluginHostCatalogDescription,
	)

	// Changed description and secrets
	valuesStrUpdate1 := fmt.Sprintf(`
	description = "%s"
	attributes_json = jsonencode({
		foo = "bar"
		zip = "zoop"
	})
	secrets_json = jsonencode({
		flush = "fluppies"
	})
	`,
		testPluginHostCatalogDescriptionUpdate,
	)

	// Changed description, no secrets update
	valuesStrUpdate2 := fmt.Sprintf(`
	description = "%s"
	attributes_json = jsonencode({
		foo = "bar"
		zip = "zoop"
	})
	secrets_json = jsonencode({
		flush = "fluppies"
	})
	`,
		testPluginHostCatalogDescriptionUpdate2,
	)

	// Same description, now empty secrets
	valuesStrUpdate3 := fmt.Sprintf(`
		description = "%s"
		attributes_json = jsonencode({
			foo = "bar"
			zip = "zoop"
		})
		`,
		testPluginHostCatalogDescriptionUpdate2,
	)

	// Same description, now explicitly unset secrets and blankify attrs
	valuesStrUpdate4 := fmt.Sprintf(`
		description = "%s"
		secrets_json = "null"
		`,
		testPluginHostCatalogDescriptionUpdate2,
	)

	// Set values again
	valuesStrUpdate5 := fmt.Sprintf(`
		description = "%s"
		attributes_json = jsonencode({
			foo = "bar"
			zip = "zoop"
		})
		secrets_json = jsonencode({
			flush = "fluppies"
		})
		`,
		testPluginHostCatalogDescriptionUpdate2,
	)

	// Explicitly set both secrets and attributes to null
	valuesStrUpdate6 := fmt.Sprintf(`
		description = "%s"
		attributes_json = "null"
		secrets_json = "null"
		`,
		testPluginHostCatalogDescriptionUpdate2,
	)

	initialHcl := fmt.Sprintf(projPluginHostCatalogBase, initialValuesStr)
	update1Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate1)
	update2Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate2)
	update3Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate3)
	update4Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate4)
	update5Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate5)
	update6Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate6)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckHostCatalogResourceDestroy(t, provider, baseHostCatalogType),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, firstProjectFoo, initialHcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists(provider, "boundary_scope.org1"),
					testAccCheckScopeResourceExists(provider, "boundary_scope.proj1"),
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslyEmptyNowSet),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescription),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update1Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslySetButChanged),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update2Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslySetNoChange),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// this runs the same HCL; mostly used in some manual checking
				// to ensure update is still called even when nothing has
				// changed (as we need for secrets)
				Config: testConfig(url, fooOrg, firstProjectFoo, update2Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslySetNoChange),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update3Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslySetNoChange),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update4Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslySetNowEmpty),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update5Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslyEmptyNowSet),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update6Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslySetNowEmpty),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
		},
	})
}

func TestAccPluginHostCatalogWithWorkerFilter(t *testing.T) {
	t.Skip("Skipping test until Boundary Terraform Provider can unit tests for Boundary Enterprise only features")

	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resName := "boundary_host_catalog_plugin.foo"
	initialValuesStr := fmt.Sprintf(`
	description = "%s"
	attributes_json = jsonencode({
		foo = "bar"
		zip = "zap"
	})
	secrets_json = jsonencode({
		hush = "puppies"
	})
	`,
		testPluginHostCatalogDescription,
	)

	// Changed description, secrets and added worker filter.
	valuesStrUpdate1 := fmt.Sprintf(`
	description = "%s"
	worker_filter = %q
	attributes_json = jsonencode({
		foo = "bar"
		zip = "zoop"
	})
	secrets_json = jsonencode({
		flush = "fluppies"
	})
	`,
		testPluginHostCatalogDescriptionUpdate,
		testPluginHostCatalogWorkerFilterUpdate,
	)

	// Changed description, no secrets update, nullify worker filter.
	valuesStrUpdate2 := fmt.Sprintf(`
	description = "%s"
	worker_filter = null
	attributes_json = jsonencode({
		foo = "bar"
		zip = "zoop"
	})
	secrets_json = jsonencode({
		flush = "fluppies"
	})
	`,
		testPluginHostCatalogDescriptionUpdate2,
	)

	// Same description and set a worker filter again, now empty secrets.
	valuesStrUpdate3 := fmt.Sprintf(`
		description = "%s"
		worker_filter = %q
		attributes_json = jsonencode({
			foo = "bar"
			zip = "zoop"
		})
		`,
		testPluginHostCatalogDescriptionUpdate2,
		testPluginHostCatalogWorkerFilterUpdate2,
	)

	// Same description, now explicitly unset secrets and blankify attrs and
	// worker filter.
	valuesStrUpdate4 := fmt.Sprintf(`
		description = "%s"
		secrets_json = "null"
		`,
		testPluginHostCatalogDescriptionUpdate2,
	)

	// Set values again
	valuesStrUpdate5 := fmt.Sprintf(`
		description = "%s"
		worker_filter = %q
		attributes_json = jsonencode({
			foo = "bar"
			zip = "zoop"
		})
		secrets_json = jsonencode({
			flush = "fluppies"
		})
		`,
		testPluginHostCatalogDescriptionUpdate2,
		testPluginHostCatalogWorkerFilterUpdate2,
	)

	// Explicitly set both secrets and attributes to null
	valuesStrUpdate6 := fmt.Sprintf(`
		description = "%s"
		worker_filter = %q
		attributes_json = "null"
		secrets_json = "null"
		`,
		testPluginHostCatalogDescriptionUpdate2,
		testPluginHostCatalogWorkerFilterUpdate2,
	)

	initialHcl := fmt.Sprintf(projPluginHostCatalogBase, initialValuesStr)
	update1Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate1)
	update2Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate2)
	update3Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate3)
	update4Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate4)
	update5Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate5)
	update6Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate6)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckHostCatalogResourceDestroy(t, provider, baseHostCatalogType),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, firstProjectFoo, initialHcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists(provider, "boundary_scope.org1"),
					testAccCheckScopeResourceExists(provider, "boundary_scope.proj1"),
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslyEmptyNowSet),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescription),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update1Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslySetButChanged),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate),
					resource.TestCheckResourceAttr(resName, WorkerFilterKey, testPluginHostCatalogWorkerFilterUpdate),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update2Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslySetNoChange),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// this runs the same HCL; mostly used in some manual checking
				// to ensure update is still called even when nothing has
				// changed (as we need for secrets)
				Config: testConfig(url, fooOrg, firstProjectFoo, update2Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslySetNoChange),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update3Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslySetNoChange),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
					resource.TestCheckResourceAttr(resName, WorkerFilterKey, testPluginHostCatalogWorkerFilterUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update4Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslySetNowEmpty),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update5Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslyEmptyNowSet),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
					resource.TestCheckResourceAttr(resName, WorkerFilterKey, testPluginHostCatalogWorkerFilterUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update6Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttributesStatePreviouslySetNowEmpty),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
					resource.TestCheckResourceAttr(resName, WorkerFilterKey, testPluginHostCatalogWorkerFilterUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
		},
	})
}

func testAccCheckPluginHostCatalogResourceExists(testProvider *schema.Provider, name string, expAttrState expectedAttributesState) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("No ID is set")
		}

		md := testProvider.Meta().(*metaData)
		hcClient := hostcatalogs.NewClient(md.client)

		val, err := hcClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading host catalog %q: %v", id, err)
		}
		if val == nil {
			return errors.New("empty val returned")
		}
		secretsHmac := val.GetResponse().Map[SecretsHmacKey]
		attrs := val.GetResponse().Map["attributes"]
		switch expAttrState {
		case expectedAttributesStatePreviouslyEmptyNowSet:
			if currentPluginHostCatalogSecretsHmacValue != "" {
				return errors.New("expected no previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val == "" {
				return errors.New("expected non-empty new secrets hmac value")
			}
			currentPluginHostCatalogSecretsHmacValue = val
			if currentPluginHostCatalogAttributesValue != "" {
				return fmt.Errorf("expected no previous attributes value, got %s", currentPluginHostCatalogAttributesValue)
			}
			if attrs == nil {
				return errors.New("expected non-empty new attributes value")
			}
			attrsVal, err := json.Marshal(attrs)
			if err != nil {
				return fmt.Errorf("error marshaling attrs: %w", err)
			}
			currentPluginHostCatalogAttributesValue = string(attrsVal)

		case expectedAttributesStatePreviouslySetButChanged:
			if currentPluginHostCatalogSecretsHmacValue == "" {
				return errors.New("expected previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val == "" {
				return errors.New("expected non-empty new secrets hmac value")
			}
			if val == currentPluginHostCatalogSecretsHmacValue {
				return errors.New("expected changed secrets hmac value")
			}
			currentPluginHostCatalogSecretsHmacValue = val
			if currentPluginHostCatalogAttributesValue == "" {
				return errors.New("expected previous attributes value")
			}
			if attrs == nil {
				return errors.New("expected non-empty new attributes value")
			}
			attrsVal, err := json.Marshal(attrs)
			if err != nil {
				return fmt.Errorf("error marshaling attrs: %w", err)
			}
			if string(attrsVal) == currentPluginHostCatalogAttributesValue {
				return errors.New("expected changed attrs value")
			}
			currentPluginHostCatalogAttributesValue = string(attrsVal)

		case expectedAttributesStatePreviouslySetNoChange:
			if currentPluginHostCatalogSecretsHmacValue == "" {
				return errors.New("expected previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val == "" {
				return errors.New("expected non-empty new secrets hmac value")
			}
			if val != currentPluginHostCatalogSecretsHmacValue {
				return errors.New("expected same secrets hmac value")
			}
			if currentPluginHostCatalogAttributesValue == "" {
				return errors.New("expected previous attributes value")
			}
			if attrs == nil {
				return errors.New("expected non-empty new attributes value")
			}
			attrsVal, err := json.Marshal(attrs)
			if err != nil {
				return fmt.Errorf("error marshaling attrs: %w", err)
			}
			if string(attrsVal) == "" {
				return errors.New("expected non-empty new attributes value")
			}
			if string(attrsVal) != currentPluginHostCatalogAttributesValue {
				return errors.New("expected same attributes value")
			}

		case expectedAttributesStatePreviouslySetNowEmpty:
			if currentPluginHostCatalogSecretsHmacValue == "" {
				return errors.New("expected previous secrets hmac value")
			}
			if secretsHmac != nil {
				return fmt.Errorf("expected empty new secrets hmac value, got %s", secretsHmac.(string))
			}
			currentPluginHostCatalogSecretsHmacValue = ""
			if currentPluginHostCatalogAttributesValue == "" {
				return errors.New("expected previous attributes value")
			}
			if attrs != nil {
				return fmt.Errorf("expected empty new attributes value, got %s", attrs)
			}
			currentPluginHostCatalogAttributesValue = ""
		}

		testStep++
		return nil
	}
}
