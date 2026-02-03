// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/boundary/testing/vault"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	vaultLdapCredResc            = "boundary_credential_library_vault_ldap.example"
	vaultLdapCredLibName         = "foo"
	vaultLdapCredLibDesc         = "the foo"
	vaultLdapCredLibPath         = "/ldap/static-cred/foo"
	vaultLdapCredLibStringUpdate = "_random"
)

var vaultLdapCredLibResource = fmt.Sprintf(`
resource "boundary_credential_library_vault_ldap" "example" {
	name                = "%s"
	description         = "%s"
	credential_store_id = boundary_credential_store_vault.example.id
  	path                = "%s"
}`, vaultLdapCredLibName,
	vaultLdapCredLibDesc,
	vaultLdapCredLibPath)

var vaultLdapCredLibResourceUpdate = fmt.Sprintf(`
resource "boundary_credential_library_vault_ldap" "example" {
  	name                = "%s"
	description         = "%s"
  	credential_store_id = boundary_credential_store_vault.example.id
  	path                = "%s"
}`, vaultLdapCredLibName+vaultLdapCredLibStringUpdate,
	vaultLdapCredLibDesc+vaultLdapCredLibStringUpdate,
	vaultLdapCredLibPath+vaultLdapCredLibStringUpdate)

func TestAccCredentialLibraryVaultLdap(t *testing.T) {
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
		CheckDestroy:      testAccCheckCredentialLibraryVaultResourceDestroy(t, provider, ldapVaultCredentialLibraryType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultLdapCredLibResource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultLdapCredResc, NameKey, vaultLdapCredLibName),
					resource.TestCheckResourceAttr(vaultLdapCredResc, DescriptionKey, vaultLdapCredLibDesc),
					resource.TestCheckResourceAttr(vaultLdapCredResc, credentialLibraryVaultLdapPathKey, vaultLdapCredLibPath),

					testAccCheckCredentialLibraryResourceExists(provider, vaultLdapCredResc),
				),
			},
			importStep(vaultLdapCredResc),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, vaultLdapCredLibResourceUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(vaultLdapCredResc, NameKey, vaultLdapCredLibName+vaultLdapCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultLdapCredResc, DescriptionKey, vaultLdapCredLibDesc+vaultLdapCredLibStringUpdate),
					resource.TestCheckResourceAttr(vaultLdapCredResc, credentialLibraryVaultLdapPathKey, vaultLdapCredLibPath+vaultLdapCredLibStringUpdate),
					testAccCheckCredentialLibraryResourceExists(provider, vaultLdapCredResc),
				),
			},
			importStep(vaultLdapCredResc),
		},
	})
}
