package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/hostcatalogs"
	"github.com/hashicorp/go-secure-stdlib/parseutil"
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
				Computed:    true, // If name is provided this will be computed
			},
			PluginNameKey: {
				Description: "The name of the plugin that should back the resource. This or " + PluginIdKey + " must be defined.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
			},
			AttributesJsonKey: {
				Description: `The attributes for the host catalog. Either values encoded with the "jsonencode" function, pre-escaped JSON string, or a file:// or env:// path. Set to a string "null" or remove the block to clear all attributes in the host catalog.`,
				Type:        schema.TypeString,
				Optional:    true,
				// If set to null in config and nothing comes from API, consider
				// it the same. Same if config changes from empty to null.
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					switch {
					case old == new:
						return true
					case old == "null" && new == "":
						return true
					case old == "" && new == "null":
						return true
					default:
						return false
					}
				},
			},
			SecretsJsonKey: {
				Description: `The secrets for the host catalog. Either values encoded with the "jsonencode" function, pre-escaped JSON string, or a file:// or env:// path. Set to a string "null" to clear any existing values. NOTE: Unlike "attributes_json", removing this block will NOT clear secrets from the host catalog; this allows injecting secrets for one call, then removing them for storage.`,
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
			},
			SecretsHmacKey: {
				Description: "The HMAC'd secrets value returned from the server.",
				Type:        schema.TypeString,
				Optional:    true,
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
	// Plugin stuff
	{
		if err := d.Set(PluginIdKey, raw[PluginIdKey]); err != nil {
			return err
		}
		pluginRaw, ok := raw["plugin"]
		if !ok {
			return fmt.Errorf("plugin field not found in response")
		}
		pluginInfo, ok := pluginRaw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("plugin field in response has wrong type")
		}
		pluginNameRaw, ok := pluginInfo["name"]
		if !ok {
			return fmt.Errorf("plugin name field not found in response")
		}
		pluginName, ok := pluginNameRaw.(string)
		if !ok {
			return fmt.Errorf("plugin name field in response has wrong type")
		}
		if err := d.Set(PluginNameKey, pluginName); err != nil {
			return err
		}
	}
	// Attributes stuff
	{
		attrRaw, ok := raw["attributes"]
		switch ok {
		case true:
			encodedAttributes, err := json.Marshal(attrRaw)
			if err != nil {
				return err
			}
			if err := d.Set(AttributesJsonKey, string(encodedAttributes)); err != nil {
				return err
			}
		default:
			d.Set(AttributesJsonKey, nil)
		}
	}
	// Secrets stuff
	{
		// We do not save secrets into the state file, and they're not returned in
		// the response
		secretsRaw, ok := raw[SecretsHmacKey]
		switch ok {
		case true:
			if err := d.Set(SecretsHmacKey, secretsRaw); err != nil {
				return err
			}
		default:
			d.Set(SecretsHmacKey, nil)
		}
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

	attrsVal, ok := d.GetOk(AttributesJsonKey)
	if ok {
		attrsStr, err := parseutil.ParsePath(attrsVal.(string))
		if err != nil && !errors.Is(err, parseutil.ErrNotAUrl) {
			return diag.Errorf("error parsing path with attributes: %v", err)
		}
		switch attrsStr {
		case "null":
			opts = append(opts, hostcatalogs.DefaultAttributes())
		default:
			// What comes in is json-encoded but we want to set a
			// map[string]interface{} so we unmarshal it and set that
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(attrsStr), &m); err != nil {
				return diag.Errorf("error unmarshaling attributes: %v", err)
			}
			opts = append(opts, hostcatalogs.WithAttributes(m))
		}
	}

	secretsVal, ok := d.GetOk(SecretsJsonKey)
	if ok {
		secretsStr, err := parseutil.ParsePath(secretsVal.(string))
		if err != nil && !errors.Is(err, parseutil.ErrNotAUrl) {
			return diag.Errorf("error parsing path with secrets: %v", err)
		}
		switch secretsStr {
		case "null":
			opts = append(opts, hostcatalogs.DefaultSecrets())
		default:
			// What comes in is json-encoded but we want to set a
			// map[string]interface{} so we unmarshal it and set that
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(secretsStr), &m); err != nil {
				return diag.Errorf("error unmarshaling secrets: %v", err)
			}
			opts = append(opts, hostcatalogs.WithSecrets(m))
		}
	}

	hcClient := hostcatalogs.NewClient(md.client)

	hccr, err := hcClient.Create(ctx, hostCatalogTypePlugin, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating host catalog: %v", err)
	}
	if hccr == nil {
		return diag.Errorf("host catalog nil after create")
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

	if d.HasChange(NameKey) {
		opts = append(opts, hostcatalogs.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			opts = append(opts, hostcatalogs.WithName(nameStr))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, hostcatalogs.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			opts = append(opts, hostcatalogs.WithDescription(descStr))
		}
	}

	if d.HasChange(AttributesJsonKey) {
		attrsVal, ok := d.GetOk(AttributesJsonKey)
		if ok {
			attrsStr, err := parseutil.ParsePath(attrsVal.(string))
			if err != nil && !errors.Is(err, parseutil.ErrNotAUrl) {
				return diag.Errorf("error parsing path with attributes: %v", err)
			}
			switch attrsStr {
			case "null", "":
				opts = append(opts, hostcatalogs.DefaultAttributes())
			default:
				// What comes in is json-encoded but we want to set a
				// map[string]interface{} so we unmarshal it and set that
				var m map[string]interface{}
				if err := json.Unmarshal([]byte(attrsStr), &m); err != nil {
					return diag.Errorf("error unmarshaling attributes: %v", err)
				}
				opts = append(opts, hostcatalogs.WithAttributes(m))
			}
		} else {
			opts = append(opts, hostcatalogs.DefaultAttributes())
		}
	}

	// We don't save it in state so we can't compare; we can only look to see if
	// it's set. If it is, set whatever is there.
	secretsVal, ok := d.GetOk(SecretsJsonKey)
	if ok {
		secretsStr, err := parseutil.ParsePath(secretsVal.(string))
		if err != nil && !errors.Is(err, parseutil.ErrNotAUrl) {
			return diag.Errorf("error parsing path with secrets: %v", err)
		}
		switch secretsStr {
		case "null":
			opts = append(opts, hostcatalogs.DefaultSecrets())
		default:
			// What comes in is json-encoded but we want to set a
			// map[string]interface{} so we unmarshal it and set that
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(secretsStr), &m); err != nil {
				return diag.Errorf("error unmarshaling secrets: %v", err)
			}
			opts = append(opts, hostcatalogs.WithSecrets(m))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, hostcatalogs.WithAutomaticVersioning(true))
		hcrr, err := hcClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating host catalog: %v", err)
		}
		if hcrr == nil {
			return diag.Errorf("host catalog nil after update")
		}
		if err := setFromHostCatalogPluginResponseMap(d, hcrr.GetResponse().Map); err != nil {
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
