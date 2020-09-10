package provider

import (
	"context"

	"github.com/hashicorp/boundary/api/users"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	userNameKey        = "name"
	userDescriptionKey = "description"
	userScopeIdKey     = "scope_id"
)

func resourceUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUserCreate,
		ReadContext:   resourceUserRead,
		UpdateContext: resourceUserUpdate,
		DeleteContext: resourceUserDelete,
		Schema: map[string]*schema.Schema{
			userNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			userDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			userScopeIdKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

// convertUserToResourceData creates a ResourceData type from a User
func convertUserToResourceData(u *users.User, d *schema.ResourceData) diag.Diagnostics {
	if u.Name != "" {
		if err := d.Set(userNameKey, u.Name); err != nil {
			return diag.FromErr(err)
		}
	}

	if u.Description != "" {
		if err := d.Set(userDescriptionKey, u.Description); err != nil {
			return diag.FromErr(err)
		}
	}

	if u.ScopeId != "" {
		if err := d.Set(userScopeIdKey, u.ScopeId); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(u.Id)

	return nil
}

// convertResourceDataToUser returns a localy built User using the values provided in the ResourceData.
func convertResourceDataToUser(d *schema.ResourceData) *users.User {
	u := new(users.User)

	if descVal, ok := d.GetOk(userDescriptionKey); ok {
		u.Description = descVal.(string)
	}

	if nameVal, ok := d.GetOk(userNameKey); ok {
		u.Name = nameVal.(string)
	}

	if scopeIdVal, ok := d.GetOk(userScopeIdKey); ok {
		u.ScopeId = scopeIdVal.(string)
	}

	if d.Id() != "" {
		u.Id = d.Id()
	}

	return u
}

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	u := convertResourceDataToUser(d)
	usrs := users.NewClient(client)

	u, apiErr, err := usrs.Create(
		ctx,
		u.ScopeId,
		users.WithName(u.Name),
		users.WithDescription(u.Description))
	if err != nil {
		return diag.Errorf("error calling new user: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating user: %s", apiErr.Message)
	}

	d.SetId(u.Id)

	return convertUserToResourceData(u, d)
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	u := convertResourceDataToUser(d)
	usrs := users.NewClient(client)

	u, apiErr, err := usrs.Read(ctx, u.Id)
	if err != nil {
		return diag.Errorf("error reading user: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading user: %s", apiErr.Message)
	}

	return convertUserToResourceData(u, d)
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	u := convertResourceDataToUser(d)
	usrs := users.NewClient(client)

	u, apiErr, err := usrs.Update(
		ctx,
		u.Id,
		0,
		users.WithAutomaticVersioning(true),
		users.WithName(u.Name),
		users.WithDescription(u.Description))
	if err != nil {
		return diag.FromErr(err)
	}
	if apiErr != nil {
		return diag.Errorf("error updating user: %s\n   Invalid request fields: %v\n", apiErr.Message, apiErr.Details.RequestFields)
	}

	return convertUserToResourceData(u, d)
}

func resourceUserDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	u := convertResourceDataToUser(d)
	usrs := users.NewClient(client)

	_, apiErr, err := usrs.Delete(ctx, u.Id)
	if err != nil {
		return diag.Errorf("error deleting user: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error deleting user: %s", apiErr.Message)
	}

	return nil
}
