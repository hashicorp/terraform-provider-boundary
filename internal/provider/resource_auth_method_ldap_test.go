// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/jimlambrt/gldap/testdirectory"
)

const (
	testAuthMethodLdapDesc = "test auth method ldap"

	testAuthMethodLdapDescUpdate = "test auth method ldap update"

	testAuthMethodLdapCert = `-----BEGIN CERTIFICATE-----
MIIDsjCCApoCCQCslgm7fAu/VzANBgkqhkiG9w0BAQsFADCBmjELMAkGA1UEBhMC
VVMxCzAJBgNVBAgMAldBMRMwEQYDVQQHDApCZWxsaW5naGFtMRIwEAYDVQQKDAlI
YXNoaUNvcnAxETAPBgNVBAsMCEJvdW5kYXJ5MRswGQYDVQQDDBJib3VuZGFyeXBy
b2plY3QuaW8xJTAjBgkqhkiG9w0BCQEWFmptYWxuaWNrQGhhc2hpY29ycC5jb20w
HhcNMjEwNDA2MjMzNTIxWhcNMjYwNDA1MjMzNTIxWjCBmjELMAkGA1UEBhMCVVMx
CzAJBgNVBAgMAldBMRMwEQYDVQQHDApCZWxsaW5naGFtMRIwEAYDVQQKDAlIYXNo
aUNvcnAxETAPBgNVBAsMCEJvdW5kYXJ5MRswGQYDVQQDDBJib3VuZGFyeXByb2pl
Y3QuaW8xJTAjBgkqhkiG9w0BCQEWFmptYWxuaWNrQGhhc2hpY29ycC5jb20wggEi
MA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDUa/ZLwYhTQ2hGzlwy/sB9xY9h
dM4qzY8DUF3Sin/+J2j59q3/vXZA7PS+o1GhoG1nW3J51zgyNFOY/EKHdtxVBken
YTTG+JswzrcTxsMV7/sYDgCLq6W8dPMV72gPH/3dRi3/0KUHtA6rBOf0Shf0f6Sz
7VmgTWcNmLvXpHKOs4YkjOL/tyflTgNm5j1dOa53TtwtMyvCcpGrB7PGL8m5+E2U
qxOzQ9kWfA6zr4Gl5rIm+Us8Ez3n1yGwjwFBteexk1Fot8zWKhoy7pZ3ZjWRpjwL
hfGs5eJs4kERQVAGONt39ZIR6OzOFxAsvI9WrMvxAsdCK63RtF2k4r0X21yDAgMB
AAEwDQYJKoZIhvcNAQELBQADggEBAJZcl7Zxjya23IcOV8jZDdCHtqnbcg9TcQb+
kpX1uEKJMFJoNmNK1q//nJxG1YBn3G8t9XtO6Kc6egdGHXWnOsM37N9hbYPJ2kW1
WWAwqWkQbV3wb0cc6MuU1S9xivOqwM046ZIcjrWR4T4tEUSUfYc3I+Yd8APdapn8
vePgWnmi/aSsx9RxVOUrzmVhzgN7rQJZGwnYYnxl4cwy2jxpysmXzg/grfXCZs/V
Kkc7Y5Ph6vRQ+vPCeB7QUxHxjlr8aq+rYDIaSiZ+/4+qyme0ergfvZmMSU8A3NNS
tYIMds5s2lIqVwOoyzpBEOjWBhUThH+aZu1A5c7Cb7s1eLSRX70=
-----END CERTIFICATE-----`
)

var (
	testAuthMethodLdap = `
resource "boundary_auth_method_ldap" "test-ldap" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]

	start_tls   	  = true
	discover_dn 	  = true
	anon_group_search = true
	upn_domain  	  = "upn-domain-test"
  	urls         	  = ["ldaps://ldap1", "ldaps://ldap2"]
	user_dn           = "user-dn"
	user_attr         = "user-attr"
	user_filter       = "user-filter"
	enable_groups     = true
	group_dn          = "group-dn"
	group_attr        = "group-attr"
	group_filter      = "group-filter"
	bind_dn           = "bind-dn"
	bind_password     = "bind-password"
	maximum_page_size = 10
	dereference_aliases = "DerefAlways"
	use_token_groups  = true
  	certificates 	  = [
<<EOT
%s
EOT
  ]
}`

	testPrimaryAuthMethodLdap = `
resource "boundary_auth_method_ldap" "test-ldap" {
	name        		 = "test"
	description 		 = "test auth method ldap"
	scope_id    		 = "global"
	is_primary_for_scope = true
  	urls         	  	 = ["ldap://%s:%d"]
	user_dn           	 = "%s"
	group_dn          	 = "%s"
	discover_dn 	  	 = true
	enable_groups    	 = true
	insecure_tls	 	 = true
}`

	testAuthMethodLdapUpdate = `
resource "boundary_auth_method_ldap" "test-ldap" {
	name                 = "test"
	description          = "%s"
	scope_id             = boundary_scope.org1.id
	is_primary_for_scope = true
	depends_on           = [boundary_role.org1_admin]

	start_tls            = false
	insecure_tls  		 = true
	discover_dn 		 = false
	anon_group_search 	 = false
	upn_domain  		 = "upn-domain-test-updated"
	urls                 = ["ldaps://ldap1-updated", "ldaps://ldap2-updated"]
	user_dn              = "user-dn-updated"
	user_attr            = "user-attr-updated"
	user_filter          = "user-filter-updated"
	enable_groups        = false
	group_dn             = "group-dn-updated"
	group_attr           = "group-attr-updated"
	group_filter         = "group-filter-updated"
	bind_dn              = "bind-dn-updated"
	bind_password        = "bind-password-updated" 
	use_token_groups     = false
	state                = "inactive"
	maximum_page_size = 100
	dereference_aliases = "NeverDerefAliases"
  	certificates         = [
<<EOT
%s
EOT
  ]
	}`
)

func TestAccAuthMethodLdap(t *testing.T) {
	td := testdirectory.Start(t,
		testdirectory.WithDefaults(t, &testdirectory.Defaults{AllowAnonymousBind: true}),
	)
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	tdCert := strings.TrimSpace(td.Cert())
	createConfig := fmt.Sprintf(testAuthMethodLdap, testAuthMethodLdapDesc, tdCert)
	updateConfig := fmt.Sprintf(testAuthMethodLdapUpdate, testAuthMethodLdapDescUpdate, testAuthMethodLdapCert)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAuthMethodResourceDestroy(t, provider, ldapAuthMethodType),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, createConfig),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", "description", testAuthMethodLdapDesc),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapStateField, "active-public"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapStartTlsField, "true"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapInsecureTlsField, "false"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapDiscoverDnField, "true"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapAnonGrpSearchField, "true"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapUpnDomainField, "upn-domain-test"),
					testAccCheckAuthMethodAttrAryValueSet(provider, "boundary_auth_method_ldap.test-ldap", authMethodLdapUrlsField, []string{"ldaps://ldap1", "ldaps://ldap2"}),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapUserDnField, "user-dn"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapUserAttrField, "user-attr"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapUserFilterField, "user-filter"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapEnableGrpsField, "true"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapGroupDnField, "group-dn"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapGroupAttrField, "group-attr"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapGroupFilterField, "group-filter"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapBindDnField, "bind-dn"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapBindPasswordField, "bind-password"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapUseTokenGrpsField, "true"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapMaxPageSizeField, "10"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapDerefAliasesField, "DerefAlways"),
					testAccCheckAuthMethodAttrAryValueSet(provider, "boundary_auth_method_ldap.test-ldap", authMethodLdapCertificatesField, []string{tdCert}),
				),
			},
			importStep("boundary_auth_method_ldap.test-ldap", "is_primary_for_scope", "bind_password", "client_certificate_key"),
			{
				// update
				Config: testConfig(url, fooOrg, updateConfig),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapStateField, "inactive"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", "description", testAuthMethodLdapDescUpdate),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", "name", "test"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapStartTlsField, "false"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapInsecureTlsField, "true"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapDiscoverDnField, "false"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapAnonGrpSearchField, "false"), resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapUpnDomainField, "upn-domain-test-updated"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapUpnDomainField, "upn-domain-test-updated"),
					testAccCheckAuthMethodAttrAryValueSet(provider, "boundary_auth_method_ldap.test-ldap", authMethodLdapUrlsField, []string{"ldaps://ldap1-updated", "ldaps://ldap2-updated"}),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapUserDnField, "user-dn-updated"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapUserAttrField, "user-attr-updated"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapUserFilterField, "user-filter-updated"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapEnableGrpsField, "false"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapGroupDnField, "group-dn-updated"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapGroupAttrField, "group-attr-updated"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapGroupFilterField, "group-filter-updated"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapBindDnField, "bind-dn-updated"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapBindPasswordField, "bind-password-updated"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapUseTokenGrpsField, "false"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapMaxPageSizeField, "100"),
					resource.TestCheckResourceAttr("boundary_auth_method_ldap.test-ldap", authMethodLdapDerefAliasesField, "NeverDerefAliases"),
					testAccCheckAuthMethodAttrAryValueSet(provider, "boundary_auth_method_ldap.test-ldap", authMethodLdapCertificatesField, []string{testAuthMethodLdapCert}),
					testAccCheckAuthMethodResourceExists(provider, "boundary_auth_method_ldap.test-ldap"),
					testAccIsPrimaryForScope(provider, "boundary_auth_method_ldap.test-ldap", true),
					testAccCheckAuthMethodResourceExists(provider, "boundary_auth_method_ldap.test-ldap"),
				),
			},
			importStep("boundary_auth_method_ldap.test-ldap", "is_primary_for_scope", "bind_password", "client_certificate_key"),
		},
	})
}
