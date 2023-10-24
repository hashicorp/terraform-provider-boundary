// Copyright (c) HashiCorp, Inc.
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

var accountPasswordRead = `
data "boundary_account" "acc_password" {
	depends_on = [ boundary_account_password.foo ]
	name = "test"
	auth_method_id = boundary_auth_method.foo.id
}`

var accountLdapRead = fmt.Sprintf(`
data "boundary_account" "acc_ldap" {
	depends_on = [ boundary_account_ldap.foo ]
	name = "%s"
	auth_method_id = boundary_auth_method_ldap.foo.id
}`, testAccountLdapName)

var accountOidcRead = `
data "boundary_account" "acc_oidc" {
	depends_on = [ boundary_account_oidc.foo ]
	name = "test"
	auth_method_id = boundary_auth_method_oidc.foo.id
}`

func TestAccAccountReadPassword(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg, fooAccountPassword, accountPasswordRead),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccountResourceExists(provider, "boundary_account_password.foo"),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_password", IDKey),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_password", AuthMethodIdKey),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_password", NameKey),
					resource.TestCheckResourceAttr("data.boundary_account.acc_password", DescriptionKey, fooAccountPasswordDesc),
				),
			},
		},
	})
}

func TestAccAccountReadLdap(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg, testAccountLdap, accountLdapRead),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccountResourceExists(provider, "boundary_account_ldap.foo"),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_ldap", IDKey),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_ldap", AuthMethodIdKey),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_ldap", NameKey),
					resource.TestCheckResourceAttr("data.boundary_account.acc_ldap", DescriptionKey, testAccountLdapDesc),
				),
			},
		},
	})
}

func TestAccAccountReadOidc(t *testing.T) {
	tp := oidc.StartTestProvider(t)
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	tpCert := strings.TrimSpace(tp.CACert())
	createConfig := fmt.Sprintf(fooAccountOidc, tp.Addr(), tpCert, fooAccountOidcDesc, tp.ExpectedSubject(), tp.Addr())

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg, createConfig, accountOidcRead),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccountResourceExists(provider, "boundary_account_oidc.foo"),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_oidc", IDKey),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_oidc", AuthMethodIdKey),
					resource.TestCheckResourceAttrSet("data.boundary_account.acc_oidc", NameKey),
					resource.TestCheckResourceAttr("data.boundary_account.acc_oidc", DescriptionKey, fooAccountOidcDesc),
				),
			},
		},
	})
}
