// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

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
		DeprecationMessage: "Deprecated: use `boundary_host_catalog_static` instead.",
		Description:        "Deprecated: use `boundary_host_catalog_static` instead.",

		CreateContext: resourceHostCatalogStaticCreate(true),
		ReadContext:   resourceHostCatalogStaticRead(true),
		UpdateContext: resourceHostCatalogStaticUpdate(true),
		DeleteContext: resourceHostCatalogStaticDelete,
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
				Description: "The host catalog type. Only `static` is supported.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceHostCatalogStatic() *schema.Resource {
	return &schema.Resource{
		Description: "The static host catalog resource allows you to configure a Boundary static-type host catalog. Host " +
			"catalogs are always part of a project, so a project resource should be used inline or you " +
			"should have the project ID in hand to successfully configure a host catalog.",

		CreateContext: resourceHostCatalogStaticCreate(false),
		ReadContext:   resourceHostCatalogStaticRead(false),
		UpdateContext: resourceHostCatalogStaticUpdate(false),
		DeleteContext: resourceHostCatalogStaticDelete,
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
		},
	}
}

func setFromHostCatalogStaticResponseMap(d *schema.ResourceData, raw map[string]interface{}, hasTypeKey bool) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw["scope_id"]); err != nil {
		return err
	}
	if hasTypeKey {
		if err := d.Set(TypeKey, raw["type"]); err != nil {
			return err
		}
	}
	d.SetId(raw["id"].(string))
	return nil
}

func resourceHostCatalogStaticCreate(hasTypeKey bool) schema.CreateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		md := meta.(*metaData)

		var scopeId string
		if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
			scopeId = scopeIdVal.(string)
		} else {
			return diag.Errorf("no scope ID provided")
		}

		typeStr := hostCatalogTypeStatic
		if hasTypeKey {
			// Perform backwards-compat validation
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

		if err := setFromHostCatalogStaticResponseMap(d, hccr.GetResponse().Map, hasTypeKey); err != nil {
			return diag.FromErr(err)
		}

		return nil
	}
}

func resourceHostCatalogStaticRead(hasTypeKey bool) schema.ReadContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

		if err := setFromHostCatalogStaticResponseMap(d, hcrr.GetResponse().Map, hasTypeKey); err != nil {
			return diag.FromErr(err)
		}

		return nil
	}
}

func resourceHostCatalogStaticUpdate(hasTypeKey bool) schema.UpdateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

		if len(opts) > 0 {
			opts = append(opts, hostcatalogs.WithAutomaticVersioning(true))
			hcrr, err := hcClient.Update(ctx, d.Id(), 0, opts...)
			if err != nil {
				return diag.Errorf("error updating host catalog: %v", err)
			}
			if hcrr == nil {
				return diag.Errorf("host catalog nil after update")
			}
			if err := setFromHostCatalogStaticResponseMap(d, hcrr.GetResponse().Map, hasTypeKey); err != nil {
				return diag.FromErr(err)
			}
		}

		return nil
	}
}

func resourceHostCatalogStaticDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hcClient := hostcatalogs.NewClient(md.client)

	_, err := hcClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting host catalog: %v", err)
	}

	return nil
}
