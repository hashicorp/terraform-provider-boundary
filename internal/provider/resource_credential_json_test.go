// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentials"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	jsonCredResc       = "boundary_credential_json.example"
	jsonCredName       = "bar"
	jsonCredNameUpdate = "foo"
	jsonCredDesc       = "the bar"
	jsonCredDescUpdate = "the foo"
	jsonCredObj        = `jsonencode({
		password = "password",
		username = "admin"
	})`
	jsonCredObjUpdate = `jsonencode({
		password = "password",
		username = "db-admin"
	})`
)

func jsonCredResource(name, description, object string) string {
	return fmt.Sprintf(`
resource "boundary_credential_store_static" "example" {
	name  = "static store name"
	description = "static store description"
	scope_id = boundary_scope.proj1.id
	depends_on = [boundary_role.proj1_admin]
}

resource "boundary_credential_json" "example" {
	name  = "%s"
	description = "%s"
	credential_store_id = boundary_credential_store_static.example.id
	object = %s
}`, name, description, object)
}

func TestAccCredentialJson(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	res := jsonCredResource(
		jsonCredName,
		jsonCredDesc,
		jsonCredObj,
	)

	resUpdate := jsonCredResource(
		jsonCredNameUpdate,
		jsonCredDescUpdate,
		jsonCredObjUpdate,
	)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialResourceDestroy(t, provider, jsonCredentialType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(jsonCredResc, NameKey, jsonCredName),
					resource.TestCheckResourceAttr(jsonCredResc, DescriptionKey, jsonCredDesc),
					resource.TestCheckResourceAttr(jsonCredResc, credentialJsonObjectKey, `{"password":"password","username":"admin"}`),

					testAccCheckCredentialJsonObjectHmac(),
					testAccCheckCredentialResourceExists(provider, jsonCredResc),
				),
			},
			importStep(jsonCredResc, credentialJsonObjectKey),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, resUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(jsonCredResc, NameKey, jsonCredNameUpdate),
					resource.TestCheckResourceAttr(jsonCredResc, DescriptionKey, jsonCredDescUpdate),
					resource.TestCheckResourceAttr(jsonCredResc, credentialJsonObjectKey, `{"password":"password","username":"db-admin"}`),

					testAccCheckCredentialJsonObjectHmac(),
					testAccCheckCredentialResourceExists(provider, jsonCredResc),
				),
			},
			importStep(jsonCredResc, credentialJsonObjectKey),
			{
				// Run a plan only update and verify no changes
				PlanOnly: true,
				Config:   testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(jsonCredResc, credentialJsonObjectKey),
			{
				// update again but apply a preConfig to externally update resource
				PreConfig: func() { jsonCredExternalUpdate(t, provider) },
				Config:    testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(jsonCredResc, credentialJsonObjectKey),
		},
	})
}

func testAccCheckCredentialResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("no ID is set")
		}
		storeId = id

		md := testProvider.Meta().(*metaData)
		c := credentials.NewClient(md.client)
		if _, err := c.Read(context.Background(), id); err != nil {
			return fmt.Errorf("got an error reading %q: %w", id, err)
		}

		return nil
	}
}

type credentialType string

const (
	jsonCredentialType                   credentialType = "boundary_credential_json"
	sshPrivateKeyCredentialType          credentialType = "boundary_credential_ssh_private_key"
	usernamePasswordCredentialType       credentialType = "boundary_credential_username_password"
	usernamePasswordDomainCredentialType credentialType = "boundary_credential_username_password_domain"
)

func testAccCheckCredentialResourceDestroy(t *testing.T, testProvider *schema.Provider, typ credentialType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case string(typ):
				id := rs.Primary.ID

				c := credentials.NewClient(md.client)
				_, err := c.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed credential %q: %v", id, err)
				}
			default:
				continue
			}
		}
		return nil
	}
}

func testAccCheckCredentialJsonObjectHmac() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[jsonCredResc]
		if !ok {
			return fmt.Errorf("not found: %s", jsonCredResc)
		}

		computed := rs.Primary.Attributes["object_hmac"]
		if len(computed) != 43 {
			return fmt.Errorf("Computed password hmac not the expected length of 43 characters. hmac: %q", computed)
		}

		return nil
	}
}

func jsonCredExternalUpdate(t *testing.T, testProvider *schema.Provider) {
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
	opts = append(opts, credentials.WithDescription("this is an updated description"))

	_, err = c.Update(context.Background(), cr.Item.Id, cr.Item.Version, opts...)
	if err != nil {
		t.Fatal(fmt.Errorf("got an error updating %q: %w", cr.Item.Id, err))
	}
}
