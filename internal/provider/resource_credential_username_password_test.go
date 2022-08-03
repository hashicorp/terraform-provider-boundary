package provider

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentials"
	"github.com/hashicorp/boundary/testing/controller"
	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
	"github.com/hashicorp/go-kms-wrapping/v2/aead"
	"github.com/hashicorp/go-kms-wrapping/v2/extras/multi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"golang.org/x/crypto/hkdf"
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

					testAccCheckCredentialStoreUsernamePasswordHmac(provider, tc, usernamePasswordCredPassword),
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

					testAccCheckCredentialStoreUsernamePasswordHmac(provider, tc, usernamePasswordCredPassword+usernamePasswordCredUpdate),
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
				// TODO: Boundary currently causes an error on moving back to a previously
				// used token, for now verify that a plan only step had changes
				PreConfig:          func() { usernamePasswordCredExternalUpdate(t, provider) },
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Config:             testConfig(url, fooOrg, firstProjectFoo, resUpdate),
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

func testAccCheckCredentialStoreUsernamePasswordHmac(testProvider *schema.Provider, tc *controller.TestController, password string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		sRs, ok := s.RootModule().Resources[staticCredStoreResc]
		if !ok {
			return fmt.Errorf("not found: %s", staticCredStoreResc)
		}

		upRs, ok := s.RootModule().Resources[usernamePasswordCredResc]
		if !ok {
			return fmt.Errorf("not found: %s", usernamePasswordCredResc)
		}

		databaseWrapper, err := tc.Kms().GetWrapper(context.Background(), sRs.Primary.Attributes["scope_id"], 1)
		if err != nil {
			return err
		}
		hmac, err := HmacSha256(context.Background(), []byte(password), databaseWrapper, []byte(sRs.Primary.Attributes["id"]), nil)
		if err != nil {
			return err
		}
		computed := upRs.Primary.Attributes["password_hmac"]
		if hmac != computed {
			return fmt.Errorf("HMACs do not match. expected %q, got %q", hmac, computed)
		}

		return nil
	}
}

// HmacSha256 the provided data. Supports WithPrefix, WithEd25519 and WithPrk
// options. WithEd25519 is a "legacy" way to complete this operation and should
// not be used in new operations unless backward compatibility is needed. The
// WithPrefix option will prepend the prefix to the hmac-sha256 value.
func HmacSha256(ctx context.Context, data []byte, cipher wrapping.Wrapper, salt, info []byte) (string, error) {
	const op = "crypto.HmacSha256"
	var key [32]byte
	reader, err := NewDerivedReader(ctx, cipher, 32, salt, info)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	edKey, _, err := ed25519.GenerateKey(reader)
	if err != nil {
		return "", fmt.Errorf("%s: unable to generate derived key", op)
	}
	n := copy(key[:], edKey)
	if n != 32 {
		return "", fmt.Errorf("%s: expected to copy 32 bytes and got: %d", op, n)
	}

	mac := hmac.New(sha256.New, key[:])
	_, _ = mac.Write(data)
	hmac := mac.Sum(nil)

	return base64.RawURLEncoding.EncodeToString(hmac), nil
}

// DerivedReader returns a reader from which keys can be read, using the
// given wrapper, reader length limit, salt and context info. Salt and info can
// be nil.
//
// Example:
//	reader, _ := NewDerivedReader(wrapper, userId, jobId)
// 	key := ed25519.GenerateKey(reader)
func NewDerivedReader(ctx context.Context, wrapper wrapping.Wrapper, lenLimit int64, salt, info []byte) (*io.LimitedReader, error) {
	const op = "crypto.NewDerivedReader"
	if wrapper == nil {
		return nil, fmt.Errorf("%s: missing wrapper", op)
	}
	if lenLimit < 20 {
		return nil, fmt.Errorf("%s: lenLimit must be >= 20", op)
	}
	var aeadWrapper *aead.Wrapper
	switch w := wrapper.(type) {
	case *multi.PooledWrapper:
		raw := w.WrapperForKeyId("__base__")
		var ok bool
		if aeadWrapper, ok = raw.(*aead.Wrapper); !ok {
			return nil, fmt.Errorf("%s: unexpected wrapper type from multiwrapper base", op)
		}
	case *aead.Wrapper:
		aeadWrapper = w
	default:
		return nil, fmt.Errorf("%s: unknown wrapper type", op)
	}

	keyBytes, err := aeadWrapper.KeyBytes(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: error reading aead key bytes: %w", op, err)
	}
	if keyBytes == nil {
		return nil, fmt.Errorf("%s: aead wrapper missing bytes", op)
	}

	reader := hkdf.New(sha256.New, keyBytes, salt, info)
	return &io.LimitedReader{
		R: reader,
		N: lenLimit,
	}, nil
}
