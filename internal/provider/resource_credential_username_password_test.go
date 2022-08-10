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
	usernamePasswordCredResc     = "boundary_credential_username_password.example"
	usernamePasswordCredName     = "foo"
	usernamePasswordCredDesc     = "the foo"
	usernamePasswordCredUsername = "default_username"
	usernamePasswordCredPassword = "default_password"
	usernamePasswordCredUpdate   = "_random"
)

func usernamePasswordCredResource(name, description, username, password string) string {
	return fmt.Sprintf(`
resource "boundary_credential_store_static" "example" {
	name  = "static store name"
	description = "static store description"
	scope_id = boundary_scope.proj1.id
	depends_on = [boundary_role.proj1_admin]
}

resource "boundary_credential_username_password" "example" {
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

func TestAccCredentialUsernamePassword(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	res := usernamePasswordCredResource(
		usernamePasswordCredName,
		usernamePasswordCredDesc,
		usernamePasswordCredUsername,
		usernamePasswordCredPassword,
	)

	resUpdate := usernamePasswordCredResource(
		usernamePasswordCredName,
		usernamePasswordCredDesc,
		usernamePasswordCredUsername+usernamePasswordCredUpdate,
		usernamePasswordCredPassword+usernamePasswordCredUpdate,
	)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialUsernamePasswordResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordCredResc, NameKey, usernamePasswordCredName),
					resource.TestCheckResourceAttr(usernamePasswordCredResc, DescriptionKey, usernamePasswordCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordCredResc, credentialUsernamePasswordUsernameKey, usernamePasswordCredUsername),
					resource.TestCheckResourceAttr(usernamePasswordCredResc, credentialUsernamePasswordPasswordKey, usernamePasswordCredPassword),

					testAccCheckCredentialStoreUsernamePasswordHmac(provider),
					testAccCheckCredentialUsernamePasswordResourceExists(provider, usernamePasswordCredResc),
				),
			},
			importStep(usernamePasswordCredResc, credentialUsernamePasswordPasswordKey),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, resUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(usernamePasswordCredResc, NameKey, usernamePasswordCredName),
					resource.TestCheckResourceAttr(usernamePasswordCredResc, DescriptionKey, usernamePasswordCredDesc),
					resource.TestCheckResourceAttr(usernamePasswordCredResc, credentialUsernamePasswordUsernameKey, usernamePasswordCredUsername+usernamePasswordCredUpdate),
					resource.TestCheckResourceAttr(usernamePasswordCredResc, credentialUsernamePasswordPasswordKey, usernamePasswordCredPassword+usernamePasswordCredUpdate),

					testAccCheckCredentialStoreUsernamePasswordHmac(provider),
					testAccCheckCredentialUsernamePasswordResourceExists(provider, usernamePasswordCredResc),
				),
			},
			importStep(usernamePasswordCredResc, credentialUsernamePasswordPasswordKey),
			{
				// Run a plan only update and verify no changes
				PlanOnly: true,
				Config:   testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(usernamePasswordCredResc, credentialUsernamePasswordPasswordKey),
			{
				// update again but apply a preConfig to externally update resource
				PreConfig: func() { usernamePasswordCredExternalUpdate(t, provider) },
				Config:    testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(usernamePasswordCredResc, credentialUsernamePasswordPasswordKey),
		},
	})
}

func usernamePasswordCredExternalUpdate(t *testing.T, testProvider *schema.Provider) {
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

func testAccCheckCredentialUsernamePasswordResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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

func testAccCheckCredentialUsernamePasswordResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_credential_username_password":
				id := rs.Primary.ID

				c := credentials.NewClient(md.client)
				_, err := c.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed username password credential %q: %v", id, err)
				}
			default:
				continue
			}
		}
		return nil
	}
}

func testAccCheckCredentialStoreUsernamePasswordHmac(testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[usernamePasswordCredResc]
		if !ok {
			return fmt.Errorf("not found: %s", usernamePasswordCredResc)
		}

		computed := rs.Primary.Attributes["password_hmac"]
		if len(computed) != 43 {
			return fmt.Errorf("Computed password hmac not the expected length of 43 characters. hmac: %q", computed)
		}

		return nil
	}
}
