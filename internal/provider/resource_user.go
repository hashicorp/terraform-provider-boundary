package provider

import (
	"context"

	"github.com/hashicorp/boundary/api/users"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUserCreate,
		ReadContext:   resourceUserRead,
		UpdateContext: resourceUserUpdate,
		DeleteContext: resourceUserDelete,
		Schema: map[string]*schema.Schema{
			NameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			DescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			ScopeIdKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []users.Option{}

	var name *string
	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		name = &nameStr
		opts = append(opts, users.WithName(nameStr))
	}

	var desc *string
	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		desc = &descStr
		opts = append(opts, users.WithDescription(descStr))
	}

	usrs := users.NewClient(md.client)

	u, apiErr, err := usrs.Create(
		ctx,
		scopeId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling create user: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating user: %s", apiErr.Message)
	}

	d.Set(NameKey, name)
	d.Set(DescriptionKey, desc)
	d.SetId(u.Id)

	return nil
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	usrs := users.NewClient(md.client)

	u, apiErr, err := usrs.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling read user: %v", err)
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

	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(ScopeIdKey, raw["scope_id"])

	return nil
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	usrs := users.NewClient(md.client)

	opts := []users.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, users.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, users.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, users.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
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
			return diag.Errorf("error calling update user: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating user: %s", apiErr.Message)
		}
	}

	if d.HasChange(NameKey) {
		d.Set(NameKey, name)
	}
	if d.HasChange(DescriptionKey) {
		d.Set(DescriptionKey, desc)
	}

	return nil
}

func resourceUserDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	usrs := users.NewClient(md.client)

	_, apiErr, err := usrs.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete user: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error deleting user: %s", apiErr.Message)
	}

	return nil
}
