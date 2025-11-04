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
	passwordCredResc     = "boundary_credential_password.example"
	passwordCredName     = "foo"
	passwordCredDesc     = "the foo"
	passwordCredPassword = "default_password"
	passwordCredUpdate   = "_random"
)

func passwordCredResource(name, description, password string) string {
	return fmt.Sprintf(`
resource "boundary_credential_store_static" "example" {
    name  = "static store name"
    description = "static store description"
    scope_id = boundary_scope.proj1.id
    depends_on = [boundary_role.proj1_admin]
}

resource "boundary_credential_password" "example" {
    name  = "%s"
    description = "%s"
    credential_store_id = boundary_credential_store_static.example.id
    password = "%s"
}`, name,
		description,
		password)
}

func TestAccCredentialPassword(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	res := passwordCredResource(
		passwordCredName,
		passwordCredDesc,
		passwordCredPassword,
	)

	resUpdate := passwordCredResource(
		passwordCredName,
		passwordCredDesc,
		passwordCredPassword+passwordCredUpdate,
	)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialResourceDestroy(t, provider, passwordCredentialType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(passwordCredResc, NameKey, passwordCredName),
					resource.TestCheckResourceAttr(passwordCredResc, DescriptionKey, passwordCredDesc),
					resource.TestCheckResourceAttr(passwordCredResc, credentialPasswordKey, passwordCredPassword),

					testAccCheckCredentialPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, passwordCredResc),
				),
			},
			importStep(passwordCredResc, credentialPasswordKey),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, resUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(passwordCredResc, NameKey, passwordCredName),
					resource.TestCheckResourceAttr(passwordCredResc, DescriptionKey, passwordCredDesc),
					resource.TestCheckResourceAttr(passwordCredResc, credentialPasswordKey, passwordCredPassword+passwordCredUpdate),

					testAccCheckCredentialPasswordHmac(),
					testAccCheckCredentialResourceExists(provider, passwordCredResc),
				),
			},
			importStep(passwordCredResc, credentialPasswordKey),
			{
				// Run a plan only update and verify no changes
				PlanOnly: true,
				Config:   testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(passwordCredResc, credentialPasswordKey),
			{
				// update again but apply a preConfig to externally update resource
				PreConfig: func() { passwordCredExternalUpdate(t, provider) },
				Config:    testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(passwordCredResc, credentialPasswordKey),
		},
	})
}

func passwordCredExternalUpdate(t *testing.T, testProvider *schema.Provider) {
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

func testAccCheckCredentialPasswordHmac() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[passwordCredResc]
		if !ok {
			return fmt.Errorf("not found: %s", passwordCredResc)
		}

		computed := rs.Primary.Attributes["password_hmac"]
		if len(computed) != 43 {
			return fmt.Errorf("Computed password hmac not the expected length of 43 characters. hmac: %q", computed)
		}

		return nil
	}
}
