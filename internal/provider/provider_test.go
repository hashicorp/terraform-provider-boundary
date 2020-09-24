package provider

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	wrapping "github.com/hashicorp/go-kms-wrapping"
	"github.com/hashicorp/go-kms-wrapping/wrappers/aead"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testProvider *schema.Provider
var testProviders map[string]*schema.Provider

var (
	tcLoginName = "user"
	tcPassword  = "passpass"
	tcPAUM      = "ampw_0000000000"
	tcConfig    = []controller.Option{
		controller.WithDefaultAuthMethodId(tcPAUM),
		controller.WithDefaultLoginName(tcLoginName),
		controller.WithDefaultPassword(tcPassword),
	}
	tcRecoveryKey = "7xtkEoS5EXPbgynwd+dDLHopaCqK8cq0Rpep4eooaTs="
)

func init() {
	testProvider = New()
	testProviders = map[string]*schema.Provider{
		"boundary": testProvider,
	}
}

func testWrapper(t *testing.T, key string) wrapping.Wrapper {
	var keyBytes []byte
	switch key {
	case "":
		keyBytes = make([]byte, 32)
		n, err := rand.Read(keyBytes)
		if err != nil {
			t.Fatal(err)
		}
		if n != 32 {
			t.Fatal(n)
		}
		key = base64.StdEncoding.EncodeToString(keyBytes)
	default:
		var err error
		keyBytes, err = base64.StdEncoding.DecodeString(key)
		if err != nil {
			t.Fatal(err)
		}
	}
	wrapper := aead.NewWrapper(nil)

	_, err := wrapper.SetConfig(map[string]string{
		"key_id": key,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := wrapper.SetAESGCMKeyBytes(keyBytes); err != nil {
		t.Fatal(err)
	}
	return wrapper
}

func testConfig(url string, res ...string) string {
	provider := fmt.Sprintf(`
provider "boundary" {
	addr             = "%s"
	auth_method_id       = "%s"
	password_auth_method_login_name = "%s"
	password_auth_method_password = "%s"
}`, url, tcPAUM, tcLoginName, tcPassword)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func testConfigWithToken(url, token string, res ...string) string {
	provider := fmt.Sprintf(`
provider "boundary" {
	addr  = "%s"
	token = "%s"
}`, url, token)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func testConfigWithRecovery(url string, res ...string) string {
	provider := fmt.Sprintf(`
provider "boundary" {
	addr             = "%s"
	auth_method_id       = "%s"
	password_auth_method_login_name = "%s"
	password_auth_method_password = "%s"
	recovery_kms_hcl = <<DOC
	kms "aead" {
		purpose = ["recovery", "config"]
		aead_type = "aes-gcm"
		key = "7xtkEoS5EXPbgynwd+dDLHopaCqK8cq0Rpep4eooaTs="
		key_id = "global_recovery"
	}
	DOC
}`, url, tcPAUM, tcLoginName, tcPassword)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func TestProvider(t *testing.T) {
	if err := New().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
