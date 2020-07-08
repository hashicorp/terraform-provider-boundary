package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/watchtower/api/scopes"
	"github.com/hashicorp/watchtower/api/users"
)

const (
	userCreatedTimeKey = "created_time"
	userIDKey          = "id"
	userUpdatedTimeKey = "updated_time"
	userDisabledKey    = "disabled"
)

func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceWatchtowerUserRead,
		Schema: map[string]*schema.Schema{
			userNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			userIDKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			userDescriptionKey: {
				Type:     schema.TypeString,
				Computed: true,
			},
			userCreatedTimeKey: {
				Type:     schema.TypeString,
				Computed: true,
			},
			userUpdatedTimeKey: {
				Type:     schema.TypeString,
				Computed: true,
			},
			userDisabledKey: {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

// convertUserToDataSource creates a ResourceData type from a User
func convertUserToDataSource(u *users.User, d *schema.ResourceData) error {
	fmt.Printf("user '%+v'\n", u)
	if err := d.Set(userNameKey, u.Name); err != nil {
		return err
	}

	if err := d.Set(userDescriptionKey, u.Description); err != nil {
		return err
	}

	if err := d.Set(userCreatedTimeKey, u.CreatedTime.String()); err != nil {
		return err
	}

	if err := d.Set(userUpdatedTimeKey, u.UpdatedTime.String()); err != nil {
		return err
	}

	if err := d.Set(userDisabledKey, u.Disabled); err != nil {
		return err
	}

	if err := d.Set(userIDKey, u.Id); err != nil {
		return err
	}

	return nil
}

func dataSourceWatchtowerUserRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Organization{
		Client: client,
	}

	var (
		id   = d.Get(userIDKey)
		name = d.Get(userNameKey)
	)

	if id != "" && name != "" {
		return fmt.Errorf("'name' and 'id' are mutually exclusive, please pass one attribute only for user data source")
	}

	u := &users.User{}

	if id != "" {
		//	u := convertResourceDataToUser(d)
		u := &users.User{Id: id.(string)}

		u, apiErr, err := o.ReadUser(ctx, u)
		if err != nil {
			return fmt.Errorf("basic error reading user: %s", err.Error())
		}
		if apiErr != nil {
			return fmt.Errorf("API error reading user: %s", *apiErr.Message)
		}
	}

	if name != "" {
		users, apiErr, err := o.ListUsers(ctx)
		if err != nil {
			return err
		}
		if apiErr != nil {
			return fmt.Errorf("API err listing users: %s", *apiErr.Message)
		}

		for _, user := range users {
			if user.Name == name {
				u = user
			}
		}

	}

	return convertUserToDataSource(u, d)

}
