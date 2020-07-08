package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/watchtower/testing/controller"
)

var fooUserDataSource = `
data "watchtower_user" "foo" {
  name = "test"
}`

func TestAccDataSourceUser(t *testing.T) {
	tc := controller.NewTestController(t, controller.WithDefaultOrgId("o_0000000000"))
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	fooUserDataSourceName := "data.watchtower_user.foo"

	resource.Test(t, resource.TestCase{
		Providers:    testProviders,
		CheckDestroy: testAccCheckUserResourceDestroy(t),
		Steps: []resource.TestStep{
			{
				// test create and read
				Config: testConfig(url, fooUser, fooUserDataSource),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists("watchtower_user.foo"),
					resource.TestCheckResourceAttr(fooUserDataSourceName, userDescriptionKey, fooUserDescription),
					resource.TestCheckResourceAttr(fooUserDataSourceName, userNameKey, "test"),
				),
			},
		},
	})
}
