// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"golang.org/x/crypto/ssh/testdata"
)

const (
	sshPrivateKeyCredResc = "boundary_credential_ssh_private_key.example"
	sshPrivateKeyCredName = "Dr Jekyll"
	sshPrivateKeyCredDesc = "my best description"
	sshPrivateKeyUpdate   = "some magic update string"
	sshPrivateKeyUsername = "my-user"
)

var staticStore = fmt.Sprintf(`
resource "boundary_credential_store_static" "ssh_store" {
	name        = "static store name"
	description = "static store description"
	scope_id    = boundary_scope.proj1.id
	depends_on  = [boundary_role.proj1_admin]
}`)

func sshPrivateKeyResource(name, description, username, privateKey, passphrase string) string {
	return fmt.Sprintf(`
resource "boundary_credential_ssh_private_key" "example" {
	name                   = %q
	description            = %q
	credential_store_id    = boundary_credential_store_static.ssh_store.id
	username               = %q
	private_key            = %q
	private_key_passphrase = %q
}`, name, description, username, privateKey, passphrase)
}

func TestAccCredentialSshPrivateKey(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	privKey := string(testdata.PEMBytes["rsa"])
	privKeyUpdate := string(testdata.PEMEncryptedKeys[0].PEMBytes)
	privKeyUpdatePassphrase := testdata.PEMEncryptedKeys[0].EncryptionKey

	res := sshPrivateKeyResource(
		sshPrivateKeyCredName,
		sshPrivateKeyCredDesc,
		sshPrivateKeyUsername,
		privKey,
		"",
	)

	resUpdate := sshPrivateKeyResource(
		sshPrivateKeyCredName+sshPrivateKeyUpdate,
		sshPrivateKeyCredDesc+sshPrivateKeyUpdate,
		sshPrivateKeyUsername+sshPrivateKeyUpdate,
		privKeyUpdate,
		privKeyUpdatePassphrase,
	)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialResourceDestroy(t, provider, sshPrivateKeyCredentialType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, staticStore, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(sshPrivateKeyCredResc, NameKey, sshPrivateKeyCredName),
					resource.TestCheckResourceAttr(sshPrivateKeyCredResc, DescriptionKey, sshPrivateKeyCredDesc),
					resource.TestCheckResourceAttr(sshPrivateKeyCredResc, credentialSshPrivateKeyUsernameKey, sshPrivateKeyUsername),
					resource.TestCheckResourceAttr(sshPrivateKeyCredResc, credentialSshPrivateKeyPrivateKeyKey, privKey),
					resource.TestCheckResourceAttr(sshPrivateKeyCredResc, credentialSshPrivateKeyPassphraseKey, ""),

					testAccCheckCredentialStoreSshPrivateKeyHmac(),
					testAccCheckCredentialResourceExists(provider, sshPrivateKeyCredResc),
				),
			},
			importStep(sshPrivateKeyCredResc, credentialSshPrivateKeyPrivateKeyKey, credentialSshPrivateKeyPassphraseKey),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, staticStore, resUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(sshPrivateKeyCredResc, NameKey, sshPrivateKeyCredName+sshPrivateKeyUpdate),
					resource.TestCheckResourceAttr(sshPrivateKeyCredResc, DescriptionKey, sshPrivateKeyCredDesc+sshPrivateKeyUpdate),
					resource.TestCheckResourceAttr(sshPrivateKeyCredResc, credentialSshPrivateKeyUsernameKey, sshPrivateKeyUsername+sshPrivateKeyUpdate),
					resource.TestCheckResourceAttr(sshPrivateKeyCredResc, credentialSshPrivateKeyPrivateKeyKey, privKeyUpdate),
					resource.TestCheckResourceAttr(sshPrivateKeyCredResc, credentialSshPrivateKeyPassphraseKey, privKeyUpdatePassphrase),

					testAccCheckCredentialStoreSshPrivateKeyHmac(),
					testAccCheckCredentialResourceExists(provider, sshPrivateKeyCredResc),
				),
			},
		},
	})
}

func testAccCheckCredentialStoreSshPrivateKeyHmac() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[sshPrivateKeyCredResc]
		if !ok {
			return fmt.Errorf("not found: %s", sshPrivateKeyCredResc)
		}

		computed := rs.Primary.Attributes["private_key_hmac"]
		if len(computed) != 43 {
			return fmt.Errorf("computed private key hmac not the expected length of 43 characters, got: %q", computed)
		}

		if rs.Primary.Attributes["private_key_passphrase"] != "" {
			// We set a passphrase, validate the computed hmac is expected length
			computed := rs.Primary.Attributes["private_key_passphrase_hmac"]
			if len(computed) != 43 {
				return fmt.Errorf("computed private key passphrase hmac not the expected length of 43 characters, got: %q", computed)
			}
		}

		return nil
	}
}
