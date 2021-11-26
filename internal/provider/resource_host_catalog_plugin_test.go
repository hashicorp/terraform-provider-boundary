package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
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
)

type expectedAttrSecretsHmacState uint

const (
	expectedAttrSecretsHmacStatePreviouslyEmptyNowSet expectedAttrSecretsHmacState = iota
	expectedAttrSecretsHmacStatePreviouslySetNoChange
	expectedAttrSecretsHmacStatePreviouslySetButChanged
	expectedAttrSecretsHmacStatePreviouslySetNowEmpty
)

var projPluginHostCatalogBase = `
resource "boundary_host_catalog_plugin" "foo" {
	name        = "foo"
	scope_id    = boundary_scope.proj1.id
	type        = "plugin"
	plugin_name = "loopback"
%s
	depends_on  = [boundary_role.proj1_admin]
}`

var (
	currentSecretsHmacValue string
	currentAttributesValue  string
)

func TestAccPluginHostCatalogCreateUpdate(t *testing.T) {
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
	_ = update5Hcl
	_ = update6Hcl

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckPluginHostCatalogResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, firstProjectFoo, initialHcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScopeResourceExists(provider, "boundary_scope.org1"),
					testAccCheckScopeResourceExists(provider, "boundary_scope.proj1"),
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttrSecretsHmacStatePreviouslyEmptyNowSet),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescription),
				),
			},
			importStep(resName, "secrets"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update1Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttrSecretsHmacStatePreviouslySetButChanged),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate),
				),
			},
			importStep(resName, "secrets"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update2Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttrSecretsHmacStatePreviouslySetNoChange),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
			},
			importStep(resName, "secrets"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update3Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttrSecretsHmacStatePreviouslySetNoChange),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
			},
			importStep(resName, "secrets"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update4Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttrSecretsHmacStatePreviouslySetNowEmpty),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
			},
			importStep(resName, "secrets"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update5Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttrSecretsHmacStatePreviouslyEmptyNowSet),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
			},
			importStep(resName, "secrets"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update6Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedAttrSecretsHmacStatePreviouslySetNowEmpty),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
			},
			importStep(resName, "secrets"),
		},
	})
}

func testAccCheckPluginHostCatalogResourceExists(testProvider *schema.Provider, name string, expSecrets expectedAttrSecretsHmacState) resource.TestCheckFunc {
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
		switch expSecrets {
		case expectedAttrSecretsHmacStatePreviouslyEmptyNowSet:
			if currentSecretsHmacValue != "" {
				return errors.New("expected no previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val == "" {
				return errors.New("expected non-empty new secrets hmac value")
			}
			currentSecretsHmacValue = val
			if currentAttributesValue != "" {
				return fmt.Errorf("expected no previous attributes value, got %s", currentAttributesValue)
			}
			if attrs == nil {
				return errors.New("expected non-empty new attributes value")
			}
			attrsVal, err := json.Marshal(attrs)
			if err != nil {
				return fmt.Errorf("error marshaling attrs: %w", err)
			}
			currentAttributesValue = string(attrsVal)

		case expectedAttrSecretsHmacStatePreviouslySetButChanged:
			if currentSecretsHmacValue == "" {
				return errors.New("expected previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val == "" {
				return errors.New("expected non-empty new secrets hmac value")
			}
			if val == currentSecretsHmacValue {
				return errors.New("expected changed secrets hmac value")
			}
			currentSecretsHmacValue = val
			if currentAttributesValue == "" {
				return errors.New("expected previous attributes value")
			}
			if attrs == nil {
				return errors.New("expected non-empty new attributes value")
			}
			attrsVal, err := json.Marshal(attrs)
			if err != nil {
				return fmt.Errorf("error marshaling attrs: %w", err)
			}
			if string(attrsVal) == currentAttributesValue {
				return errors.New("expected changed attrs value")
			}
			currentAttributesValue = string(attrsVal)

		case expectedAttrSecretsHmacStatePreviouslySetNoChange:
			if currentSecretsHmacValue == "" {
				return errors.New("expected previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val == "" {
				return errors.New("expected non-empty new secrets hmac value")
			}
			if val != currentSecretsHmacValue {
				return errors.New("expected same secrets hmac value")
			}
			if currentAttributesValue == "" {
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
			if string(attrsVal) != currentAttributesValue {
				return errors.New("expected same attributes value")
			}

		case expectedAttrSecretsHmacStatePreviouslySetNowEmpty:
			if currentSecretsHmacValue == "" {
				return errors.New("expected previous secrets hmac value")
			}
			if secretsHmac != nil {
				return fmt.Errorf("expected empty new secrets hmac value, got %s", secretsHmac.(string))
			}
			currentSecretsHmacValue = ""
			if currentAttributesValue == "" {
				return errors.New("expected previous attributes value")
			}
			if attrs != nil {
				return fmt.Errorf("expected empty new attributes value, got %s", attrs)
			}
			currentAttributesValue = ""
		}

		return nil
	}
}

func testAccCheckPluginHostCatalogResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_host_catalog":

				id := rs.Primary.ID
				hcClient := hostcatalogs.NewClient(md.client)

				_, err := hcClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed host catalog %q: %v", id, err)
				}

			default:
				continue
			}
		}
		return nil
	}
}