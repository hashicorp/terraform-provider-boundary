package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/users"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	fooDescription       = "bar"
	fooDescriptionUpdate = "foo bar"
	fooDescriptionUnset  = ""
)

var (
	orgUser = fmt.Sprintf(`
resource "boundary_user" "org1" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`, fooDescription)

	orgUserUpdate = fmt.Sprintf(`
resource "boundary_user" "org1" {
	name        = "test"
	description = "%s"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`, fooDescriptionUpdate)

	orgUserWithAccts = `
resource "boundary_user" "org1" {
	name        = "test"
	description = "with accts"
	scope_id    = boundary_scope.org1.id
	account_ids = [
    boundary_account.foo.id
	]
	depends_on  = [boundary_role.org1_admin]
}`

	orgUserWithAcctsUpdate = `
resource "boundary_user" "org1" {
	name        = "test"
	description = "with accts"
	scope_id    = boundary_scope.org1.id
	depends_on  = [boundary_role.org1_admin]
}`
)

// NOTE: this test also tests out the direct token auth mechanism.

func TestAccUser(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]
	token := tc.Token().Token

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckUserResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfigWithToken(url, token, fooOrg, orgUser),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists(provider, "boundary_user.org1"),
					resource.TestCheckResourceAttr("boundary_user.org1", DescriptionKey, fooDescription),
					resource.TestCheckResourceAttr("boundary_user.org1", NameKey, "test"),
				),
			},
			importStep("boundary_user.org1"),
			{
				// test update description
				Config: testConfigWithToken(url, token, fooOrg, orgUserUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists(provider, "boundary_user.org1"),
					resource.TestCheckResourceAttr("boundary_user.org1", DescriptionKey, fooDescriptionUpdate),
					resource.TestCheckResourceAttr("boundary_user.org1", NameKey, "test"),
				),
			},
			importStep("boundary_user.org1"),
		},
	})
}

func TestAccUserWithAccounts(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]
	token := tc.Token().Token

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckUserResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfigWithToken(url, token, fooOrg, fooAccount, orgUserWithAccts),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists(provider, "boundary_user.org1"),
					testAccCheckAccountResourceExists(provider, "boundary_account.foo"),
					resource.TestCheckResourceAttr("boundary_user.org1", DescriptionKey, "with accts"),
					resource.TestCheckResourceAttr("boundary_user.org1", NameKey, "test"),
					testAccCheckUserResourceAccountsSet(provider, "boundary_user.org1", []string{"boundary_account.foo"}),
				),
			},
			importStep("boundary_user.org1"),
			importStep("boundary_account.foo", "password"),
			{
				// test update description
				Config: testConfigWithToken(url, token, fooOrg, fooAccount, orgUserWithAcctsUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists(provider, "boundary_user.org1"),
					testAccCheckAccountResourceExists(provider, "boundary_account.foo"),
					resource.TestCheckResourceAttr("boundary_user.org1", DescriptionKey, "with accts"),
					resource.TestCheckResourceAttr("boundary_user.org1", NameKey, "test"),
				),
			},
			importStep("boundary_user.org1"),
			importStep("boundary_account.foo", "password"),
		},
	})
}

func testAccCheckUserResourceAccountsSet(testProvider *schema.Provider, name string, accounts []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("user resource not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("user resource ID is not set")
		}

		// ensure accts are declared in state
		acctIDs := []string{}
		for _, acctResourceName := range acctIDs {
			ur, ok := s.RootModule().Resources[acctResourceName]
			if !ok {
				return fmt.Errorf("account resource not found: %s", acctResourceName)
			}

			acctID := ur.Primary.ID
			if id == "" {
				return fmt.Errorf("account resource ID not set")
			}

			acctIDs = append(acctIDs, acctID)
		}

		// check boundary to ensure it matches
		md := testProvider.Meta().(*metaData)
		usrClient := users.NewClient(md.client)

		u, err := usrClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading user %q: %v", id, err)
		}

		// for every account set on the user in the state, ensure
		// each group in boundary has the same setings
		if len(u.Item.AccountIds) == 0 {
			return fmt.Errorf("no account found on user")
		}

		for _, stateAccount := range acctIDs {
			ok := false
			for _, gotAccount := range u.Item.AccountIds {
				if gotAccount == stateAccount {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("account in state not set in boundary:\n  in state: %+v\n  in boundary: %+v", acctIDs, u.Item.AccountIds)
			}
		}

		return nil
	}
}

func testAccCheckUserResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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
		usrs := users.NewClient(md.client)

		_, err := usrs.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading user %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckUserResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_user":

				id := rs.Primary.ID
				usrs := users.NewClient(md.client)

				_, err := usrs.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Status != http.StatusNotFound {
					return fmt.Errorf("Didn't get a 404 when reading destroyed user %q: %v", id, err)
				}

			default:
				t.Logf("Got unknown resource type %q", rs.Type)
			}
		}
		return nil
	}
}
