// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary-sdk/api"
	"github.com/hashicorp/boundary-sdk/api/apptokens"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

const (
	fooAppTokenName = "test-appt"
	fooAppTokenDesc = "test app token"
)

var fooAppToken = fmt.Sprintf(`
resource "boundary_app_token" "foo" {
    name                  = "%s"
    description           = "%s"
    scope_id              = boundary_scope.org1.id
    time_to_live_seconds  = 3600
    time_to_stale_seconds = 600
    permissions {
        label        = "read groups perm"
        grants       = ["ids=*;type=group;actions=list,read"]
        grant_scopes = ["this"]
    }
    depends_on = [boundary_role.org1_admin]
}`, fooAppTokenName, fooAppTokenDesc)

func TestAccAppToken(t *testing.T) {
	cfg, err := loadTestConfig()
	require.NoError(t, err)
	suffix := id.UniqueId()
	url := cfg.BoundaryAddr

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckAppTokenResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooOrg(suffix), fooAppToken),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("boundary_app_token.foo", "name", fooAppTokenName),
					resource.TestCheckResourceAttr("boundary_app_token.foo", "description", fooAppTokenDesc),
					resource.TestCheckResourceAttr("boundary_app_token.foo", "time_to_live_seconds", "3600"),
					resource.TestCheckResourceAttr("boundary_app_token.foo", "time_to_stale_seconds", "600"),
					resource.TestCheckResourceAttr("boundary_app_token.foo", "permissions.0.label", "read groups perm"),
					resource.TestCheckResourceAttr("boundary_app_token.foo", "permissions.0.grants.0", "ids=*;type=group;actions=list,read"),
					resource.TestCheckResourceAttr("boundary_app_token.foo", "permissions.0.grant_scopes.0", "this"),
					resource.TestCheckResourceAttr("boundary_app_token.foo", "permissions.0.deleted_scopes.#", "0"),
					resource.TestCheckResourceAttr("boundary_app_token.foo", "status", "active"),
					resource.TestCheckResourceAttrSet("boundary_app_token.foo", "expiration_time"),
					resource.TestCheckResourceAttrSet("boundary_app_token.foo", "created_by_user_id"),
					resource.TestCheckResourceAttrSet("boundary_app_token.foo", "id"),
					resource.TestCheckResourceAttrSet("boundary_app_token.foo", "token"),
					testAccCheckAppTokenResourceExists(provider, "boundary_app_token.foo"),
				),
			},
			// import step - ignore TTL because it's required for Create, but not returned on Read
			{
				ResourceName:            "boundary_app_token.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "time_to_live_seconds"},
				ImportStatePersist:      true, // Retains imported state for the next step
			},
			{
				Config:   testConfig(url, fooOrg(suffix), fooAppToken),
				PlanOnly: true, // Verifies plan against original config has no diff / no replacement
			},
		},
	})
}

func testAccCheckAppTokenResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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

		atClient := apptokens.NewClient(md.client)

		if _, err := atClient.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error when reading app token %q: %w", id, err)
		}

		return nil
	}
}

func testAccCheckAppTokenResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_app_token":
				id := rs.Primary.ID

				atClient := apptokens.NewClient(md.client)

				_, err := atClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed app token %q: %v", id, err)
				}

			default:
				continue
			}
		}
		return nil
	}
}
