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
	hostCatalogTypePlugin = "plugin"
)

func resourceHostCatalogPlugin() *schema.Resource {
	return &schema.Resource{
		Description: "The host catalog resource allows you to configure a Boundary plugin-type host catalog. Host " +
			"catalogs are always part of a project, so a project resource should be used inline or you " +
			"should have the project ID in hand to successfully configure a host catalog.",

		CreateContext: resourceHostCatalogPluginCreate,
		ReadContext:   resourceHostCatalogPluginRead,
		UpdateContext: resourceHostCatalogPluginUpdate,
		DeleteContext: resourceHostCatalogPluginDelete,
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
			PluginIdKey: {
				Description: "The ID of the plugin that should back the resource. This or " + PluginNameKey + " must be defined.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			PluginNameKey: {
				Description: "The name of the plugin that should back the resource. This or " + PluginIdKey + " must be defined.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			TypeKey: {
				Description: "The host catalog type. Only `plugin` is supported, and is the default.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Default:     "plugin",
			},
			AttributesKey: {
				Description: "The attributes for the host catalog.",
				Type:        schema.TypeMap,
				Optional:    true,
			},
			SecretsKey: {
				Description: "The secrets for the host catalog.",
				Type:        schema.TypeMap,
				Optional:    true,
			},
			SecretsHmacKey: {
				Description: "The HMAC'd secrets value returned from the server.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func setFromHostCatalogPluginResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw[NameKey]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw[DescriptionKey]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw[ScopeIdKey]); err != nil {
		return err
	}
	if err := d.Set(PluginIdKey, raw[PluginIdKey]); err != nil {
		return err
	}
	if err := d.Set(TypeKey, raw[TypeKey]); err != nil {
		return err
	}
	if err := d.Set(AttributesKey, raw[AttributesKey]); err != nil {
		return err
	}
	if err := d.Set(SecretsHmacKey, raw[SecretsHmacKey]); err != nil {
		return err
	}
	d.SetId(raw[IDKey].(string))
	return nil
}

func resourceHostCatalogPluginCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
	case hostCatalogTypePlugin:
	default:
		return diag.Errorf("invalid type provided")
	}

	opts := []hostcatalogs.Option{}

	var foundPluginId bool
	var foundPluginName bool
	if pluginIdVal, ok := d.GetOk(PluginIdKey); ok {
		pluginId := pluginIdVal.(string)
		opts = append(opts, hostcatalogs.WithPluginId(pluginId))
		foundPluginId = true
	}
	if pluginNameVal, ok := d.GetOk(PluginNameKey); ok {
		pluginName := pluginNameVal.(string)
		opts = append(opts, hostcatalogs.WithPluginName(pluginName))
		foundPluginName = true
	}
	if !foundPluginId && !foundPluginName {
		return diag.Errorf("neither plugin ID nor plugin name provided")
	}

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

	attrsVal, ok := d.GetOk(AttributesKey)
	if ok {
		attrs := attrsVal.(map[string]interface{})
		opts = append(opts, hostcatalogs.WithAttributes(attrs))
	}

	hcClient := hostcatalogs.NewClient(md.client)

	hccr, err := hcClient.Create(ctx, typeStr, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating host catalog: %v", err)
	}
	if hccr == nil {
		return diag.Errorf("nil host catalog after create")
	}

	if err := setFromHostCatalogPluginResponseMap(d, hccr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceHostCatalogPluginRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hcClient := hostcatalogs.NewClient(md.client)

	hcrr, err := hcClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading host catalog: %v", err)
	}
	if hcrr == nil {
		return diag.Errorf("host catalog nil after read")
	}

	if err := setFromHostCatalogPluginResponseMap(d, hcrr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceHostCatalogPluginUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		if err := d.Set(NameKey, name); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(DescriptionKey) {
		if err := d.Set(DescriptionKey, desc); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceHostCatalogPluginDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hcClient := hostcatalogs.NewClient(md.client)

	_, err := hcClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting host catalog: %v", err)
	}

	return nil
}
