package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/watchtower/api/scopes"
)

func dataSourceWatchtowerUser() *schema.Resource {
	return &schema.Resource{
		Read:   dataSourceWatchtowerUserRead,
		Schema: resourceUser().Schema,
	}

}

func dataSourceWatchtowerUserRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Organization{
		Client: client,
	}

	u := convertResourceDataToUser(d)

	u, apiErr, err := o.ReadUser(ctx, u)
	if err != nil {
		return fmt.Errorf("error reading user: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error reading user: %s", *apiErr.Message)
	}

	return convertUserToResourceData(u, d)
}
