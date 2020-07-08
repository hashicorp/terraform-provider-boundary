package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/watchtower/testing/controller"
)

var (
	fooUserResourceName        = "foo"
	fooUserResourceDescription = "bar"

	fooUserResource = fmt.Sprintf(`
resource "watchtower_user" "foo" {
  name = "%s"
	description = "%s"
}`, fooUserResourceName, fooUserResourceDescription)

	fooUserDataSource = fmt.Sprintf(`
data "watchtower_user" "foo" {
  name = "%s"
}`, fooUserResourceName)

	fooUserDataSourceByID = `
data "watchtower_user" "foo" {
  id = watchtower_user.foo.id
}`
)

func TestAccDataSourceFooUser(t *testing.T) {
	tc := controller.NewTestController(t, controller.WithDefaultOrgId("o_0000000000"))
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	fooUserDataSourceName := "data.watchtower_user.foo"

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				// create the user resource
				Config: testConfig(url, fooUserResource),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists("watchtower_user.foo"),
					resource.TestCheckResourceAttr("watchtower_user", userNameKey, fooUserResourceName),
					resource.TestCheckResourceAttr("watchtower_user", userDescriptionKey, fooUserResourceDescription),
				),
			},
			{
				Config: testConfig(url, fooUserDataSource),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fooUserDataSourceName, userDescriptionKey, fooUserResourceDescription),
					resource.TestMatchResourceAttr(fooUserDataSourceName, userCreatedTimeKey, regexp.MustCompile("^20[0-9]{2}-")),
					resource.TestMatchResourceAttr(fooUserDataSourceName, userUpdatedTimeKey, regexp.MustCompile("^20[0-9]{2}-")),
					resource.TestCheckResourceAttr(fooUserDataSourceName, userNameKey, fooUserResourceName),
					resource.TestCheckResourceAttr(fooUserDataSourceName, userDisabledKey, "false"),
				),
			},
		},
	})
}

func TestAccDataSourceFooUserByID(t *testing.T) {
	tc := controller.NewTestController(t, controller.WithDefaultOrgId("o_0000000000"))
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	fooUserDataSourceName := "data.watchtower_user.foo"

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: testConfig(url, fooUser, fooUserDataSourceByID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserResourceExists("watchtower_user.foo"),
					//				resource.TestCheckResourceAttr(fooUserDataSourceName, userDescriptionKey, fooUserResourceDescription),
					resource.TestMatchResourceAttr(fooUserDataSourceName, userCreatedTimeKey, regexp.MustCompile("^20[0-9]{2}-")),
					resource.TestMatchResourceAttr(fooUserDataSourceName, userUpdatedTimeKey, regexp.MustCompile("^20[0-9]{2}-")),
					resource.TestCheckResourceAttr(fooUserDataSourceName, userNameKey, fooUserResourceName),
					resource.TestCheckResourceAttr(fooUserDataSourceName, userDisabledKey, "false"),
				),
			},
		},
	})
}
