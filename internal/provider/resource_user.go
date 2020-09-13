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

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	var scopeId string
	if scopeIdVal, ok := d.GetOk(userScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []users.Option{}

	var name *string
	nameVal, ok := d.GetOk(userNameKey)
	if ok {
		nameStr := nameVal.(string)
		name = &nameStr
		opts = append(opts, users.WithName(nameStr))
	}

	var desc *string
	descVal, ok := d.GetOk(userDescriptionKey)
	if ok {
		descStr := descVal.(string)
		desc = &descStr
		opts = append(opts, users.WithDescription(descStr))
	}

	usrs := users.NewClient(client)

	u, apiErr, err := usrs.Create(
		ctx,
		scopeId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling new user: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating user: %s", apiErr.Message)
	}

	if name != nil {
		if err := d.Set(userNameKey, name); err != nil {
			return diag.FromErr(err)
		}
	}

	if desc != nil {
		if err := d.Set(userDescriptionKey, *desc); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(u.Id)

	return nil
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	usrs := users.NewClient(client)

	u, apiErr, err := usrs.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error reading user: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading user: %s", apiErr.Message)
	}
	if u == nil {
		return diag.Errorf("user nil after read")
	}

	raw := u.LastResponseMap()
	if raw == nil {
		return []diag.Diagnostic{
			{
				Severity: diag.Warning,
				Summary:  "response map empty after read",
			},
		}
	}

	d.Set(userNameKey, raw["name"])
	d.Set(userDescriptionKey, raw["description"])
	d.Set(userScopeIdKey, raw["scope_id"])

	return nil
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	usrs := users.NewClient(client)

	opts := []users.Option{}

	var name *string
	if d.HasChange(userNameKey) {
		opts = append(opts, users.DefaultName())
		nameVal, ok := d.GetOk(userNameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, users.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(userDescriptionKey) {
		opts = append(opts, users.DefaultDescription())
		descVal, ok := d.GetOk(userDescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, users.WithDescription(descStr))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, users.WithAutomaticVersioning(true))
		_, apiErr, err := usrs.Update(
			ctx,
			d.Id(),
			0,
			opts...)
		if err != nil {
			return diag.Errorf("error updating user: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating user: %s", apiErr.Message)
		}
	}

	if d.HasChange(userNameKey) {
		d.Set(userNameKey, name)
	}
	if d.HasChange(userDescriptionKey) {
		d.Set(userDescriptionKey, desc)
	}

	return nil
}

func resourceUserDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	usrs := users.NewClient(client)

	_, apiErr, err := usrs.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete user: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error deleting user: %s", apiErr.Message)
	}

	return nil
}
