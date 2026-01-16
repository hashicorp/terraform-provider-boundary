// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/cap/oidc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	fooAccountOidcDesc       = "test account oidc"
	fooAccountOidcDescUpdate = "test account oidc update"
)

var fooAccountOidc = `
resource "boundary_auth_method_oidc" "foo" {
	name        = "test"
	description = "test account oidc auth method"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]

	issuer            = "%s"
	client_id         = "foo_id"
	client_secret     = "foo_secret"
	max_age           = 0
	api_url_prefix    = "http://localhost:9200"
	idp_ca_certs   = [
<<EOT
%s
EOT
	]
	allowed_audiences = ["foo_aud"]
	signing_algorithms = ["ES256"]
	account_claim_maps = ["oid=sub"]
	claims_scopes = ["profile"]
}

resource "boundary_account_oidc" "foo" {
	name           = "test"
	description    = "%s"
	subject		   = "%s"
	issuer		   = "%s"
	auth_method_id = boundary_auth_method_oidc.foo.id
}`

func TestAccOidcAccount(t *testing.T) {
	tp := oidc.StartTestProvider(t)
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider

	tpCert := strings.TrimSpace(tp.CACert())
	createConfig := fmt.Sprintf(fooAccountOidc, tp.Addr(), tpCert, fooAccountOidcDesc, tp.ExpectedSubject(), tp.Addr())
	updateConfig := fmt.Sprintf(fooAccountOidc, tp.Addr(), tpCert, fooAccountOidcDescUpdate, tp.ExpectedSubject(), tp.Addr())

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAccountResourceDestroy(t, provider, oidcAccountType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, createConfig),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_account_oidc.foo", "description", fooAccountOidcDesc),
					resource.TestCheckResourceAttr("boundary_account_oidc.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_account_oidc.foo", "issuer", tp.Addr()),
					resource.TestCheckResourceAttr("boundary_account_oidc.foo", "subject", tp.ExpectedSubject()),
					testAccCheckAccountResourceExists(provider, "boundary_account_oidc.foo"),
				),
			},
			importStep("boundary_account_oidc.foo", "oidc"),
			{
				// update
				Config: testConfig(url, fooOrg, updateConfig),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_account_oidc.foo", "description", fooAccountOidcDescUpdate),
					resource.TestCheckResourceAttr("boundary_account_oidc.foo", "name", "test"),
					resource.TestCheckResourceAttr("boundary_account_oidc.foo", "issuer", tp.Addr()),
					resource.TestCheckResourceAttr("boundary_account_oidc.foo", "subject", tp.ExpectedSubject()),
					testAccCheckAccountResourceExists(provider, "boundary_account_oidc.foo"),
				),
			},
			importStep("boundary_account_oidc.foo", "oidc"),
		},
	})
}
