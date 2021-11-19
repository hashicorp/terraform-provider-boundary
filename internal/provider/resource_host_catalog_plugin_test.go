package provider

import (
	"context"
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
	testPluginHostCatalogZipAttr            = "zap"
	testPluginHostCatalogZipAttrUpdate      = "zoop"
)

type expectedSecretsHmacState uint

const (
	expectedSecretsHmacStatePreviouslyEmptyNowSet expectedSecretsHmacState = iota
	expectedSecretsHmacStatePreviouslySetNoChange
	expectedSecretsHmacStatePreviouslySetButChanged
	expectedSecretsHmacStatePreviouslySetNowEmpty
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

var currentSecretsHmacValue string

func TestAccPluginHostCatalogCreate(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resName := "boundary_host_catalog_plugin.foo"
	initialValuesStr := fmt.Sprintf(`
	description = "%s"
	attributes = {
		foo = "bar"
		zip = "%s"
	}
	secrets = {
		hush = "puppies"
	}
	`,
		testPluginHostCatalogDescription,
		testPluginHostCatalogZipAttr,
	)

	// Changed description and secrets
	valuesStrUpdate1 := fmt.Sprintf(`
	description = "%s"
	attributes = {
		foo = "bar"
		zip = "%s"
	}
	secrets = {
		flush = "fluppies"
	}
	`,
		testPluginHostCatalogDescriptionUpdate,
		testPluginHostCatalogZipAttrUpdate,
	)

	// Changed description, no secrets update
	valuesStrUpdate2 := fmt.Sprintf(`
	description = "%s"
	attributes = {
		foo = "bar"
		zip = "%s"
	}
	secrets = {
		flush = "fluppies"
	}
	`,
		testPluginHostCatalogDescriptionUpdate2,
		testPluginHostCatalogZipAttrUpdate,
	)

	// Same description, now empty secrets
	valuesStrUpdate3 := fmt.Sprintf(`
		description = "%s"
		attributes = {
			foo = "bar"
			zip = "%s"
		}
		`,
		testPluginHostCatalogDescriptionUpdate2,
		testPluginHostCatalogZipAttrUpdate,
	)

	// Same description, now explicitly unset secrets
	valuesStrUpdate4 := fmt.Sprintf(`
		description = "%s"
		attributes = {
			foo = "bar"
			zip = "%s"
		}
		secrets = {}
		`,
		testPluginHostCatalogDescriptionUpdate2,
		testPluginHostCatalogZipAttrUpdate,
	)

	initialHcl := fmt.Sprintf(projPluginHostCatalogBase, initialValuesStr)
	update1Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate1)
	update2Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate2)
	update3Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate3)
	update4Hcl := fmt.Sprintf(projPluginHostCatalogBase, valuesStrUpdate4)
	_ = update4Hcl

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
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedSecretsHmacStatePreviouslyEmptyNowSet),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescription),
				),
			},
			importStep(resName, "secrets"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update1Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedSecretsHmacStatePreviouslySetButChanged),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate),
				),
			},
			importStep(resName, "secrets"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update2Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedSecretsHmacStatePreviouslySetNoChange),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
			},
			importStep(resName, "secrets"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update3Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedSecretsHmacStatePreviouslySetNoChange),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
			},
			importStep(resName, "secrets"),

			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, update4Hcl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginHostCatalogResourceExists(provider, resName, expectedSecretsHmacStatePreviouslySetNowEmpty),
					resource.TestCheckResourceAttr(resName, DescriptionKey, testPluginHostCatalogDescriptionUpdate2),
				),
			},
			importStep(resName, "secrets"),
		},
	})
}

func testAccCheckPluginHostCatalogResourceExists(testProvider *schema.Provider, name string, expSecrets expectedSecretsHmacState) resource.TestCheckFunc {
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
		switch expSecrets {
		case expectedSecretsHmacStatePreviouslyEmptyNowSet:
			if currentSecretsHmacValue != "" {
				return errors.New("expected no previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val == "" {
				return errors.New("expected non-empty new secrets hmac value")
			}
			currentSecretsHmacValue = val

		case expectedSecretsHmacStatePreviouslySetButChanged:
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

		case expectedSecretsHmacStatePreviouslySetNoChange:
			if currentSecretsHmacValue == "" {
				return fmt.Errorf("expected previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val == "" {
				return errors.New("expected non-empty new secrets hmac value")
			}
			if val != currentSecretsHmacValue {
				return errors.New("expected same secrets hmac value")
			}

		case expectedSecretsHmacStatePreviouslySetNowEmpty:
			if currentSecretsHmacValue == "" {
				return fmt.Errorf("expected previous secrets hmac value")
			}
			val := secretsHmac.(string)
			if val != "" {
				return errors.New("expected empty new secrets hmac value")
			}
			currentSecretsHmacValue = ""
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
