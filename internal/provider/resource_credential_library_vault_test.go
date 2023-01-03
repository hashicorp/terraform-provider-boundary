// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentiallibraries"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/boundary/testing/vault"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	vaultCredResc                 = "boundary_credential_library_vault.example"
	vaultCredTypedResc            = "boundary_credential_library_vault.typed_example"
	vaultCredUsernamePasswordResc = "boundary_credential_library_vault.username_password_mapping_override"
	vaultCredSshPrivateKeyResc    = "boundary_credential_library_vault.ssh_private_key_mapping_override"
	vaultCredLibName              = "foo"
	vaultCredLibDesc              = "the foo"
	vaultCredLibPath              = "/foo/bar"
	vaultCredLibMethodGet         = "GET"
	vaultCredLibMethodPost        = "POST"
	vaultCredLibRequestBody       = "foobar"
	vaultCredLibStringUpdate      = "_random"
)

var vaultCredLibResource = fmt.Sprintf(`
resource "boundary_credential_library_vault" "example" {
	name                = "%s"
	description         = "%s"
	credential_store_id = boundary_credential_store_vault.example.id
  	path                = "%s"
  	http_method         = "%s"
}`, vaultCredLibName,
	vaultCredLibDesc,
	vaultCredLibPath,
	vaultCredLibMethodGet)

var vaultCredLibResourceUpdate = fmt.Sprintf(`
resource "boundary_credential_library_vault" "example" {
  	name                = "%s"
	description         = "%s"
  	credential_store_id = boundary_credential_store_vault.example.id
  	path                = "%s"
  	http_method         = "%s"
  	http_request_body   = "%s"
}`, vaultCredLibName+vaultCredLibStringUpdate,
	vaultCredLibDesc+vaultCredLibStringUpdate,
	vaultCredLibPath+vaultCredLibStringUpdate,
	vaultCredLibMethodPost,
	vaultCredLibRequestBody)

var vaultTypedCredLibResource = fmt.Sprintf(`
resource "boundary_credential_library_vault" "typed_example" {
	name                = "%s"
	description         = "%s"
	credential_store_id = boundary_credential_store_vault.example.id
  	path                = "%s"
  	http_method         = "%s"
	credential_type     = "ssh_private_key"
}`, vaultCredLibName,
	vaultCredLibDesc,
	vaultCredLibPath,
	vaultCredLibMethodGet)

var vaultUsernamePasswordMappingOverrideCredLibResource = fmt.Sprintf(`
resource "boundary_credential_library_vault" "username_password_mapping_override" {
	name                         = "%s"
	description                  = "%s"
	credential_store_id          = boundary_credential_store_vault.example.id
	path                         = "%s"
	http_method                  = "%s"
	credential_type              = "username_password"
	credential_mapping_overrides = {
		password_attribute = "alternative_password_label"
		username_attribute = "alternative_username_label"
	}
}`, vaultCredLibName,
	vaultCredLibDesc,
	vaultCredLibPath,
	vaultCredLibMethodGet)

var vaultUsernamePasswordMappingOverrideCredLibResourceUpdate = fmt.Sprintf(`
	resource "boundary_credential_library_vault" "username_password_mapping_override" {
		name                         = "%s"
		description                  = "%s"
		credential_store_id          = boundary_credential_store_vault.example.id
		path                         = "%s"
		http_method                  = "%s"
		credential_type              = "username_password"
		credential_mapping_overrides = {
			password_attribute = "updated_password_label"
			username_attribute = "updated_username_label"
		}
	}`, vaultCredLibName,
	vaultCredLibDesc,
	vaultCredLibPath,
	vaultCredLibMethodGet)

var vaultUsernamePasswordMappingOverrideCredLibResourceRemove = fmt.Sprintf(`
	resource "boundary_credential_library_vault" "username_password_mapping_override" {
		name                         = "%s"
		description                  = "%s"
		credential_store_id          = boundary_credential_store_vault.example.id
		path                         = "%s"
		http_method                  = "%s"
		credential_type              = "username_password"
	}`, vaultCredLibName,
	vaultCredLibDesc,
	vaultCredLibPath,
	vaultCredLibMethodGet)

var vaultSshPrivateKeyMappingOverrideCredLibResource = fmt.Sprintf(`
resource "boundary_credential_library_vault" "ssh_private_key_mapping_override" {
	name                         = "%s"
	description                  = "%s"
	credential_store_id          = boundary_credential_store_vault.example.id
	path                         = "%s"
	http_method                  = "%s"
	credential_type              = "ssh_private_key"
	credential_mapping_overrides = {
		private_key_attribute 			 = "alternative_key_label"
		private_key_passphrase_attribute = "alternative_passphrase_label"
		username_attribute 				 = "alternative_username_label"
	}
}`, vaultCredLibName,
	vaultCredLibDesc,
	vaultCredLibPath,
	vaultCredLibMethodGet)

func TestAccCredentialLibraryVault(t *testing.T) {
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
		IsUnitTest:        true,
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialLibraryVaultResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultCredLibResource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredResc, NameKey, vaultCredLibName),
					resource.TestCheckResourceAttr(vaultCredResc, DescriptionKey, vaultCredLibDesc),
					resource.TestCheckResourceAttr(vaultCredResc, credentialLibraryVaultPathKey, vaultCredLibPath),
					resource.TestCheckResourceAttr(vaultCredResc, credentialLibraryVaultHttpMethodKey, vaultCredLibMethodGet),
					resource.TestCheckResourceAttr(vaultCredResc, credentialLibraryVaultHttpRequestBodyKey, ""),

					testAccCheckCredentialLibraryVaultResourceExists(provider, vaultCredResc),
				),
			},
			importStep(vaultCredResc),

			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultCredLibResourceUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredResc, NameKey, vaultCredLibName+vaultCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultCredResc, DescriptionKey, vaultCredLibDesc+vaultCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultCredResc, credentialLibraryVaultPathKey, vaultCredLibPath+vaultCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultCredResc, credentialLibraryVaultHttpMethodKey, vaultCredLibMethodPost),
					resource.TestCheckResourceAttr(vaultCredResc, credentialLibraryVaultHttpRequestBodyKey, vaultCredLibRequestBody),

					testAccCheckCredentialLibraryVaultResourceExists(provider, vaultCredResc),
				),
			},
			importStep(vaultCredResc),

			{
				// create typed credential library, note credential type is immutable so no need for update test
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultCredLibResourceUpdate, vaultTypedCredLibResource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredTypedResc, NameKey, vaultCredLibName),
					resource.TestCheckResourceAttr(vaultCredTypedResc, DescriptionKey, vaultCredLibDesc),
					resource.TestCheckResourceAttr(vaultCredTypedResc, credentialLibraryVaultPathKey, vaultCredLibPath),
					resource.TestCheckResourceAttr(vaultCredTypedResc, credentialLibraryVaultHttpMethodKey, vaultCredLibMethodGet),
					resource.TestCheckResourceAttr(vaultCredTypedResc, credentialLibraryVaultHttpRequestBodyKey, ""),
					resource.TestCheckResourceAttr(vaultCredTypedResc, credentialLibraryCredentialTypeKey, "ssh_private_key"),

					testAccCheckCredentialLibraryVaultResourceExists(provider, vaultCredTypedResc),
				),
			},
			importStep(vaultCredResc),

			{
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultCredLibResourceUpdate, vaultUsernamePasswordMappingOverrideCredLibResource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, NameKey, vaultCredLibName),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, DescriptionKey, vaultCredLibDesc),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, credentialLibraryVaultPathKey, vaultCredLibPath),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, credentialLibraryVaultHttpMethodKey, vaultCredLibMethodGet),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, credentialLibraryVaultHttpRequestBodyKey, ""),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, credentialLibraryCredentialTypeKey, "username_password"),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, "credential_mapping_overrides.password_attribute", "alternative_password_label"),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, "credential_mapping_overrides.username_attribute", "alternative_username_label"),

					testAccCheckCredentialLibraryVaultResourceExists(provider, vaultCredUsernamePasswordResc),
				),
			},
			importStep(vaultCredResc),

			{
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultCredLibResourceUpdate, vaultUsernamePasswordMappingOverrideCredLibResourceUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, NameKey, vaultCredLibName),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, DescriptionKey, vaultCredLibDesc),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, credentialLibraryVaultPathKey, vaultCredLibPath),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, credentialLibraryVaultHttpMethodKey, vaultCredLibMethodGet),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, credentialLibraryVaultHttpRequestBodyKey, ""),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, credentialLibraryCredentialTypeKey, "username_password"),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, "credential_mapping_overrides.password_attribute", "updated_password_label"),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, "credential_mapping_overrides.username_attribute", "updated_username_label"),

					testAccCheckCredentialLibraryVaultResourceExists(provider, vaultCredUsernamePasswordResc),
				),
			},
			importStep(vaultCredResc),

			{
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultCredLibResourceUpdate, vaultUsernamePasswordMappingOverrideCredLibResourceRemove),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, NameKey, vaultCredLibName),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, DescriptionKey, vaultCredLibDesc),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, credentialLibraryVaultPathKey, vaultCredLibPath),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, credentialLibraryVaultHttpMethodKey, vaultCredLibMethodGet),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, credentialLibraryVaultHttpRequestBodyKey, ""),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, credentialLibraryCredentialTypeKey, "username_password"),
					resource.TestCheckResourceAttr(vaultCredUsernamePasswordResc, credentialLibraryCredentialMappingOverridesKey+".%", "0"),

					testAccCheckCredentialLibraryVaultResourceExists(provider, vaultCredUsernamePasswordResc),
				),
			},
			importStep(vaultCredResc),

			{
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultCredLibResourceUpdate, vaultSshPrivateKeyMappingOverrideCredLibResource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultCredSshPrivateKeyResc, NameKey, vaultCredLibName),
					resource.TestCheckResourceAttr(vaultCredSshPrivateKeyResc, DescriptionKey, vaultCredLibDesc),
					resource.TestCheckResourceAttr(vaultCredSshPrivateKeyResc, credentialLibraryVaultPathKey, vaultCredLibPath),
					resource.TestCheckResourceAttr(vaultCredSshPrivateKeyResc, credentialLibraryVaultHttpMethodKey, vaultCredLibMethodGet),
					resource.TestCheckResourceAttr(vaultCredSshPrivateKeyResc, credentialLibraryVaultHttpRequestBodyKey, ""),
					resource.TestCheckResourceAttr(vaultCredSshPrivateKeyResc, credentialLibraryCredentialTypeKey, "ssh_private_key"),
					resource.TestCheckResourceAttr(vaultCredSshPrivateKeyResc, "credential_mapping_overrides.private_key_attribute", "alternative_key_label"),
					resource.TestCheckResourceAttr(vaultCredSshPrivateKeyResc, "credential_mapping_overrides.private_key_passphrase_attribute", "alternative_passphrase_label"),
					resource.TestCheckResourceAttr(vaultCredSshPrivateKeyResc, "credential_mapping_overrides.username_attribute", "alternative_username_label"),

					testAccCheckCredentialLibraryVaultResourceExists(provider, vaultCredSshPrivateKeyResc),
				),
			},
			importStep(vaultCredResc),
		},
	})
}

func testAccCheckCredentialLibraryVaultResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("no ID is set")
		}

		md := testProvider.Meta().(*metaData)
		c := credentiallibraries.NewClient(md.client)
		if _, err := c.Read(context.Background(), id); err != nil {
			return fmt.Errorf("got an error reading %q: %w", id, err)
		}

		return nil
	}
}

func testAccCheckCredentialLibraryVaultResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_credential_library_vault":
				id := rs.Primary.ID

				c := credentiallibraries.NewClient(md.client)
				_, err := c.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed vault credential library %q: %v", id, err)
				}

			default:
				continue
			}
		}
		return nil
	}
}
