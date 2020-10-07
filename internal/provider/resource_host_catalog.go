package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/hostcatalogs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	hostCatalogTypeStatic = "static"
)

func resourceHostCatalog() *schema.Resource {
	return &schema.Resource{
		Description: "The host catalog resource allows you to configure a Boundary host catalog. Host " +
			"catalogs are always part of a project, so a project resource should be used inline or you " +
			"should have the project ID in hand to successfully configure a host catalog.",

		CreateContext: resourceHostCatalogCreate,
		ReadContext:   resourceHostCatalogRead,
		UpdateContext: resourceHostCatalogUpdate,
		DeleteContext: resourceHostCatalogDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the host catalog.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The host catalog name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The host catalog description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID in which the resource is created.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			TypeKey: {
				Description: "The host catalog type. Only `Static` (yes, title case) is supported.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
		},
	}
}

func setFromHostCatalogResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(ScopeIdKey, raw["scope_id"])
	d.Set(TypeKey, raw["type"])
	d.SetId(raw["id"].(string))
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
	if typeVal, ok := d.GetOk(TypeKey); ok {
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

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, hostcatalogs.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, hostcatalogs.WithDescription(descStr))
	}

	hcClient := hostcatalogs.NewClient(md.client)

	hccr, err := hcClient.Create(ctx, typeStr, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating host catalog: %v", err)
	}
	if hccr == nil {
		return diag.Errorf("nil host catalog after create")
	}

	setFromHostCatalogResponseMap(d, hccr.GetResponseMap())

	return nil
}

func resourceHostCatalogRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hcClient := hostcatalogs.NewClient(md.client)

	hcrr, err := hcClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Status == int32(http.StatusNotFound) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading host catalog: %v", err)
	}
	if hcrr == nil {
		return diag.Errorf("host catalog nil after read")
	}

	setFromHostCatalogResponseMap(d, hcrr.GetResponseMap())

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
		_, err := hcClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating host catalog: %v", err)
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

	_, err := hcClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting host catalog: %v", err)
	}

	return nil
}
