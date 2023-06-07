// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/hostsets"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-secure-stdlib/parseutil"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	hostSetTypePlugin = "plugin"
)

func resourceHostSetPlugin() *schema.Resource {
	return &schema.Resource{
		Description: "The host_set_plugin resource allows you to configure a Boundary host set. Host sets are " +
			"always part of a host catalog, so a host catalog resource should be used inline or you " +
			"should have the host catalog ID in hand to successfully configure a host set.",

		CreateContext: resourceHostSetPluginCreate,
		ReadContext:   resourceHostSetPluginRead,
		UpdateContext: resourceHostSetPluginUpdate,
		DeleteContext: resourceHostSetPluginDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the host set.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The host set name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The host set description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			TypeKey: {
				Description: "The type of host set",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     hostSetTypePlugin,
			},
			HostCatalogIdKey: {
				Description: "The catalog for the host set.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			PreferredEndpointsKey: {
				Description: "The ordered list of preferred endpoints.",
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			SyncIntervalSecondsKey: {
				Description: "The value to set for the sync interval seconds.",
				Type:        schema.TypeInt,
				Optional:    true,
				ValidateDiagFunc: func(in interface{}, _ cty.Path) diag.Diagnostics {
					val := in.(int)
					switch {
					case val >= -1:
						return nil
					default:
						return diag.Errorf("invalid value for sync_interval_seconds")
					}
				},
			},
			AttributesJsonKey: {
				Description: `The attributes for the host set. Either values encoded with the "jsonencode" function, pre-escaped JSON string, or a file:// or env:// path. Set to a string "null" or remove the block to clear all attributes in the host set.`,
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
		},
	}
}

func setFromHostSetPluginResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw[NameKey]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw[DescriptionKey]); err != nil {
		return err
	}
	if err := d.Set(HostCatalogIdKey, raw[HostCatalogIdKey]); err != nil {
		return err
	}
	if err := d.Set(TypeKey, raw[TypeKey]); err != nil {
		return err
	}
	if err := d.Set(SyncIntervalSecondsKey, raw[SyncIntervalSecondsKey]); err != nil {
		return err
	}
	if err := d.Set(PreferredEndpointsKey, raw[PreferredEndpointsKey]); err != nil {
		return err
	}
	// Attributes stuff
	{
		attrRaw, ok := raw["attributes"]
		switch ok {
		case true:
			if attrMap, ok := attrRaw.(map[string]interface{}); ok {
				// The data structure for AWS Host Set Plugin filter is different from the terraform input data structure
				// This causes diffs even if there as been no change to the filters
				// Flatten attribute filters data structure from Boundary SDK to match terraform input data structure
				// This sets the value which is used for `attributes_json` diff checker
				if filtersInt, ok := attrMap["filters"]; ok {
					if filters, ok := filtersInt.([]interface{}); ok {
						var flattenedFilters []string
						for _, f := range filters {
							if filter, ok := f.(string); ok {
								flattenedFilters = append(flattenedFilters, filter)
							}
						}
						attrMap["filters"] = strings.Join(flattenedFilters, ",")
					}
				}
				attrRaw = attrMap
			}

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
	d.SetId(raw[IDKey].(string))
	return nil
}

func resourceHostSetPluginCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var hostsetHostCatalogId string
	if hostsetHostCatalogIdVal, ok := d.GetOk(HostCatalogIdKey); ok {
		hostsetHostCatalogId = hostsetHostCatalogIdVal.(string)
	} else {
		return diag.Errorf("no host catalog ID provided")
	}

	opts := []hostsets.Option{}

	var typeStr string
	if typeVal, ok := d.GetOk(TypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}
	switch typeStr {
	// NOTE: When other types are added, ensure they don't accept hostSetIds if
	// it's not allowed
	case hostSetTypePlugin:
	default:
		return diag.Errorf("invalid type provided")
	}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, hostsets.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, hostsets.WithDescription(descStr))
	}

	syncIntervalSecondsVal, ok := d.GetOk(SyncIntervalSecondsKey)
	if ok {
		syncIntervalSecondsInt := syncIntervalSecondsVal.(int)
		opts = append(opts, hostsets.WithSyncIntervalSeconds(int32(syncIntervalSecondsInt)))
	}

	var preferredEndpoints []string
	if preferredEndpointsVal, ok := d.GetOk(PreferredEndpointsKey); ok {
		list := preferredEndpointsVal.([]interface{})
		preferredEndpoints = make([]string, 0, len(list))
		for _, i := range list {
			preferredEndpoints = append(preferredEndpoints, i.(string))
		}
		opts = append(opts, hostsets.WithPreferredEndpoints(preferredEndpoints))
	}

	attrsVal, ok := d.GetOk(AttributesJsonKey)
	if ok {
		attrsStr, err := parseutil.ParsePath(attrsVal.(string))
		if err != nil && !errors.Is(err, parseutil.ErrNotAUrl) {
			return diag.Errorf("error parsing path with attributes: %v", err)
		}
		switch attrsStr {
		case "null":
			opts = append(opts, hostsets.DefaultAttributes())
		default:
			// What comes in is json-encoded but we want to set a
			// map[string]interface{} so we unmarshal it and set that
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(attrsStr), &m); err != nil {
				return diag.Errorf("error unmarshaling attributes: %v", err)
			}
			opts = append(opts, hostsets.WithAttributes(m))
		}
	}

	hsClient := hostsets.NewClient(md.client)

	hscr, err := hsClient.Create(ctx, hostsetHostCatalogId, opts...)
	if err != nil {
		return diag.Errorf("error creating host set: %v", err)
	}
	if hscr == nil {
		return diag.Errorf("nil host set after create")
	}
	if err := setFromHostSetPluginResponseMap(d, hscr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceHostSetPluginRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hsClient := hostsets.NewClient(md.client)

	hsrr, err := hsClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading host set: %v", err)
	}
	if hsrr == nil {
		return diag.Errorf("host set nil after read")
	}

	if err := setFromHostSetPluginResponseMap(d, hsrr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceHostSetPluginUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hsClient := hostsets.NewClient(md.client)

	opts := []hostsets.Option{}

	if d.HasChange(NameKey) {
		opts = append(opts, hostsets.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			opts = append(opts, hostsets.WithName(nameStr))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, hostsets.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			opts = append(opts, hostsets.WithDescription(descStr))
		}
	}

	if d.HasChange(SyncIntervalSecondsKey) {
		opts = append(opts, hostsets.DefaultSyncIntervalSeconds())
		syncIntervalSecondsVal, ok := d.GetOk(SyncIntervalSecondsKey)
		if ok {
			syncIntervalSecondsInt := syncIntervalSecondsVal.(int)
			opts = append(opts, hostsets.WithSyncIntervalSeconds(int32(syncIntervalSecondsInt)))
		}
	}

	if d.HasChange(PreferredEndpointsKey) {
		opts = append(opts, hostsets.DefaultPreferredEndpoints())
		var preferredEndpoints []string
		if preferredEndpointsVal, ok := d.GetOk(PreferredEndpointsKey); ok {
			list := preferredEndpointsVal.([]interface{})
			preferredEndpoints = make([]string, 0, len(list))
			for _, i := range list {
				preferredEndpoints = append(preferredEndpoints, i.(string))
			}
			opts = append(opts, hostsets.WithPreferredEndpoints(preferredEndpoints))
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
				opts = append(opts, hostsets.DefaultAttributes())
			default:
				// What comes in is json-encoded but we want to set a
				// map[string]interface{} so we unmarshal it and set that
				var m map[string]interface{}
				if err := json.Unmarshal([]byte(attrsStr), &m); err != nil {
					return diag.Errorf("error unmarshaling attributes: %v", err)
				}
				opts = append(opts, hostsets.WithAttributes(m))
			}
		} else {
			opts = append(opts, hostsets.DefaultAttributes())
		}
	}

	if len(opts) > 0 {
		opts = append(opts, hostsets.WithAutomaticVersioning(true))
		hsrr, err := hsClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating host set: %v", err)
		}
		if hsrr == nil {
			return diag.Errorf("host set nil after update")
		}
		if err := setFromHostSetPluginResponseMap(d, hsrr.GetResponse().Map); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceHostSetPluginDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hsClient := hostsets.NewClient(md.client)

	_, err := hsClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting host set: %s", err.Error())
	}

	return nil
}
