// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/boundary/api/credentiallibraries"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/boundary/testing/vault"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	vaultSshCertCredResc            = "boundary_credential_library_vault_ssh_certificate.example"
	vaultSshCertCredExtCOResc       = "boundary_credential_library_vault_ssh_certificate.ext_co_example"
	vaultSshCertCredLibName         = "foo"
	vaultSshCertCredLibDesc         = "the foo"
	vaultSshCertCredLibPath         = "/ssh/sign/foo"
	vaultSshCertCredUsername        = "bar"
	vaultSshCertCredLibStringUpdate = "_random"
	vaultSshCertCredKeyType         = "ecdsa"
	vaultSshCertCredKeyBits         = 256
)

var vaultSshCertCredLibResource = fmt.Sprintf(`
resource "boundary_credential_library_vault_ssh_certificate" "example" {
	name                = "%s"
	description         = "%s"
	credential_store_id = boundary_credential_store_vault.example.id
  	path                = "%s"
  	username            = "%s"
	key_type            = "ed25519"
}`, vaultSshCertCredLibName,
	vaultSshCertCredLibDesc,
	vaultSshCertCredLibPath,
	vaultSshCertCredUsername)

var vaultSshCertCredLibResourceUpdate = fmt.Sprintf(`
resource "boundary_credential_library_vault_ssh_certificate" "example" {
  	name                = "%s"
	description         = "%s"
  	credential_store_id = boundary_credential_store_vault.example.id
  	path                = "%s"
  	username            = "%s"
	key_type            = "%s"
	key_bits            = %d
}`, vaultSshCertCredLibName+vaultSshCertCredLibStringUpdate,
	vaultSshCertCredLibDesc+vaultSshCertCredLibStringUpdate,
	vaultSshCertCredLibPath+vaultSshCertCredLibStringUpdate,
	vaultSshCertCredUsername+vaultSshCertCredLibStringUpdate,
	vaultSshCertCredKeyType,
	vaultSshCertCredKeyBits)

var vaultSshCertCredLibResourceExtensionsCriticalOpts = fmt.Sprintf(`
resource "boundary_credential_library_vault_ssh_certificate" "ext_co_example" {
	name                = "%s"
	description         = "%s"
	credential_store_id = boundary_credential_store_vault.example.id
  	path                = "%s"
  	username            = "%s"
	key_type            = "ed25519"
	extensions = {
	  permit-pty = ""
    }
	critical_options = {
	  force-command = "/bin/foo"
	}
}`, vaultSshCertCredLibName,
	vaultSshCertCredLibDesc,
	vaultSshCertCredLibPath,
	vaultSshCertCredUsername)

var vaultSshCertCredLibResourceExtensionsCriticalOptsUpdate = fmt.Sprintf(`
resource "boundary_credential_library_vault_ssh_certificate" "ext_co_example" {
	name                = "%s"
	description         = "%s"
	credential_store_id = boundary_credential_store_vault.example.id
  	path                = "%s"
  	username            = "%s"
	key_type            = "ed25519"
	extensions = {
	  permit-pty            = ""
	  permit-X11-forwarding = ""
    }
}`, vaultSshCertCredLibName,
	vaultSshCertCredLibDesc,
	vaultSshCertCredLibPath,
	vaultSshCertCredUsername)

var vaultSshCertCredLibResourceExtensionsCriticalOptsUpdate2 = fmt.Sprintf(`
resource "boundary_credential_library_vault_ssh_certificate" "ext_co_example" {
	name                = "%s"
	description         = "%s"
	credential_store_id = boundary_credential_store_vault.example.id
  	path                = "%s"
  	username            = "%s"
	key_type            = "ed25519"
	extensions = {
	  permit-pty = ""
    }
	critical_options = {
	  force-command  = "/bin/foo"
	  source-address = "10.10.0.0/16"
	}
}`, vaultSshCertCredLibName,
	vaultSshCertCredLibDesc,
	vaultSshCertCredLibPath,
	vaultSshCertCredUsername)

func TestAccCredentialLibraryVaultSshCertificate(t *testing.T) {
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
		CheckDestroy:      testAccCheckCredentialLibraryVaultResourceDestroy(t, provider, sshCertVaultCredentialLibraryType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultSshCertCredLibResource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultSshCertCredResc, NameKey, vaultSshCertCredLibName),
					resource.TestCheckResourceAttr(vaultSshCertCredResc, DescriptionKey, vaultSshCertCredLibDesc),
					resource.TestCheckResourceAttr(vaultSshCertCredResc, credentialLibraryVaultPathKey, vaultSshCertCredLibPath),
					resource.TestCheckResourceAttr(vaultSshCertCredResc, credentialLibraryVaultSshCertificateUsernameKey, vaultSshCertCredUsername),

					testAccCheckCredentialLibraryVaultSshCertificateResourceExists(provider, vaultSshCertCredResc),
				),
			},
			importStep(vaultSshCertCredResc),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultSshCertCredLibResourceUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultSshCertCredResc, NameKey, vaultSshCertCredLibName+vaultSshCertCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultSshCertCredResc, DescriptionKey, vaultSshCertCredLibDesc+vaultSshCertCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultSshCertCredResc, credentialLibraryVaultPathKey, vaultSshCertCredLibPath+vaultSshCertCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultSshCertCredResc, credentialLibraryVaultSshCertificateUsernameKey, vaultSshCertCredUsername+vaultSshCertCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultSshCertCredResc, credentialLibraryVaultSshCertificateKeyTypeKey, vaultSshCertCredKeyType),
					resource.TestCheckResourceAttr(vaultSshCertCredResc, credentialLibraryVaultSshCertificateKeyBitsKey, strconv.Itoa(vaultSshCertCredKeyBits)),

					testAccCheckCredentialLibraryVaultSshCertificateResourceExists(provider, vaultSshCertCredResc),
				),
			},
			importStep(vaultSshCertCredResc),
			{
				// create with extensions and critical options
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultSshCertCredLibResourceExtensionsCriticalOpts),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, NameKey, vaultSshCertCredLibName),
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, DescriptionKey, vaultSshCertCredLibDesc),
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, credentialLibraryVaultPathKey, vaultSshCertCredLibPath),
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, credentialLibraryVaultSshCertificateUsernameKey, vaultSshCertCredUsername),

					testAccCheckCredentialLibraryVaultSshCertificateResourceExists(provider, vaultSshCertCredExtCOResc),
				),
			},
			importStep(vaultSshCertCredExtCOResc),
			{
				// update with extensions and remove critical options
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultSshCertCredLibResourceExtensionsCriticalOptsUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, NameKey, vaultSshCertCredLibName),
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, DescriptionKey, vaultSshCertCredLibDesc),
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, credentialLibraryVaultPathKey, vaultSshCertCredLibPath),
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, credentialLibraryVaultSshCertificateUsernameKey, vaultSshCertCredUsername),
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, credentialLibraryVaultSshCertificateCriticalOptionsKey+".%", "0"),
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, credentialLibraryVaultSshCertificateExtensionsKey+".%", "2"),

					testAccCheckCredentialLibraryVaultSshCertificateResourceExists(provider, vaultSshCertCredExtCOResc),
				),
			},
			importStep(vaultSshCertCredExtCOResc),
			{
				// update with extensions and remove critical options
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultSshCertCredLibResourceExtensionsCriticalOptsUpdate2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, NameKey, vaultSshCertCredLibName),
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, DescriptionKey, vaultSshCertCredLibDesc),
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, credentialLibraryVaultPathKey, vaultSshCertCredLibPath),
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, credentialLibraryVaultSshCertificateUsernameKey, vaultSshCertCredUsername),
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, credentialLibraryVaultSshCertificateCriticalOptionsKey+".%", "2"),
					resource.TestCheckResourceAttr(vaultSshCertCredExtCOResc, credentialLibraryVaultSshCertificateExtensionsKey+".%", "1"),

					testAccCheckCredentialLibraryVaultSshCertificateResourceExists(provider, vaultSshCertCredExtCOResc),
				),
			},
			importStep(vaultSshCertCredExtCOResc),
		},
	})
}

func testAccCheckCredentialLibraryVaultSshCertificateResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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
