package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/workers"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	selfManagedWorkerName       = "self managed worker"
	selfManagedWorkerNameUpdate = "self managed worker update"
	selfManagedWorkerDesc       = "self managed worker description"
	selfManagedWorkerDescUpdate = "self managed worker description update"
	selfManagedWorkerToken      = "GzusqckarbczHoLGQ4UA25uSRGy7e7hUk4Qgz8T3NmSZF4oXBc1dKMRyH7Mr6W5u2QWggkZi1wsnfYMufb5LZJJnxEdywUxpE8RgAbVTahMhJm8oeH3nagrAKfWJkrTt8QuoqiupcTrFmJtDPZ5WBtqKSXiW3jdG8esCt7ZNKVaKV1Hq6MBXUf7duj9KfAy4Y3B31jDTzoF1uVnK1AsaLkEhgbHbuAH33L2KG5ivo7YeFeE6PknNoVavPSRSkoEpcXSfjvoPMAz9ttpNvH7jGWPwLti8r48NcVj41ftXWg"
)

var (
	workerLedCreate = fmt.Sprintf(`
resource "boundary_self_managed_worker" "worker_led" {
	scope_id = "global"
	name = "%s"
	description = "%s"
	worker_generated_auth_token = "%s"
}`, selfManagedWorkerName, selfManagedWorkerDesc, selfManagedWorkerToken)

	workerLedUpdate = fmt.Sprintf(`
resource "boundary_self_managed_worker" "worker_led" {
	scope_id = "global"
	name = "%s"
	description = "%s"
	worker_generated_auth_token = "%s"
}`, selfManagedWorkerNameUpdate, selfManagedWorkerDescUpdate, selfManagedWorkerToken)

	controllerLedCreate = fmt.Sprintf(`
resource "boundary_self_managed_worker" "controller_led" {
	scope_id = "global"
	name = "%s"
	description = "%s"
}`, selfManagedWorkerName, selfManagedWorkerDesc)
	controllerLedUpdate = fmt.Sprintf(`
resource "boundary_self_managed_worker" "controller_led" {
	scope_id = "global"
	name = "%s"
	description = "%s"
}`, selfManagedWorkerNameUpdate, selfManagedWorkerDescUpdate)
)

//// I'm unable to generate a worker token automatically, this will need to be run as a manual test.
// func TestSelfManagedWorkerWorkerLed(t *testing.T) {
// 	tc := controller.NewTestController(t, tcConfig...)
// 	defer tc.Shutdown()
// 	url := tc.ApiAddrs()[0]

// 	var provider *schema.Provider
// 	resource.Test(t, resource.TestCase{
// 		ProviderFactories: providerFactories(&provider),
// 		CheckDestroy:      testAccCheckSelfManagedWorkerResourceDestroy(t, provider),
// 		Steps: []resource.TestStep{
// 			{
// 				// create
// 				Config: testConfig(url, workerLedCreate),
// 				Check: resource.ComposeTestCheckFunc(
// 					testAccCheckSelfManagedWorkerResourceExists(provider, "boundary_self_managed_worker.worker_led"),
// 					resource.TestCheckResourceAttr("boundary_self_managed_worker.worker_led", "description", selfManagedWorkerDesc),
// 					resource.TestCheckResourceAttr("boundary_self_managed_worker.worker_led", "name", selfManagedWorkerName),
// 				),
// 			},
// 			importStep("boundary_self_managed_worker.worker_led"),
// 			{
// 				// update
// 				Config: testConfig(url, workerLedUpdate),
// 				Check: resource.ComposeTestCheckFunc(
// 					testAccCheckSelfManagedWorkerResourceExists(provider, "boundary_self_managed_worker.worker_led"),
// 					resource.TestCheckResourceAttr("boundary_self_managed_worker.worker_led", "description", selfManagedWorkerDescUpdate),
// 					resource.TestCheckResourceAttr("boundary_self_managed_worker.worker_led", "name", selfManagedWorkerNameUpdate),
// 				),
// 			},
// 			importStep("boundary_self_managed_worker.worker_led"),
// 		},
// 	})
// }

func TestSelfManagedWorkerControllerLed(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckSelfManagedWorkerResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, controllerLedCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSelfManagedWorkerResourceExists(provider, "boundary_self_managed_worker.controller_led"),
					resource.TestCheckResourceAttr("boundary_self_managed_worker.controller_led", "description", selfManagedWorkerDesc),
					resource.TestCheckResourceAttr("boundary_self_managed_worker.controller_led", "name", selfManagedWorkerName),
					resource.TestCheckResourceAttrSet("boundary_self_managed_worker.controller_led", "controller_generated_activation_token"),
				),
			},
			importStep("boundary_self_managed_worker.controller_led"),
			{
				// update
				Config: testConfig(url, controllerLedUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSelfManagedWorkerResourceExists(provider, "boundary_self_managed_worker.controller_led"),
					resource.TestCheckResourceAttr("boundary_self_managed_worker.controller_led", "description", selfManagedWorkerDescUpdate),
					resource.TestCheckResourceAttr("boundary_self_managed_worker.controller_led", "name", selfManagedWorkerNameUpdate),
				),
			},
			importStep("boundary_self_managed_worker.controller_led"),
		},
	})
}

func testAccCheckSelfManagedWorkerResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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

		wkrClient := workers.NewClient(md.client)

		if _, err := wkrClient.Read(context.Background(), id); err != nil {
			return fmt.Errorf("Got an error when reading worker %q: %v", id, err)
		}

		return nil
	}
}

func testAccCheckSelfManagedWorkerResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_self_managed_worker":
				id := rs.Primary.ID

				wkrClient := workers.NewClient(md.client)

				_, err := wkrClient.Read(context.Background(), id)
				if apiErr := api.AsServerError(err); apiErr == nil || apiErr.Response().StatusCode() != http.StatusNotFound {
					return fmt.Errorf("didn't get a 404 when reading destroyed worker %q: %v", id, err)
				}

			default:
				continue
			}
		}
		return nil
	}
}
