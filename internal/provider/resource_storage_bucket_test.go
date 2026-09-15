// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/storagebuckets"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

const (
	testStorageBucketDescription        = "bar"
	testStorageBucketDescriptionUpdate  = "bar foo"
	testStorageBucketDescriptionUpdate2 = "bar foo foo"
)

var orgStorageBucketBase = `
resource "boundary_storage_bucket" "foo" {
	name        	= "foo"
	scope_id    	= boundary_scope.org1.id
	plugin_name 	= "loopback"
	bucket_name   	= "default"
	worker_filter 	= "\"dev\" in \"/tags/type\""
%s
	depends_on  	= [boundary_role.org1_admin]
}`

var (
	currentStorageBucketSecretsHmacValue string
	currentStorageBucketAttributesValue  string
)

func TestAccStorageBucket(t *testing.T) {
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	url := cfg.BoundaryAddr
	suffix := id.UniqueId()

	resName := "boundary_storage_bucket.foo"
	initialValuesStr := fmt.Sprintf(
		`
	description = "%s"
	attributes_json = jsonencode({
		foo = "bar"
		zip = "zap"
	})
	secrets_json = jsonencode({
		hush = "puppies"
	})
	`,
		testStorageBucketDescription,
	)

	// Changed description and secrets
	valuesStrUpdate1 := fmt.Sprintf(
		`
	description = "%s"
	attributes_json = jsonencode({
		foo = "bar"
		zip = "zoop"
	})
	secrets_json = jsonencode({
		flush = "fluppies"
	})
	`,
		testStorageBucketDescriptionUpdate,
	)

	// Changed description, no secrets update
	valuesStrUpdate2 := fmt.Sprintf(
		`
	description = "%s"
	attributes_json = jsonencode({
		foo = "bar"
		zip = "zoop"
	})
	secrets_json = jsonencode({
		flush = "fluppies"
	})
	`,
		testStorageBucketDescriptionUpdate2,
	)

	// Null only attributes_json; keep secrets (loopback plugin cannot clear secrets)
	valuesStrUpdate3 := fmt.Sprintf(
		`
		description = "%s"
		attributes_json = "null"
		secrets_json = jsonencode({
			flush = "fluppies"
		})
		`,
		testStorageBucketDescriptionUpdate2,
	)

	// Re-add attributes_json and change secrets so HMAC changes
	valuesStrUpdate4 := fmt.Sprintf(
		`
		description = "%s"
		attributes_json = jsonencode({
			foo = "bar"
			zip = "zoop"
		})
		secrets_json = jsonencode({
			flush = "fluppies2"
		})
		`,
		testStorageBucketDescriptionUpdate2,
	)

	initialHcl := fmt.Sprintf(orgStorageBucketBase, initialValuesStr)
	update1Hcl := fmt.Sprintf(orgStorageBucketBase, valuesStrUpdate1)
	update2Hcl := fmt.Sprintf(orgStorageBucketBase, valuesStrUpdate2)
	update3Hcl := fmt.Sprintf(orgStorageBucketBase, valuesStrUpdate3)
	update4Hcl := fmt.Sprintf(orgStorageBucketBase, valuesStrUpdate4)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckStorageBucketResourceDestroy(t, provider, "boundary_storage_bucket"),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg(suffix), initialHcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists(provider, "boundary_scope.org1"),
					testAccCheckStorageBucketResourceExists(provider, resName, expectedAttributesStatePreviouslyEmptyNowSet),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testStorageBucketDescription),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg(suffix), update1Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketResourceExists(provider, resName, expectedAttributesStatePreviouslySetButChanged),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testStorageBucketDescriptionUpdate),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update
				Config: testConfig(url, fooOrg(suffix), update2Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketResourceExists(provider, resName, expectedAttributesStatePreviouslySetNoChange),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testStorageBucketDescriptionUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// this runs the same HCL; mostly used in some manual checking
				// to ensure update is still called even when nothing has
				// changed (as we need for secrets)
				Config: testConfig(url, fooOrg(suffix), update2Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketResourceExists(provider, resName, expectedAttributesStatePreviouslySetNoChange),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testStorageBucketDescriptionUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update: null attributes_json only — secrets unchanged (loopback cannot clear secrets)
				Config: testConfig(url, fooOrg(suffix), update3Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketResourceExists(provider, resName, expectedAttributesStateAttrsNowEmpty),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testStorageBucketDescriptionUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
			{
				// test update: re-add attributes_json with changed secrets
				Config: testConfig(url, fooOrg(suffix), update4Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStorageBucketResourceExists(provider, resName, expectedAttributesStateAttrsRestoredSecretsChanged),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testStorageBucketDescriptionUpdate2),
				),
				ExpectNonEmptyPlan: true,
			},
			importStep(resName, SecretsJsonKey, internalHmacUsedForSecretsConfigHmacKey, internalForceUpdateKey, internalSecretsConfigHmacKey),
		},
	})
}

func testAccCheckStorageBucketResourceExists(testProvider *schema.Provider, name string, expAttrState expectedAttributesState) resource.TestCheckFunc {
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
		sbClient := storagebuckets.NewClient(md.client)

		val, err := sbClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading storage bucket %q: %v", id, err)
		}
		if val == nil {
			return errors.New("empty val returned")
		}
		secretsHmac := val.GetResponse().Map[SecretsHmacKey]
		attrs := val.GetResponse().Map["attributes"]
		switch expAttrState {
		case expectedAttributesStatePreviouslyEmptyNowSet:
			if currentStorageBucketSecretsHmacValue != "" {
				return errors.New("expected no previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val == "" {
				return errors.New("expected non-empty new secrets hmac value")
			}
			currentStorageBucketSecretsHmacValue = val
			if currentStorageBucketAttributesValue != "" {
				return fmt.Errorf("expected no previous attributes value, got %s", currentStorageBucketAttributesValue)
			}
			if attrs == nil {
				return errors.New("expected non-empty new attributes value")
			}
			attrsVal, err := json.Marshal(attrs)
			if err != nil {
				return fmt.Errorf("error marshaling attrs: %w", err)
			}
			currentStorageBucketAttributesValue = string(attrsVal)

		case expectedAttributesStatePreviouslySetButChanged:
			if currentStorageBucketSecretsHmacValue == "" {
				return errors.New("expected previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val == "" {
				return errors.New("expected non-empty new secrets hmac value")
			}
			if val == currentStorageBucketSecretsHmacValue {
				return errors.New("expected changed secrets hmac value")
			}
			currentStorageBucketSecretsHmacValue = val
			if currentStorageBucketAttributesValue == "" {
				return errors.New("expected previous attributes value")
			}
			if attrs == nil {
				return errors.New("expected non-empty new attributes value")
			}
			attrsVal, err := json.Marshal(attrs)
			if err != nil {
				return fmt.Errorf("error marshaling attrs: %w", err)
			}
			if string(attrsVal) == currentStorageBucketAttributesValue {
				return errors.New("expected changed attrs value")
			}
			currentStorageBucketAttributesValue = string(attrsVal)

		case expectedAttributesStatePreviouslySetNoChange:
			if currentStorageBucketSecretsHmacValue == "" {
				return errors.New("expected previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val == "" {
				return errors.New("expected non-empty new secrets hmac value")
			}
			if val != currentStorageBucketSecretsHmacValue {
				return errors.New("expected same secrets hmac value")
			}
			if currentStorageBucketAttributesValue == "" {
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
			if string(attrsVal) != currentStorageBucketAttributesValue {
				return errors.New("expected same attributes value")
			}

		case expectedAttributesStatePreviouslySetNowEmpty:
			if currentStorageBucketSecretsHmacValue == "" {
				return errors.New("expected previous secrets hmac value")
			}
			if secretsHmac != nil {
				return fmt.Errorf("expected empty new secrets hmac value, got %s", secretsHmac.(string))
			}
			currentStorageBucketSecretsHmacValue = ""
			if currentStorageBucketAttributesValue == "" {
				return errors.New("expected previous attributes value")
			}
			if attrs != nil {
				return fmt.Errorf("expected empty new attributes value, got %s", attrs)
			}
			currentStorageBucketAttributesValue = ""

		case expectedAttributesStateAttrsRestoredSecretsChanged:
			// attrs go from empty back to set; secrets change
			if currentStorageBucketSecretsHmacValue == "" {
				return errors.New("expected previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val == "" {
				return errors.New("expected non-empty new secrets hmac value")
			}
			if val == currentStorageBucketSecretsHmacValue {
				return errors.New("expected changed secrets hmac value")
			}
			currentStorageBucketSecretsHmacValue = val
			if currentStorageBucketAttributesValue != "" {
				return fmt.Errorf("expected no previous attributes value, got %s", currentStorageBucketAttributesValue)
			}
			if attrs == nil {
				return errors.New("expected non-empty new attributes value")
			}
			attrsVal, err := json.Marshal(attrs)
			if err != nil {
				return fmt.Errorf("error marshaling attrs: %w", err)
			}
			currentStorageBucketAttributesValue = string(attrsVal)

		case expectedAttributesStateAttrsNowEmpty:
			// secrets unchanged (loopback plugin cannot clear secrets); only attrs cleared
			if currentStorageBucketSecretsHmacValue == "" {
				return errors.New("expected previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val == "" {
				return errors.New("expected non-empty secrets hmac value (secrets cannot be cleared)")
			}
			if val != currentStorageBucketSecretsHmacValue {
				return errors.New("expected unchanged secrets hmac value")
			}
			if currentStorageBucketAttributesValue == "" {
				return errors.New("expected previous attributes value")
			}
			if attrs != nil {
				return fmt.Errorf("expected empty new attributes value, got %s", attrs)
			}
			currentStorageBucketAttributesValue = ""
		}
		return nil
	}
}

func testAccCheckStorageBucketResourceDestroy(t *testing.T, testProvider *schema.Provider, typ string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case string(typ):

				id := rs.Primary.ID
				sbClient := storagebuckets.NewClient(md.client)

				_, err := sbClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed storage bucket %q: %v", id, err)
				}

			default:
				continue
			}
		}
		return nil
	}
}
