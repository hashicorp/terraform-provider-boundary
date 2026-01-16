// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/workers"
	"github.com/hashicorp/boundary/testing/controller"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	workerName       = "self managed worker"
	workerNameUpdate = "self managed worker update"
	workerDesc       = "self managed worker description"
	workerDescUpdate = "self managed worker description update"
	workerToken      = "GzusqckarbczHoLGQ4UA25uSRGy7e7hUk4Qgz8T3NmSZF4oXBc1dKMRyH7Mr6W5u2QWggkZi1wsnfYMufb5LZJJnxEdywUxpE8RgAbVTahMhJm8oeH3nagrAKfWJkrTt8QuoqiupcTrFmJtDPZ5WBtqKSXiW3jdG8esCt7ZNKVaKV1Hq6MBXUf7duj9KfAy4Y3B31jDTzoF1uVnK1AsaLkEhgbHbuAH33L2KG5ivo7YeFeE6PknNoVavPSRSkoEpcXSfjvoPMAz9ttpNvH7jGWPwLti8r48NcVj41ftXWg"
)

var (
	workerLedCreate = fmt.Sprintf(`
	resource "boundary_worker" "worker_led" {
		scope_id = "global"
		name = "%s"
		description = "%s"
		worker_generated_auth_token = "%s"
	}`, workerName, workerDesc, workerToken)

	workerLedUpdate = fmt.Sprintf(`
	resource "boundary_worker" "worker_led" {
		scope_id = "global"
		name = "%s"
		description = "%s"
		worker_generated_auth_token = "%s"
	}`, workerNameUpdate, workerDescUpdate, workerToken)

	controllerLedCreate = fmt.Sprintf(`
resource "boundary_worker" "controller_led" {
	scope_id = "global"
	name = "%s"
	description = "%s"
}`, workerName, workerDesc)
	controllerLedUpdate = fmt.Sprintf(`
resource "boundary_worker" "controller_led" {
	scope_id = "global"
	name = "%s"
	description = "%s"
}`, workerNameUpdate, workerDescUpdate)
)

func TestWorkerWorkerLed(t *testing.T) {
	token := os.Getenv("BOUNDARY_TF_PROVIDER_TEST_WORKER_LED_TOKEN")
	if token == "" {
		t.Skip("Not running worker led activation test without worker led token present")
	}

	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckworkerResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, workerLedCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckworkerResourceExists(provider, "boundary_worker.worker_led"),
					resource.TestCheckResourceAttr("boundary_worker.worker_led", "description", workerDesc),
					resource.TestCheckResourceAttr("boundary_worker.worker_led", "name", workerName),
					resource.TestCheckResourceAttr("boundary_worker.worker_led", "worker_generated_auth_token", workerToken),
				),
			},
			{
				// create
				Config:   testConfig(url, workerLedCreate),
				PlanOnly: true,
			},
			importStep("boundary_worker.worker_led"),
			{
				// update
				Config: testConfig(url, workerLedUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckworkerResourceExists(provider, "boundary_worker.worker_led"),
					resource.TestCheckResourceAttr("boundary_worker.worker_led", "description", workerDescUpdate),
					resource.TestCheckResourceAttr("boundary_worker.worker_led", "name", workerNameUpdate),
					resource.TestCheckResourceAttr("boundary_worker.worker_led", "worker_generated_auth_token", workerToken),
				),
			},
			importStep("boundary_worker.worker_led"),
		},
	})
}

func TestWorkerControllerLed(t *testing.T) {
	tc := controller.NewTestController(t, tcConfig...)
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	var provider *schema.Provider
	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories(&provider),
		CheckDestroy:      testAccCheckworkerResourceDestroy(t, provider),
		Steps: []resource.TestStep{
			{
				// create
				Config: testConfig(url, controllerLedCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckworkerResourceExists(provider, "boundary_worker.controller_led"),
					resource.TestCheckResourceAttr("boundary_worker.controller_led", "description", workerDesc),
					resource.TestCheckResourceAttr("boundary_worker.controller_led", "name", workerName),
					resource.TestCheckResourceAttrSet("boundary_worker.controller_led", "controller_generated_activation_token"),
				),
			},
			importStep("boundary_worker.controller_led", authorizedActions),
			{
				// update
				Config: testConfig(url, controllerLedUpdate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckworkerResourceExists(provider, "boundary_worker.controller_led"),
					resource.TestCheckResourceAttr("boundary_worker.controller_led", "description", workerDescUpdate),
					resource.TestCheckResourceAttr("boundary_worker.controller_led", "name", workerNameUpdate),
				),
			},
			importStep("boundary_worker.controller_led", authorizedActions),
		},
	})
}

func testAccCheckworkerResourceExists(testProvider *schema.Provider, name string) resource.TestCheckFunc {
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

func testAccCheckworkerResourceDestroy(t *testing.T, testProvider *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if testProvider.Meta() == nil {
			t.Fatal("got nil provider metadata")
		}
		md := testProvider.Meta().(*metaData)

		for _, rs := range s.RootModule().Resources {
			switch rs.Type {
			case "boundary_worker":
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
