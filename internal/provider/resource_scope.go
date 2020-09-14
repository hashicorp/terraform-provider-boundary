package provider

import (
	"context"

	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceScope() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScopeCreate,
		ReadContext:   resourceScopeRead,
		UpdateContext: resourceScopeUpdate,
		DeleteContext: resourceScopeDelete,

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

func resourceScopeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []scopes.Option{}

	var name *string
	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		name = &nameStr
		opts = append(opts, scopes.WithName(nameStr))
	}

	var desc *string
	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		desc = &descStr
		opts = append(opts, scopes.WithDescription(descStr))
	}

	scp := scopes.NewClient(md.client)

	p, apiErr, err := scp.Create(
		ctx,
		scopeId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling create scope: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating scope: %s", apiErr.Message)
	}

	d.Set(NameKey, name)
	d.Set(DescriptionKey, desc)
	d.SetId(p.Id)

	return nil
}

func resourceScopeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)

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

	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(ScopeIdKey, raw["scope_id"])

	return nil
}

func resourceScopeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)

	opts := []scopes.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, scopes.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, scopes.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, scopes.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
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

	if d.HasChange(NameKey) {
		d.Set(NameKey, name)
	}
	if d.HasChange(DescriptionKey) {
		d.Set(DescriptionKey, desc)
	}

	return nil
}

func resourceScopeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)

	_, apiErr, err := scp.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete scope: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error deleting scope: %s", apiErr.Message)
	}

	return nil
}
