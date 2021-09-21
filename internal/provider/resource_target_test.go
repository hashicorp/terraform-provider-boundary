package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/targets"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/boundary/testing/vault"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	fooTargetDescription       = "bar"
	fooTargetDescriptionUpdate = "foo bar"
)

var (
	fooBarHostSet = `
resource "boundary_host_catalog" "foo" {
	type        = "static"
	name        = "test"
	description = "test catalog"
	scope_id    = boundary_scope.proj1.id
	depends_on  = [boundary_role.proj1_admin]
}

resource "boundary_host" "foo" {
	name            = "foo"
	host_catalog_id = boundary_host_catalog.foo.id
	type            = "static"
	address         = "10.0.0.1"
}

resource "boundary_host" "bar" {
	name            = "bar"
	host_catalog_id = boundary_host_catalog.foo.id
	type            = "static"
	address         = "10.0.0.1"
}

resource "boundary_host_set" "foo" {
	name            = "foo"
	type            = "static"
	host_catalog_id = boundary_host_catalog.foo.id
	host_ids = [
		boundary_host.foo.id,
	]
}

resource "boundary_host_set" "bar" {
	name            = "bar"
	type            = "static"
	host_catalog_id = boundary_host_catalog.foo.id
	host_ids = [
		boundary_host.bar.id,
	]
}`

	fooBarCredLibs = `
resource "boundary_credential_library_vault" "foo" {
	name  = "foo"
	description = "foo library"
	credential_store_id = boundary_credential_store_vault.example.id
  	path = "foo/bar"
  	http_method = "GET"
}

resource "boundary_credential_library_vault" "bar" {
	name  = "bar"
	description = "bar library"
	credential_store_id = boundary_credential_store_vault.example.id
  	path = "bar/foo"
  	http_method = "GET"
}
`

	fooTarget = fmt.Sprintf(`
resource "boundary_target" "foo" {
	name         = "test"
	description  = "%s"
	type         = "tcp"
	scope_id     = boundary_scope.proj1.id
	host_source_ids = [
		boundary_host_set.foo.id
	]
	application_credential_source_ids = [
		boundary_credential_library_vault.foo.id
	]
	default_port = 22
	depends_on  = [boundary_role.proj1_admin]
	session_max_seconds = 6000
	session_connection_limit = 6
	worker_filter = "type == \"foo\""
}`, fooTargetDescription)

	fooTargetUpdate = fmt.Sprintf(`
resource "boundary_target" "foo" {
	name         = "test"
	description  = "%s"
	type         = "tcp"
	scope_id     = boundary_scope.proj1.id
	host_source_ids = [
		boundary_host_set.bar.id
	]
	application_credential_source_ids = [
		boundary_credential_library_vault.bar.id
	]
	default_port = 80
	depends_on  = [boundary_role.proj1_admin]
	session_max_seconds = 7000
	session_connection_limit = 7
	worker_filter = "type == \"bar\""
}`, fooTargetDescriptionUpdate)

	fooTargetUpdateUnsetHostAndCredSources = fmt.Sprintf(`
resource "boundary_target" "foo" {
	name         = "test"
	description  = "%s"
	type         = "tcp"
	scope_id     = boundary_scope.proj1.id
	default_port = 80
	depends_on  = [boundary_role.proj1_admin]
	session_max_seconds = 7000
	session_connection_limit = 7
	worker_filter = "type == \"bar\""
}`, fooTargetDescriptionUpdate)
)

func TestAccTarget(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)

	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	vc := vault.NewTestVaultServer(t)
	_, token := vc.CreateToken(t)
	credStoreRes := vaultCredStoreResource(vc,
		vaultCredStoreName,
		vaultCredStoreDesc,
		vaultCredStoreNamespace,
		"www.original.com",
		token,
		true)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckTargetResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, fooBarCredLibs, fooBarHostSet, fooTarget),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetResourceExists(provider, "boundary_target.foo"),
					resource.TestCheckResourceAttr("boundary_target.foo", DescriptionKey, fooTargetDescription),
					resource.TestCheckResourceAttr("boundary_target.foo", NameKey, "test"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetDefaultPortKey, "22"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetSessionMaxSecondsKey, "6000"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetSessionConnectionLimitKey, "6"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetWorkerFilterKey, `type == "foo"`),
					testAccCheckTargetResourceHostSource(provider, "boundary_target.foo", []string{"boundary_host_set.foo"}),
					testAccCheckTargetResourceAppCredSources(provider, "boundary_target.foo", []string{"boundary_credential_library_vault.foo"}),
				),
			},
			importStep("boundary_target.foo"),
			{
				// test update
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, fooBarCredLibs, fooBarHostSet, fooTargetUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetResourceExists(provider, "boundary_target.foo"),
					resource.TestCheckResourceAttr("boundary_target.foo", DescriptionKey, fooTargetDescriptionUpdate),
					resource.TestCheckResourceAttr("boundary_target.foo", targetDefaultPortKey, "80"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetSessionMaxSecondsKey, "7000"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetSessionConnectionLimitKey, "7"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetWorkerFilterKey, `type == "bar"`),
					testAccCheckTargetResourceHostSource(provider, "boundary_target.foo", []string{"boundary_host_set.bar"}),
					testAccCheckTargetResourceAppCredSources(provider, "boundary_target.foo", []string{"boundary_credential_library_vault.bar"}),
				),
			},
			importStep("boundary_target.foo"),
			{
				// test unset hosts and cred sources
				Config: testConfig(url, fooOrg, firstProjectFoo, credStoreRes, fooBarCredLibs, fooBarHostSet, fooTargetUpdateUnsetHostAndCredSources),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetResourceExists(provider, "boundary_target.foo"),
					resource.TestCheckResourceAttr("boundary_target.foo", DescriptionKey, fooTargetDescriptionUpdate),
					resource.TestCheckResourceAttr("boundary_target.foo", targetDefaultPortKey, "80"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetSessionMaxSecondsKey, "7000"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetSessionConnectionLimitKey, "7"),
					resource.TestCheckResourceAttr("boundary_target.foo", targetWorkerFilterKey, `type == "bar"`),
					testAccCheckTargetResourceHostSource(provider, "boundary_target.foo", nil),
					testAccCheckTargetResourceAppCredSources(provider, "boundary_target.foo", nil),
				),
			},
			importStep("boundary_target.foo"),
		},
	})
}

func testAccCheckTargetResourceHostSource(testProvider *schema.Provider, name string, hostSources []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("target resource not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("target resource ID is not set")
		}

		// ensure host sources are declared in state
		var hostSourceIDs []string
		for _, hostSourceResourceID := range hostSources {
			hs, ok := s.RootModule().Resources[hostSourceResourceID]
			if !ok {
				return fmt.Errorf("host source resource not found: %s", hostSourceResourceID)
			}

			hostSourceID := hs.Primary.ID
			if id == "" {
				return fmt.Errorf("host source resource ID not set")
			}

			hostSourceIDs = append(hostSourceIDs, hostSourceID)
		}

		// check boundary to ensure it matches
		md := testProvider.Meta().(*metaData)
		tgtsClient := targets.NewClient(md.client)

		t, err := tgtsClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading target %q: %v", id, err)
		}

		if len(t.Item.HostSourceIds) != len(hostSourceIDs) {
			return fmt.Errorf("tf state and boundary have different number of host sources")
		}

		for _, stateHostSourceId := range t.Item.HostSourceIds {
			ok := false
			for _, gotHostSourceID := range hostSourceIDs {
				if gotHostSourceID == stateHostSourceId {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("host source id in state not set in boundary: %s", stateHostSourceId)
			}
		}

		return nil
	}
}

func testAccCheckTargetResourceAppCredSources(testProvider *schema.Provider, name string, credSources []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("target resource not found: %s", name)
		}

		id := rs.Primary.ID
		if id == "" {
			return fmt.Errorf("target resource ID is not set")
		}

		// ensure cred sources are declared in state
		var credSourceIDs []string
		for _, credSourceResourceID := range credSources {
			cl, ok := s.RootModule().Resources[credSourceResourceID]
			if !ok {
				return fmt.Errorf("credential source resource not found: %s", credSourceResourceID)
			}

			credSourceID := cl.Primary.ID
			if id == "" {
				return fmt.Errorf("credential source resource ID not set")
			}

			credSourceIDs = append(credSourceIDs, credSourceID)
		}

		// check boundary to ensure it matches
		md := testProvider.Meta().(*metaData)
		tgtsClient := targets.NewClient(md.client)

		t, err := tgtsClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("got an error when reading target %q: %w", id, err)
		}

		if len(t.Item.ApplicationCredentialSourceIds) != len(credSourceIDs) {
			return fmt.Errorf("tf state and boundary have different number of application credential sources")
		}

		for _, stateCredSourceId := range t.Item.ApplicationCredentialSourceIds {
			ok := false
			for _, gotCredSourceID := range credSourceIDs {
				if gotCredSourceID == stateCredSourceId {
					ok = true
				}
			}
			if !ok {
				return fmt.Errorf("application credential source id in state not set in boundary: %s", stateCredSourceId)
			}
		}

		return nil
	}
}

func testAccCheckTargetResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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
		tgts := targets.NewClient(md.client)

		if _, err := tgts.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error when reading target %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckTargetResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}

		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_target":
				tgts := targets.NewClient(md.client)

				id := rs.Primary.ID

				_, err := tgts.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed target %q: %v", id, apiErr)
				}

			default:
				continue
			}
		}
		return nil
	}
}
