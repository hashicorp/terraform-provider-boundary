package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/roles"
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
	wrapper := aead.NewWrapper(nil)
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		t.Fatal(err)
	}
	_, err = wrapper.SetConfig(map[string]string{
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
 	base_url             = "%s"
	auth_method_id       = "%s"
	password_auth_method_login_name = "%s"
	password_auth_method_password = "%s"
}`, url, tcPAUM, tcLoginName, tcPassword)

	c := []string{provider}
	c = append(c, res...)
	return strings.Join(c, "\n")
}

func testConfigWithRecovery(url string, res ...string) string {
	provider := fmt.Sprintf(`
provider "boundary" {
	base_url             = "%s"
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

func setGrantScopeIdOnProject(scopeId, grantScopeId, principalId string, client *api.Client) error {
	roleClient := roles.NewClient(client)
	_, err, _ := roleClient.Create(
		context.Background(),
		scopeId,
		roles.WithName(fmt.Sprintf("TestRole_%s", grantScopeId)),
		roles.WithDescription(fmt.Sprintf("Terraform test management role for %s", grantScopeId)),
		roles.WithGrantScopeId(grantScopeId))

	return fmt.Errorf("%s", err.Message)
}

func TestProvider(t *testing.T) {
	if err := New().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
