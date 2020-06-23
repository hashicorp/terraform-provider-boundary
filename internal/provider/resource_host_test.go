package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/watchtower/testing/controller"
)

const (
	fooHostDescription = "bar"
)

var (
	fooHost = fmt.Sprintf(`
resource "watchtower_host" "foo" {
	description = "%s"
	project_id = watchtower_host_catalog.foo.id 
	type = "Static"
}`, fooHostDescription)
)

func TestAccHostCreate(t *testing.T) {
	tc := controller.NewTestController(t, controller.WithDefaultOrgId("o_0000000000"))
	defer tc.Shutdown()
	url := tc.ApiAddrs()[0]

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				// test create
				Config: testConfig(url, firstProjectBar, fooHostCatalog, fooHost),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectResourceExists("watchtower_project.project1"),
					testAccCheckHostCatalogResourceExists("watchtower_host_catalog.foo"),
					// read not yet implemented
					//testAccCheckHostResourceExists("watchtower_host.foo"),
					resource.TestCheckResourceAttr("watchtower_host.foo", hostDescriptionKey, fooHostDescription),
				),
			},
		},
	})
}

//func testAccCheckHostResourceExists(name string) resource.TestCheckFunc {
//	return func(s *terraform.State) error {
//		rs, ok := s.RootModule().Resources[name]
//		if !ok {
//			return fmt.Errorf("Not found: %s", name)
//		}
//
//		id := rs.Primary.ID
//		if id == "" {
//			return fmt.Errorf("No ID is set")
//		}
//
//		md := testProvider.Meta().(*metaData)
//
//		h := hosts.Host{Id: id}
//
//		hc := &hosts.HostCatalog{
//			Client: md.client,
//			Id:     rs.Primary.Attributes["host_catalog_id"],
//		}
//
//		// not implemented
//		if _, _, err := hc.ReadHost(md.ctx, &h); err != nil {
//			return fmt.Errorf("Got an error when reading host catalog %q: %v", id, err)
//		}
//
//		return nil
//	}
//}
