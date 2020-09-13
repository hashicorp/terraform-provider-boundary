package provider

import (
	"context"

	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	scopeDescriptionKey = "description"
	scopeNameKey        = "name"
	scopeScopeIdKey     = "scope_id"
)

func resourceScope() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScopeCreate,
		ReadContext:   resourceScopeRead,
		UpdateContext: resourceScopeUpdate,
		DeleteContext: resourceScopeDelete,

		Schema: map[string]*schema.Schema{
			scopeNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			scopeDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			scopeScopeIdKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceScopeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	var scopeId string
	if scopeIdVal, ok := d.GetOk(scopeScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []scopes.Option{}

	var name *string
	nameVal, ok := d.GetOk(scopeNameKey)
	if ok {
		nameStr := nameVal.(string)
		name = &nameStr
		opts = append(opts, scopes.WithName(nameStr))
	}

	var desc *string
	descVal, ok := d.GetOk(scopeDescriptionKey)
	if ok {
		descStr := descVal.(string)
		desc = &descStr
		opts = append(opts, scopes.WithDescription(descStr))
	}

	scp := scopes.NewClient(client)

	p, apiErr, err := scp.Create(
		ctx,
		scopeId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling new scope: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating scope: %s", apiErr.Message)
	}

	if name != nil {
		if err := d.Set(scopeNameKey, name); err != nil {
			return diag.FromErr(err)
		}
	}

	if desc != nil {
		if err := d.Set(scopeDescriptionKey, *desc); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(p.Id)

	return nil
}

func resourceScopeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	scp := scopes.NewClient(client)

	s, apiErr, err := scp.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling read scope: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading scope: %s", apiErr.Message)
	}
	if s == nil {
		return diag.Errorf("scope nil after read")
	}

	raw := s.LastResponseMap()
	if raw == nil {
		return []diag.Diagnostic{
			{
				Severity: diag.Warning,
				Summary:  "response map empty after read",
			},
		}
	}

	d.Set(scopeNameKey, raw["name"])
	d.Set(scopeDescriptionKey, raw["description"])
	d.Set(scopeScopeIdKey, raw["scope_id"])

	return nil
}

func resourceScopeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	scp := scopes.NewClient(client)

	opts := []scopes.Option{}

	var name *string
	if d.HasChange(scopeNameKey) {
		opts = append(opts, scopes.DefaultName())
		nameVal, ok := d.GetOk(scopeNameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, scopes.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(scopeDescriptionKey) {
		opts = append(opts, scopes.DefaultDescription())
		descVal, ok := d.GetOk(scopeDescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, scopes.WithDescription(descStr))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, scopes.WithAutomaticVersioning(true))
		_, apiErr, err := scp.Update(
			ctx,
			d.Id(),
			0,
			opts...)
		if err != nil {
			return diag.Errorf("error calling update scope: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating scope: %s", apiErr.Message)
		}
	}

	if d.HasChange(scopeNameKey) {
		d.Set(scopeNameKey, name)
	}
	if d.HasChange(scopeDescriptionKey) {
		d.Set(scopeDescriptionKey, desc)
	}

	return nil
}

func resourceScopeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	scp := scopes.NewClient(client)

	_, apiErr, err := scp.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete scope: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error deleting scope: %s", apiErr.Message)
	}

	return nil
}
