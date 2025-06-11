// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/boundary/api/credentials"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	usernamePasswordDomainCredResc          = "boundary_credential_username_password_domain.example"
	usernamePasswordDomainCredName          = "foo"
	usernamePasswordDomainCredDesc          = "the foo"
	usernamePasswordDomainCredUsername      = "default_username"
	usernamePasswordDomainCredUsernameAt    = "default_username@default_domain"
	usernamePasswordDomainCredUsernameSlash = `default_domain\\default_username`
	usernamePasswordDomainCredPassword      = "default_password"
	usernamePasswordDomainCredDomain        = "default_domain"
	usernamePasswordDomainCredUpdate        = "_random"
)

func usernamePasswordDomainCredResourceWithoutDomain(name, description, username, password string) string {
	return fmt.Sprintf(`
resource "boundary_credential_store_static" "example" {
	name  = "static store name"
	description = "static store description"
	scope_id = boundary_scope.proj1.id
	depends_on = [boundary_role.proj1_admin]
}

resource "boundary_credential_username_password_domain" "example" {
	name  = "%s"
	description = "%s"
	credential_store_id = boundary_credential_store_static.example.id
	username = "%s"
	password = "%s"
}`, name,
		description,
		username,
		password)
}

func usernamePasswordDomainCredResourceWithDomain(name, description, username, password, domain string) string {
	return fmt.Sprintf(`
resource "boundary_credential_store_static" "example" {
	name  = "static store name"
	description = "static store description"
	scope_id = boundary_scope.proj1.id
	depends_on = [boundary_role.proj1_admin]
}

resource "boundary_credential_username_password_domain" "example" {
	name  = "%s"
	description = "%s"
	credential_store_id = boundary_credential_store_static.example.id
	username = "%s"
	password = "%s"
	domain = "%s"
}`, name,
		description,
		username,
		password,
		domain)
}

func TestAccCredentialUsernamePasswordDomain(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	res := usernamePasswordDomainCredResourceWithDomain(
		usernamePasswordDomainCredName,
		usernamePasswordDomainCredDesc,
		usernamePasswordDomainCredUsername,
		usernamePasswordDomainCredPassword,
		usernamePasswordDomainCredDomain,
	)

	resUpdate := usernamePasswordDomainCredResourceWithDomain(
		usernamePasswordDomainCredName,
		usernamePasswordDomainCredDesc,
		usernamePasswordDomainCredUsername+usernamePasswordDomainCredUpdate,
		usernamePasswordDomainCredPassword+usernamePasswordDomainCredUpdate,
		usernamePasswordDomainCredDomain+usernamePasswordDomainCredUpdate,
	)

	resUpdateAt := usernamePasswordDomainCredResourceWithoutDomain(
		usernamePasswordDomainCredName,
		usernamePasswordDomainCredDesc,
		fmt.Sprintf("%s@%s", usernamePasswordDomainCredUsername+usernamePasswordDomainCredUpdate, usernamePasswordDomainCredDomain+usernamePasswordDomainCredUpdate),
		usernamePasswordDomainCredPassword+usernamePasswordDomainCredUpdate,
	)

	resUpdateSlash := usernamePasswordDomainCredResourceWithoutDomain(
		usernamePasswordDomainCredName,
		usernamePasswordDomainCredDesc,
		fmt.Sprintf(`%s\\%s`, usernamePasswordDomainCredDomain+usernamePasswordDomainCredUpdate, usernamePasswordDomainCredUsername+usernamePasswordDomainCredUpdate),
		usernamePasswordDomainCredPassword+usernamePasswordDomainCredUpdate,
	)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialResourceDestroy(t, provider, usernamePasswordDomainCredentialType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, NameKey, usernamePasswordDomainCredName),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, DescriptionKey, usernamePasswordDomainCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainUsernameKey, usernamePasswordDomainCredUsername),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey, usernamePasswordDomainCredPassword),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainDomainKey, usernamePasswordDomainCredDomain),

					testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, usernamePasswordDomainCredResc),
				),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, resUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, NameKey, usernamePasswordDomainCredName),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, DescriptionKey, usernamePasswordDomainCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainUsernameKey, usernamePasswordDomainCredUsername+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey, usernamePasswordDomainCredPassword+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainDomainKey, usernamePasswordDomainCredDomain+usernamePasswordDomainCredUpdate),

					testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, usernamePasswordDomainCredResc),
				),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
			{
				// run a plan only update and verify no changes
				PlanOnly: true,
				Config:   testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
			{
				// update again but apply a preConfig to externally update resource
				PreConfig: func() { usernamePasswordDomainCredExternalUpdate(t, provider) },
				Config:    testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
		},
	})

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialResourceDestroy(t, provider, usernamePasswordDomainCredentialType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, NameKey, usernamePasswordDomainCredName),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, DescriptionKey, usernamePasswordDomainCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainUsernameKey, usernamePasswordDomainCredUsername),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey, usernamePasswordDomainCredPassword),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainDomainKey, usernamePasswordDomainCredDomain),

					testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, usernamePasswordDomainCredResc),
				),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
			{
				// update from 3 distinct fields to username@domain.
				Config: testConfig(url, fooOrg, firstProjectFoo, resUpdateAt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, NameKey, usernamePasswordDomainCredName),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, DescriptionKey, usernamePasswordDomainCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainUsernameKey, usernamePasswordDomainCredUsername+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey, usernamePasswordDomainCredPassword+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainDomainKey, usernamePasswordDomainCredDomain+usernamePasswordDomainCredUpdate),

					testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, usernamePasswordDomainCredResc),
				),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
		},
	})

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialResourceDestroy(t, provider, usernamePasswordDomainCredentialType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, NameKey, usernamePasswordDomainCredName),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, DescriptionKey, usernamePasswordDomainCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainUsernameKey, usernamePasswordDomainCredUsername),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey, usernamePasswordDomainCredPassword),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainDomainKey, usernamePasswordDomainCredDomain),

					testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, usernamePasswordDomainCredResc),
				),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
			{
				// update from 3 distinct fields to domain\username.
				Config: testConfig(url, fooOrg, firstProjectFoo, resUpdateSlash),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, NameKey, usernamePasswordDomainCredName),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, DescriptionKey, usernamePasswordDomainCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainUsernameKey, usernamePasswordDomainCredUsername+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey, usernamePasswordDomainCredPassword+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainDomainKey, usernamePasswordDomainCredDomain+usernamePasswordDomainCredUpdate),

					testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, usernamePasswordDomainCredResc),
				),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
		},
	})
}

func TestAccCredentialUsernamePasswordDomain_DomainInUsernameFieldUsingAt(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	res := usernamePasswordDomainCredResourceWithoutDomain(
		usernamePasswordDomainCredName,
		usernamePasswordDomainCredDesc,
		usernamePasswordDomainCredUsernameAt,
		usernamePasswordDomainCredPassword,
	)

	resUpdate := usernamePasswordDomainCredResourceWithoutDomain(
		usernamePasswordDomainCredName,
		usernamePasswordDomainCredDesc,
		fmt.Sprintf("%s@%s", usernamePasswordDomainCredUsername+usernamePasswordDomainCredUpdate, usernamePasswordDomainCredDomain+usernamePasswordDomainCredUpdate),
		usernamePasswordDomainCredPassword+usernamePasswordDomainCredUpdate,
	)

	resUpdateSeparateFields := usernamePasswordDomainCredResourceWithDomain(
		usernamePasswordDomainCredName,
		usernamePasswordDomainCredDesc,
		usernamePasswordCredUsername+usernamePasswordCredUpdate+usernamePasswordCredUpdate,
		usernamePasswordDomainCredPassword+usernamePasswordCredUpdate+usernamePasswordCredUpdate,
		usernamePasswordDomainCredDomain+usernamePasswordCredUpdate+usernamePasswordCredUpdate,
	)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialResourceDestroy(t, provider, usernamePasswordDomainCredentialType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, NameKey, usernamePasswordDomainCredName),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, DescriptionKey, usernamePasswordDomainCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainUsernameKey, usernamePasswordDomainCredUsername),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey, usernamePasswordDomainCredPassword),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainDomainKey, usernamePasswordDomainCredDomain),

					testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, usernamePasswordDomainCredResc),
				),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, resUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, NameKey, usernamePasswordDomainCredName),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, DescriptionKey, usernamePasswordDomainCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainUsernameKey, usernamePasswordDomainCredUsername+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey, usernamePasswordDomainCredPassword+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainDomainKey, usernamePasswordDomainCredDomain+usernamePasswordDomainCredUpdate),

					testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, usernamePasswordDomainCredResc),
				),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
			{
				// Run a plan only update and verify no changes
				PlanOnly: true,
				Config:   testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
			{
				// update again but apply a preConfig to externally update resource
				PreConfig: func() { usernamePasswordDomainCredExternalUpdate(t, provider) },
				Config:    testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
			{
				// update from username@domain to use all 3 fields separately.
				Config: testConfig(url, fooOrg, firstProjectFoo, resUpdateSeparateFields),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, NameKey, usernamePasswordDomainCredName),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, DescriptionKey, usernamePasswordDomainCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainUsernameKey, usernamePasswordDomainCredUsername+usernamePasswordDomainCredUpdate+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey, usernamePasswordDomainCredPassword+usernamePasswordDomainCredUpdate+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainDomainKey, usernamePasswordDomainCredDomain+usernamePasswordDomainCredUpdate+usernamePasswordDomainCredUpdate),

					testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, usernamePasswordDomainCredResc),
				),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
		},
	})
}

func TestAccCredentialUsernamePasswordDomain_DomainInUsernameFieldUsingSlash(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	res := usernamePasswordDomainCredResourceWithoutDomain(
		usernamePasswordDomainCredName,
		usernamePasswordDomainCredDesc,
		usernamePasswordDomainCredUsernameSlash,
		usernamePasswordDomainCredPassword,
	)

	resUpdate := usernamePasswordDomainCredResourceWithoutDomain(
		usernamePasswordDomainCredName,
		usernamePasswordDomainCredDesc,
		fmt.Sprintf(`%s\\%s`, usernamePasswordDomainCredDomain+usernamePasswordDomainCredUpdate, usernamePasswordDomainCredUsername+usernamePasswordDomainCredUpdate),
		usernamePasswordDomainCredPassword+usernamePasswordDomainCredUpdate,
	)

	resUpdateSeparateFields := usernamePasswordDomainCredResourceWithDomain(
		usernamePasswordDomainCredName,
		usernamePasswordDomainCredDesc,
		usernamePasswordCredUsername+usernamePasswordCredUpdate+usernamePasswordCredUpdate,
		usernamePasswordDomainCredPassword+usernamePasswordCredUpdate+usernamePasswordCredUpdate,
		usernamePasswordDomainCredDomain+usernamePasswordCredUpdate+usernamePasswordCredUpdate,
	)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialResourceDestroy(t, provider, usernamePasswordDomainCredentialType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, NameKey, usernamePasswordDomainCredName),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, DescriptionKey, usernamePasswordDomainCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainUsernameKey, usernamePasswordDomainCredUsername),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey, usernamePasswordDomainCredPassword),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainDomainKey, usernamePasswordDomainCredDomain),

					testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, usernamePasswordDomainCredResc),
				),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, resUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, NameKey, usernamePasswordDomainCredName),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, DescriptionKey, usernamePasswordDomainCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainUsernameKey, usernamePasswordDomainCredUsername+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey, usernamePasswordDomainCredPassword+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainDomainKey, usernamePasswordDomainCredDomain+usernamePasswordDomainCredUpdate),

					testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, usernamePasswordDomainCredResc),
				),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
			{
				// Run a plan only update and verify no changes
				PlanOnly: true,
				Config:   testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
			{
				// update again but apply a preConfig to externally update resource
				PreConfig: func() { usernamePasswordDomainCredExternalUpdate(t, provider) },
				Config:    testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
			{
				// update from domain\username to use all 3 fields separately.
				Config: testConfig(url, fooOrg, firstProjectFoo, resUpdateSeparateFields),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, NameKey, usernamePasswordDomainCredName),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, DescriptionKey, usernamePasswordDomainCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainUsernameKey, usernamePasswordDomainCredUsername+usernamePasswordDomainCredUpdate+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey, usernamePasswordDomainCredPassword+usernamePasswordDomainCredUpdate+usernamePasswordDomainCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainDomainKey, usernamePasswordDomainCredDomain+usernamePasswordDomainCredUpdate+usernamePasswordDomainCredUpdate),

					testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, usernamePasswordDomainCredResc),
				),
			},
			importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
		},
	})
}

// TestResourceCredentialUsernamePasswordDomainCustomizeDiff tests the diff
// customization function by testing the diff equivalency of different
// combinations of inputs.
func TestResourceCredentialUsernamePasswordDomainCustomizeDiff(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	tests := []struct {
		name            string
		createResource  string
		planOnlyUpdates []string
	}{
		{
			name: "usernamePasswordDomainFormatToOtherFormats",
			createResource: usernamePasswordDomainCredResourceWithDomain( // Start w/ username, password, domain.
				usernamePasswordDomainCredName,
				usernamePasswordDomainCredDesc,
				usernamePasswordDomainCredUsername,
				usernamePasswordDomainCredPassword,
				usernamePasswordDomainCredDomain,
			),
			planOnlyUpdates: []string{
				// Plan an update to username@domain (with domain field still filled).
				usernamePasswordDomainCredResourceWithDomain(
					usernamePasswordDomainCredName,
					usernamePasswordDomainCredDesc,
					fmt.Sprintf("%s@%s", usernamePasswordDomainCredUsername, usernamePasswordDomainCredDomain),
					usernamePasswordDomainCredPassword,
					usernamePasswordDomainCredDomain,
				),
				// Plan an update to username@domain and empty out domain.
				usernamePasswordDomainCredResourceWithDomain(
					usernamePasswordDomainCredName,
					usernamePasswordDomainCredDesc,
					fmt.Sprintf("%s@%s", usernamePasswordDomainCredUsername, usernamePasswordDomainCredDomain),
					usernamePasswordDomainCredPassword,
					"",
				),
				// Plan an update to username@domain and nil out domain.
				usernamePasswordDomainCredResourceWithoutDomain(
					usernamePasswordDomainCredName,
					usernamePasswordDomainCredDesc,
					fmt.Sprintf("%s@%s", usernamePasswordDomainCredUsername, usernamePasswordDomainCredDomain),
					usernamePasswordDomainCredPassword,
				),
				// Plan an update to domain\username (with domain field still filled).
				usernamePasswordDomainCredResourceWithDomain(
					usernamePasswordDomainCredName,
					usernamePasswordDomainCredDesc,
					fmt.Sprintf(`%s\\%s`, usernamePasswordDomainCredDomain, usernamePasswordDomainCredUsername),
					usernamePasswordDomainCredPassword,
					usernamePasswordDomainCredDomain,
				),
				// Plan an update to domain\username and empty out domain.
				usernamePasswordDomainCredResourceWithDomain(
					usernamePasswordDomainCredName,
					usernamePasswordDomainCredDesc,
					fmt.Sprintf(`%s\\%s`, usernamePasswordDomainCredDomain, usernamePasswordDomainCredUsername),
					usernamePasswordDomainCredPassword,
					"",
				),
				// Plan an update to domain\username and nil out domain.
				usernamePasswordDomainCredResourceWithoutDomain(
					usernamePasswordDomainCredName,
					usernamePasswordDomainCredDesc,
					fmt.Sprintf(`%s\\%s`, usernamePasswordDomainCredDomain, usernamePasswordDomainCredUsername),
					usernamePasswordDomainCredPassword,
				),
			},
		},
		{
			name: "usernameAtpasswordFormatToOtherFormats",
			createResource: usernamePasswordDomainCredResourceWithoutDomain( // Start w/ username@domain and password.
				usernamePasswordDomainCredName,
				usernamePasswordDomainCredDesc,
				fmt.Sprintf("%s@%s", usernamePasswordDomainCredUsername, usernamePasswordDomainCredDomain),
				usernamePasswordDomainCredPassword,
			),
			planOnlyUpdates: []string{
				// Plan an update to domain\username.
				usernamePasswordDomainCredResourceWithoutDomain(
					usernamePasswordDomainCredName,
					usernamePasswordDomainCredDesc,
					fmt.Sprintf(`%s\\%s`, usernamePasswordDomainCredDomain, usernamePasswordDomainCredUsername),
					usernamePasswordDomainCredPassword,
				),
				// Plan an update to add the same domain in the domain field.
				usernamePasswordDomainCredResourceWithDomain(
					usernamePasswordDomainCredName,
					usernamePasswordDomainCredDesc,
					fmt.Sprintf("%s@%s", usernamePasswordDomainCredUsername, usernamePasswordDomainCredDomain),
					usernamePasswordDomainCredPassword,
					usernamePasswordDomainCredDomain,
				),
				// Plan an update to add the same username in the username field.
				usernamePasswordDomainCredResourceWithDomain(
					usernamePasswordDomainCredName,
					usernamePasswordDomainCredDesc,
					usernamePasswordDomainCredUsername,
					usernamePasswordDomainCredPassword,
					usernamePasswordDomainCredDomain,
				),
			},
		},
		{
			name: "domainSlashUsernameFormatToOtherFormats",
			createResource: usernamePasswordDomainCredResourceWithoutDomain( // Start w/ domain\username and password.
				usernamePasswordDomainCredName,
				usernamePasswordDomainCredDesc,
				fmt.Sprintf(`%s\\%s`, usernamePasswordDomainCredDomain, usernamePasswordDomainCredUsername),
				usernamePasswordDomainCredPassword,
			),
			planOnlyUpdates: []string{
				// Plan an update to username@domain.
				usernamePasswordDomainCredResourceWithoutDomain(
					usernamePasswordDomainCredName,
					usernamePasswordDomainCredDesc,
					fmt.Sprintf("%s@%s", usernamePasswordDomainCredUsername, usernamePasswordDomainCredDomain),
					usernamePasswordDomainCredPassword,
				),
				// Plan an update to add the same domain in the domain field.
				usernamePasswordDomainCredResourceWithDomain(
					usernamePasswordDomainCredName,
					usernamePasswordDomainCredDesc,
					fmt.Sprintf("%s@%s", usernamePasswordDomainCredUsername, usernamePasswordDomainCredDomain),
					usernamePasswordDomainCredPassword,
					usernamePasswordDomainCredDomain,
				),
				// Plan an update to add the same username in the username field.
				usernamePasswordDomainCredResourceWithDomain(
					usernamePasswordDomainCredName,
					usernamePasswordDomainCredDesc,
					usernamePasswordDomainCredUsername,
					usernamePasswordDomainCredPassword,
					usernamePasswordDomainCredDomain,
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *schema.Provider
			tc := resource.TestCase{
				ProviderFactories: providerFactories(&provider),
				CheckDestroy:      testAccCheckCredentialResourceDestroy(t, provider, usernamePasswordDomainCredentialType),
				Steps: []resource.TestStep{
					{
						// Create resource.
						Config: testConfig(url, fooOrg, firstProjectFoo, tt.createResource),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, NameKey, usernamePasswordDomainCredName),
							resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, DescriptionKey, usernamePasswordDomainCredDesc),
							resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainUsernameKey, usernamePasswordDomainCredUsername),
							resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey, usernamePasswordDomainCredPassword),
							resource.TestCheckResourceAttr(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainDomainKey, usernamePasswordDomainCredDomain),

							testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac(),
							testAccCheckCredentialResourceExists(provider, usernamePasswordDomainCredResc),
						),
					},
					importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
					{
						// Run a plan-only update on itself and verify no changes.
						PlanOnly: true,
						Config:   testConfig(url, fooOrg, firstProjectFoo, tt.createResource),
					},
					importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey),
				},
			}

			for _, resUpdate := range tt.planOnlyUpdates {
				tc.Steps = append(tc.Steps, resource.TestStep{
					PlanOnly: true,
					Config:   testConfig(url, fooOrg, firstProjectFoo, resUpdate),
				}, importStep(usernamePasswordDomainCredResc, credentialUsernamePasswordDomainPasswordKey))
			}

			resource.Test(t, tc)
		})
	}
}

func usernamePasswordDomainCredExternalUpdate(t *testing.T, testProvider *schema.Provider) {
	if storeId == "" {
		t.Fatal("storeId must be set before testing an external update")
	}

	md := testProvider.Meta().(*metaData)
	c := credentials.NewClient(md.client)
	cr, err := c.Read(context.Background(), storeId)
	if err != nil {
		t.Fatal(fmt.Errorf("got an error reading %q: %w", storeId, err))
	}

	// update credential options
	var opts []credentials.Option
	opts = append(opts, credentials.WithDescription("this is an updated description, my guy"))

	_, err = c.Update(context.Background(), cr.Item.Id, cr.Item.Version, opts...)
	if err != nil {
		t.Fatal(fmt.Errorf("got an error updating %q: %w", cr.Item.Id, err))
	}
}

func testAccCheckCredentialStoreUsernamePasswordDomainPasswordHmac() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[usernamePasswordDomainCredResc]
		if !ok {
			return fmt.Errorf("not found: %s", usernamePasswordDomainCredResc)
		}

		computed := rs.Primary.Attributes["password_hmac"]
		if len(computed) != 43 {
			return fmt.Errorf("Computed password hmac not the expected length of 43 characters. hmac: %q", computed)
		}

		return nil
	}
}
