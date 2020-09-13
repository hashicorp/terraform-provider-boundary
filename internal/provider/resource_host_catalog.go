package provider

import (
	"context"

	"github.com/hashicorp/boundary/api/hostcatalogs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	hostCatalogTypeKey    = "type"
	hostCatalogTypeStatic = "static"
)

func resourceHostCatalog() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceHostCatalogCreate,
		ReadContext:   resourceHostCatalogRead,
		UpdateContext: resourceHostCatalogUpdate,
		DeleteContext: resourceHostCatalogDelete,
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
			hostCatalogTypeKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}

}

func resourceHostCatalogCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	var typeStr string
	if typeVal, ok := d.GetOk(hostCatalogTypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}
	switch typeStr {
	case hostCatalogTypeStatic:
	default:
		return diag.Errorf("invalid type provided")
	}

	opts := []hostcatalogs.Option{}

	var name *string
	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		name = &nameStr
		opts = append(opts, hostcatalogs.WithName(nameStr))
	}

	var desc *string
	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		desc = &descStr
		opts = append(opts, hostcatalogs.WithDescription(descStr))
	}

	hcClient := hostcatalogs.NewClient(md.client)

	hc, apiErr, err := hcClient.Create(
		ctx,
		typeStr,
		scopeId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling create host catalog: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating host catalog: %s", apiErr.Message)
	}

	if name != nil {
		if err := d.Set(NameKey, name); err != nil {
			return diag.FromErr(err)
		}
	}

	if desc != nil {
		if err := d.Set(DescriptionKey, *desc); err != nil {
			return diag.FromErr(err)
		}
	}

	d.Set(hostCatalogTypeKey, hc.Type)
	d.SetId(hc.Id)

	return nil
}

func resourceHostCatalogRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hcClient := hostcatalogs.NewClient(md.client)

	hc, apiErr, err := hcClient.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling read host catalog: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading host catalog: %s", apiErr.Message)
	}
	if hc == nil {
		return diag.Errorf("host catalog nil after read")
	}

	raw := hc.LastResponseMap()
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
	d.Set(hostCatalogTypeKey, raw["type"])

	return nil
}

func resourceHostCatalogUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hcClient := hostcatalogs.NewClient(md.client)

	opts := []hostcatalogs.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, hostcatalogs.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, hostcatalogs.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, hostcatalogs.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, hostcatalogs.WithDescription(descStr))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, hostcatalogs.WithAutomaticVersioning(true))
		_, apiErr, err := hcClient.Update(
			ctx,
			d.Id(),
			0,
			opts...)
		if err != nil {
			return diag.Errorf("error calling update host catalog: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating host catalog: %s", apiErr.Message)
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

func resourceHostCatalogDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hcClient := hostcatalogs.NewClient(md.client)

	_, apiErr, err := hcClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete host catalog: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error deleting host catalog: %s", apiErr.Message)
	}

	return nil
}
