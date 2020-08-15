package provider

import (
	"fmt"

	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/boundary/api/users"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	userNameKey        = "name"
	userDescriptionKey = "description"
	userScopeIDKey     = "scope_id"
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
			userScopeIDKey: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
		},
	}
}

// convertUserToResourceData creates a ResourceData type from a User
func convertUserToResourceData(u *users.User, d *schema.ResourceData) error {
	if u.Name != "" {
		if err := d.Set(userNameKey, u.Name); err != nil {
			return err
		}
	}

	if u.Description != "" {
		if err := d.Set(userDescriptionKey, u.Description); err != nil {
			return err
		}
	}

	if u.Scope.Id != "" {
		if err := d.Set(userScopeIDKey, u.Scope.Id); err != nil {
			return err
		}
	}

	d.SetId(u.Id)

	return nil
}

// convertResourceDataToUser returns a localy built User using the values provided in the ResourceData.
func convertResourceDataToUser(d *schema.ResourceData) *users.User {
	u := &users.User{Scope: &scopes.ScopeInfo{}}

	if descVal, ok := d.GetOk(userDescriptionKey); ok {
		u.Description = descVal.(string)
	}

	if nameVal, ok := d.GetOk(userNameKey); ok {
		u.Name = nameVal.(string)
	}

	if scopeIDVal, ok := d.GetOk(userScopeIDKey); ok {
		u.Scope.Id = scopeIDVal.(string)
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

	u := convertResourceDataToUser(d)
	usrs := users.NewUsersClient(client)

	u, apiErr, err := usrs.Create(
		ctx,
		users.WithName(u.Name),
		users.WithDescription(u.Description),
		users.WithScopeId(u.Scope.Id))
	if err != nil {
		return fmt.Errorf("error calling new user: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error creating user: %s", apiErr.Message)
	}

	d.SetId(u.Id)

	return convertUserToResourceData(u, d)
}

func resourceUserRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	u := convertResourceDataToUser(d)
	usrs := users.NewUsersClient(client)

	u, apiErr, err := usrs.Read(ctx, u.Id, users.WithScopeId(u.Scope.Id))
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

	u := convertResourceDataToUser(d)
	usrs := users.NewUsersClient(client)

	if d.HasChange(userNameKey) {
		u.Name = d.Get(userNameKey).(string)
	}

	if d.HasChange(userDescriptionKey) {
		u.Description = d.Get(userDescriptionKey).(string)
	}

	u.Scope.Id = d.Get(userScopeIDKey).(string)

	u, apiErr, err := usrs.Update(
		ctx,
		u.Id,
		0,
		users.WithAutomaticVersioning(),
		users.WithName(u.Name),
		users.WithDescription(u.Description),
		users.WithScopeId(u.Scope.Id))
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

	u := convertResourceDataToUser(d)
	usrs := users.NewUsersClient(client)

	_, apiErr, err := usrs.Delete(ctx, u.Id, users.WithScopeId(u.Scope.Id))
	if err != nil {
		return fmt.Errorf("error deleting user: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error deleting user: %s", apiErr.Message)
	}

	return nil
}
