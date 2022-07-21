package provider

import (

	// "crypto/hmac"
	// "crypto/sha256"
	// "encoding/base64"
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentialstores"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	// "golang.org/x/crypto/blake2b"
)

const (
	staticCredStoreResc      = "boundary_credential_store_static.example"
	staticCredStoreName      = "foo"
	staticCredStoreDesc      = "the foo"
	staticCredStoreNamespace = "static"
	staticCredStoreUpdate    = "_random"
)

func staticCredStoreResource(name string, description string) string {
	// caCert := fmt.Sprintf("\"%s\"", strings.Replace(string(vc.CaCert), "\n", `\n`, -1))
	// clientCert := fmt.Sprintf("\"%s\"", strings.Replace(string(vc.ClientCert), "\n", `\n`, -1))
	// clientKey := fmt.Sprintf("\"%s\"", strings.Replace(string(vc.ClientKey), "\n", `\n`, -1))

	return fmt.Sprintf(`
resource "boundary_credential_store_static" "example" {
	name  = "%s"
	description = "%s"
	scope_id = boundary_scope.proj1.id
}`, name,
		description)
}

// func tokenHmac(token, accessor string) string {
// 	key := blake2b.Sum256([]byte(accessor))
// 	mac := hmac.New(sha256.New, key[:])
// 	_, _ = mac.Write([]byte(token))
// 	hmac := mac.Sum(nil)
// 	return base64.RawURLEncoding.EncodeToString(hmac)
// }

func TestAccCredentialStoreStatic(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	res := staticCredStoreResource(staticCredStoreName,
		staticCredStoreDesc)

	resUpdate := staticCredStoreResource(staticCredStoreName+staticCredStoreUpdate,
		staticCredStoreDesc+staticCredStoreUpdate)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckCredentialStoreStaticResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, firstProjectFoo, res),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(staticCredStoreResc, NameKey, staticCredStoreName),
					resource.TestCheckResourceAttr(staticCredStoreResc, DescriptionKey, staticCredStoreDesc),

					testAccCheckCredentialStoreVaultResourceExists(provider, staticCredStoreResc),
				),
			},
			importStep(staticCredStoreResc),
			{
				// update
				Config: testConfig(url, fooOrg, firstProjectFoo, resUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(staticCredStoreResc, NameKey, staticCredStoreName+staticCredStoreUpdate),
					resource.TestCheckResourceAttr(staticCredStoreResc, DescriptionKey, staticCredStoreDesc+staticCredStoreUpdate),

					testAccCheckCredentialStoreVaultResourceExists(provider, staticCredStoreResc),
				),
			},
			importStep(staticCredStoreResc),
			{
				// Run a plan only update and verify no changes
				PlanOnly: true,
				Config:   testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(staticCredStoreResc),
			{
				// update again but apply a preConfig to externally update resource
				// TODO: Boundary currently causes an error on moving back to a previously
				// used token, for now verify that a plan only step had changes
				PreConfig:          func() { externalUpdate(t, provider) },
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Config:             testConfig(url, fooOrg, firstProjectFoo, resUpdate),
			},
			importStep(staticCredStoreResc),
		},
	})
}

// var storeId string

// // externalUpdate uses the global storeId, therefore this function cannot be called until
// // a previous test step that calls testAccCheckCredentialStoreVaultResourceExists has completed.
// func externalUpdate(t *testing.T, testProvider *schema.Provider) {
// 	if storeId == "" {
// 		t.Fatal("storeId must be set before testing an external update")
// 	}
// 	vs := vault.NewTestVaultServer(t, vault.WithTestVaultTLS(vault.TestClientTLS))
// 	_, token := vs.CreateToken(t)

// 	md := testProvider.Meta().(*metaData)
// 	c := credentialstores.NewClient(md.client)
// 	cr, err := c.Read(context.Background(), storeId)
// 	if err != nil {
// 		t.Fatal(fmt.Errorf("got an error reading %q: %w", storeId, err))
// 	}

// 	// update Vault server to existing store
// 	var opts []credentialstores.Option
// 	opts = append(opts, credentialstores.WithVaultCredentialStoreToken(token))
// 	opts = append(opts, credentialstores.WithVaultCredentialStoreAddress(vs.Addr))
// 	opts = append(opts, credentialstores.WithVaultCredentialStoreCaCert(string(vs.CaCert)))
// 	opts = append(opts, credentialstores.WithVaultCredentialStoreClientCertificate(string(vs.ClientCert)))
// 	opts = append(opts, credentialstores.WithVaultCredentialStoreClientCertificateKey(string(vs.ClientKey)))

// 	_, err = c.Update(context.Background(), cr.Item.Id, cr.Item.Version, opts...)
// 	if err != nil {
// 		t.Fatal(fmt.Errorf("got an error updating %q: %w", cr.Item.Id, err))
// 	}
// }

// func testAccCheckCredentialStoreVaultResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
// 	return func(s *terraform.State) error {
// 		rs, ok := s.RootModule().Resources[name]
// 		if !ok {
// 			return fmt.Errorf("not found: %s", name)
// 		}

// 		id := rs.Primary.ID
// 		if id == "" {
// 			return fmt.Errorf("no ID is set")
// 		}
// 		storeId = id

// 		md := testProvider.Meta().(*metaData)
// 		c := credentialstores.NewClient(md.client)
// 		if _, err := c.Read(context.Background(), id); err != nil {
// 			return fmt.Errorf("got an error reading %q: %w", id, err)
// 		}

// 		return nil
// 	}
// }

func testAccCheckCredentialStoreStaticResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_credential_store_static":
				id := rs.Primary.ID

				c := credentialstores.NewClient(md.client)
				_, err := c.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed static credential store %q: %v", id, err)
				}
			default:
				continue
			}
		}
		return nil
	}
}
