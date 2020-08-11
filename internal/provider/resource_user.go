package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/boundary/api/users"
)

const (
	userNameKey        = "name"
	userDescriptionKey = "description"
)

func resourceUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceUserCreate,
		Read:   resourceUserRead,
		Update: resourceUserUpdate,
		Delete: resourceUserDelete,
		Schema: map[string]*schema.Schema{
			userNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			userDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}

}

// convertUserToResourceData creates a ResourceData type from a User
func convertUserToResourceData(u *users.User, d *schema.ResourceData) error {
	if u.Name != nil {
		if err := d.Set(userNameKey, u.Name); err != nil {
			return err
		}
	}

	if u.Description != nil {
		if err := d.Set(userDescriptionKey, u.Description); err != nil {
			return err
		}
	}

	d.SetId(u.Id)

	return nil
}

// convertResourceDataToUser returns a localy built User using the values provided in the ResourceData.
func convertResourceDataToUser(d *schema.ResourceData) *users.User {
	u := &users.User{}
	if descVal, ok := d.GetOk(userDescriptionKey); ok {
		desc := descVal.(string)
		u.Description = &desc
	}
	if nameVal, ok := d.GetOk(userNameKey); ok {
		name := nameVal.(string)
		u.Name = &name
	}

	if d.Id() != "" {
		u.Id = d.Id()
	}

	return u
}

func resourceUserCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Org{
		Client: client,
	}

	u := convertResourceDataToUser(d)

	u, apiErr, err := o.CreateUser(ctx, u)
	if err != nil {
		return fmt.Errorf("error calling new user: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error creating user: %s", apiErr.Message)
	}

	d.SetId(u.Id)

	return nil
}

func resourceUserRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Org{
		Client: client,
	}

	u := convertResourceDataToUser(d)

	u, apiErr, err := o.ReadUser(ctx, u)
	if err != nil {
		return fmt.Errorf("error reading user: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error reading user: %s", apiErr.Message)
	}

	return convertUserToResourceData(u, d)
}

func resourceUserUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Org{
		Client: client,
	}

	u := convertResourceDataToUser(d)

	if d.HasChange(userNameKey) {
		n := d.Get(userNameKey).(string)
		u.Name = &n
	}

	if d.HasChange(userDescriptionKey) {
		d := d.Get(userDescriptionKey).(string)
		u.Description = &d
	}

	u, apiErr, err := o.UpdateUser(ctx, u)
	if err != nil {
		return err
	}
	if apiErr != nil {
		return fmt.Errorf("error updating user: %s\n   Invalid request fields: %v\n", apiErr.Message, apiErr.Details.RequestFields)
	}

	return convertUserToResourceData(u, d)
}

func resourceUserDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Org{
		Client: client,
	}

	u := convertResourceDataToUser(d)

	_, apiErr, err := o.DeleteUser(ctx, u)
	if err != nil {
		return fmt.Errorf("error deleting user: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error deleting user: %s", apiErr.Message)
	}

	return nil
}
