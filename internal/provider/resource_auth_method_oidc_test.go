package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	fooAuthMethodOidcDesc       = "test auth method oidc"
	fooAuthMethodOidcDescUpdate = "test auth method oidc update"
)

var (
	fooAuthMethodOidc = fmt.Sprintf(`
resource "boundary_auth_method_oidc" "foo" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]

  issuer            = "https://test.com"
  client_id         = "foo_id"
  client_secret     = "foo_secret"
  max_age           = 10
  #api_url_prefix    = "http://"
  ca_certs   = [
<<EOT
-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----
EOT
  ]
	allowed_audiences = ["foo_aud"]
}`, fooAuthMethodOidcDesc)

	fooAuthMethodOidcUpdate = fmt.Sprintf(`
resource "boundary_auth_method_oidc" "foo" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]

  issuer            = "https://test-update.com"
  client_id         = "foo_id_update"
  client_secret     = "foo_secret_update"
  max_age           = 1
  #api_url_prefix    = "http://"
  ca_certs   = [
<<EOT
-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----
EOT
  ]
  allowed_audiences = ["foo_aud"]
}`, fooAuthMethodOidcDescUpdate)
)

func TestAccAuthMethodOidc(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAuthMethodOidcResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, fooOrg, fooAuthMethodOidc),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_oidc.foo", "description", fooAuthMethodOidcDesc),
					resource.TestCheckResourceAttr("boundary_auth_method_oidc.foo", "name", "test"),
					testAccCheckAuthMethodOidcResourceExists(provider, "boundary_auth_method_oidc.foo"),
				),
			},
			importStep("boundary_auth_method_oidc.foo", "client_secret"),
			{
				// update
				Config: testConfig(url, fooOrg, fooAuthMethodOidcUpdate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_auth_method_oidc.foo", "description", fooAuthMethodOidcDescUpdate),
					resource.TestCheckResourceAttr("boundary_auth_method_oidc.foo", "name", "test"),
					testAccCheckAuthMethodOidcResourceExists(provider, "boundary_auth_method_oidc.foo"),
				),
			},
			importStep("boundary_auth_method_oidc.foo", "client_secret"),
		},
	})
}

func testAccCheckAuthMethodOidcResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("No ID is set")
		}

		md := testProvider.Meta().(*metaData)

		amClient := authmethods.NewClient(md.client)

		if _, err := amClient.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error when reading auth method %q: %w", id, err)
		}

		return nil
	}
}

func testAccCheckAuthMethodOidcResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_auth_method_oidc":
				id := rs.Primary.ID

				amClient := authmethods.NewClient(md.client)

				_, err := amClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed auth method %q: %v", id, err)
				}

			default:
				continue
			}
		}
		return nil
	}
}
