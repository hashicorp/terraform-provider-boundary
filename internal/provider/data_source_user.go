package provider

import (
	"errors"
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
	fmt.Printf("[DEBUG] converting user to data source:\n'%+v'\n", u)

	if u.Name != nil {
		fmt.Printf("[DEBUG] setting user data source name attribute to '%s'\n", *u.Name)
		if err := d.Set(userNameKey, *u.Name); err != nil {
			return err
		}
	}

	if u.Description != nil {
		fmt.Printf("[DEBUG] setting user data source description attribute to '%s'\n", *u.Description)
		if err := d.Set(userDescriptionKey, *u.Description); err != nil {
			return err
		}

		fmt.Printf("[DEBUG] resource after description set:\n%+v\n", d)
		fmt.Printf("[DEBUG] description after set: %s\n", d.Get(userDescriptionKey))
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

	if id != "" {
		fmt.Printf("[DEBUG] searching for user by id: '%s'\n", id)
		user := &users.User{Id: id.(string)}

		user, apiErr, err := o.ReadUser(ctx, user)
		if err != nil {
			return fmt.Errorf("basic error reading user: %s", err.Error())
		}
		if apiErr != nil {
			return fmt.Errorf("API error reading user: %s", *apiErr.Message)
		}

		fmt.Printf("[DEBUG] found user by id:\n%+v\n", user)
		return convertUserToDataSource(user, d)
	}

	if name != "" {
		fmt.Printf("[DEBUG] searching for user by name: '%s'\n", name)
		users, apiErr, err := o.ListUsers(ctx)
		if err != nil {
			return err
		}
		if apiErr != nil {
			return fmt.Errorf("API err listing users: %s", *apiErr.Message)
		}

		if len(users) == 0 {
			return errors.New("list users returned no users")
		}

		for _, user := range users {
			if user.Name == name {
				fmt.Printf("[DEBUG] found user by name:\n%+v\n", user)
				return convertUserToDataSource(user, d)
			}
		}
		return fmt.Errorf("user '%s' not found in watchtower", name)
	}

	return errors.New("id or name parameter must be passed when using the users data source")
}
