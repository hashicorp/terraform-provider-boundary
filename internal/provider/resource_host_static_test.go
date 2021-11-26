package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/hosts"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccHost(t *testing.T) {
	t.Run("non-static", func(t *testing.T) {
		t.Parallel()
		testAccHost(t, false)
	})
	t.Run("static", func(t *testing.T) {
		t.Parallel()
		testAccHost(t, true)
	})
}

func testAccHost(t *testing.T, static bool) {
	const (
		fooHostAddress       = "10.0.0.1"
		fooHostAddressUpdate = "10.10.0.0"
	)

	projCatalog := `
	resource "%s" "foo" {
		name        = "test"
		description = "test catalog"
		%s
		scope_id    = boundary_scope.proj1.id
		depends_on  = [boundary_role.proj1_admin]
	}`

	projHost := `
	resource "%s" "foo" {
		host_catalog_id = %s.foo.id
		name            = "test"
		description     = "test host"
		%s
		address         = "%s"
	}`

	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	//	org := iam.TestOrg(t, tc.IamRepo())
	url := tc.ApiAddrs()[0]

	catalogName := "boundary_host_catalog"
	hostName := "boundary_host"
	typeStr := `type = "static"`
	if static {
		catalogName = "boundary_host_catalog_static"
		hostName = "boundary_host_static"
		typeStr = ""
	}
	fooHostName := fmt.Sprintf("%s.foo", hostName)
	hcBlock := fmt.Sprintf(projCatalog, catalogName, typeStr)
	hostBlock := fmt.Sprintf(projHost, hostName, catalogName, typeStr, fooHostAddress)
	hostUpdateBlock := fmt.Sprintf(projHost, hostName, catalogName, typeStr, fooHostAddressUpdate)

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckHostResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// test project host create
				Config: testConfig(url, fooOrg, firstProjectFoo, hcBlock, hostBlock),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostAttributeSet(provider, fooHostName, "address", fooHostAddress),
					testAccCheckHostResourceExists(provider, fooHostName),
					resource.TestCheckResourceAttr(fooHostName, "name", "test"),
					resource.TestCheckResourceAttr(fooHostName, "description", "test host"),
					resource.TestCheckResourceAttr(fooHostName, "address", fooHostAddress),
				),
			},
			importStep(fooHostName),
			{
				// test project host update
				Config: testConfig(url, fooOrg, firstProjectFoo, hcBlock, hostUpdateBlock),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHostAttributeSet(provider, fooHostName, "address", fooHostAddressUpdate),
					testAccCheckHostResourceExists(provider, fooHostName),
					resource.TestCheckResourceAttr(fooHostName, "name", "test"),
					resource.TestCheckResourceAttr(fooHostName, "description", "test host"),
					resource.TestCheckResourceAttr(fooHostName, "address", fooHostAddressUpdate),
				),
			},
			importStep(fooHostName),
		},
	})
}

func testAccCheckHostResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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

		hostsClient := hosts.NewClient(md.client)

		if _, err := hostsClient.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error when reading host %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckHostAttributeSet(testProvider *schema.Provider, name, attrKey, wantAttrVal string) resource.TestCheckFunc {
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
		hostsClient := hosts.NewClient(md.client)

		h, err := hostsClient.Read(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Got an error when reading host %q: %v", id, err)
		}

		if len(h.Item.Attributes) == 0 {
			return errors.New("no host attributes found")
		}

		gotAttrVal, ok := h.Item.Attributes[attrKey]
		if !ok {
			return fmt.Errorf("attribute not found on host: '%s'", attrKey)
		}

		if gotAttrVal != wantAttrVal {
			return fmt.Errorf("got incorrect value for '%s': got '%s', want '%s'", attrKey, gotAttrVal, wantAttrVal)
		}

		return nil
	}
}

func testAccCheckHostResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_host":
				id := rs.Primary.ID

				hostsClient := hosts.NewClient(md.client)

				_, err := hostsClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed host %q: %v", id, apiErr)
				}

			default:
				continue
			}
		}
		return nil
	}
}
