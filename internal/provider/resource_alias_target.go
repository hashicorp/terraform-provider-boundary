// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/aliases"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	aliasTypeTarget = "target"

	aliasTargetAuthorizeSessionHostIdKey = "authorize_session_host_id"
)

func resourceAliasTarget() *schema.Resource {
	return &schema.Resource{
		Description: "The target alias resource allows you to configure a Boundary target alias.",

		CreateContext: resourceTargetAliasCreate,
		ReadContext:   resourceTargetAliasRead,
		UpdateContext: resourceTargetAliasUpdate,
		DeleteContext: resourceTargetAliasDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the account.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The alias name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The alias description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			ValueKey: {
				Description: "The value of the alias.",
				Type:        schema.TypeString,
				Required:    true,
			},
			DestinationIdKey: {
				Description: "The destination of the alias.",
				Type:        schema.TypeString,
				Optional:    true,
			},

			// Target specific configurable parameters
			aliasTargetAuthorizeSessionHostIdKey: {
				Description: "The host id to pass to Boundary when performing an authorize session action.",
				Type:        schema.TypeString,
				Optional:    true,
			},

			TypeKey: {
				Description: "The type of alias; hardcoded.",
				Type:        schema.TypeString,
				Default:     aliasTypeTarget,
				Optional:    true,
			},
		},
	}
}

func setFromTargetAliasResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw["scope_id"]); err != nil {
		return err
	}
	if err := d.Set(ValueKey, raw["value"]); err != nil {
		return err
	}
	if err := d.Set(DestinationIdKey, raw["destination_id"]); err != nil {
		return err
	}
	if err := d.Set(TypeKey, raw["type"]); err != nil {
		return err
	}

	if attrsVal, ok := raw["attributes"]; ok {
		attrs := attrsVal.(map[string]interface{})
		tarAttrs, err := aliases.AttributesMapToTargetAliasAttributes(attrs)
		if err != nil {
			return err
		}
		if tarAttrs.AuthorizeSessionArguments != nil && tarAttrs.AuthorizeSessionArguments.HostId != "" {
			if err := d.Set(aliasTargetAuthorizeSessionHostIdKey, tarAttrs.AuthorizeSessionArguments.HostId); err != nil {
				return err
			}
		}
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceTargetAliasCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	opts := []aliases.Option{}

	if typeVal, ok := d.GetOk(ValueKey); ok {
		opts = append(opts, aliases.WithValue(typeVal.(string)))
	} else {
		return diag.Errorf("no alias value provided")
	}

	if nameVal, ok := d.GetOk(NameKey); ok {
		opts = append(opts, aliases.WithName(nameVal.(string)))
	}

	if descVal, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, aliases.WithDescription(descVal.(string)))
	}

	if destVal, ok := d.GetOk(DestinationIdKey); ok {
		opts = append(opts, aliases.WithDestinationId(destVal.(string)))
	}

	if hostIdVal, ok := d.GetOk(aliasTargetAuthorizeSessionHostIdKey); ok {
		opts = append(opts, aliases.WithTargetAliasAuthorizeSessionArgumentsHostId(hostIdVal.(string)))
	}

	aliasClient := aliases.NewClient(md.client)

	alcr, err := aliasClient.Create(ctx, typeStr, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating alias: %v", err)
	}
	if alcr == nil {
		return diag.Errorf("nil alias after create")
	}

	if err := setFromTargetAliasResponseMap(d, alcr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceTargetAliasRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	alClient := aliases.NewClient(md.client)

	alrr, err := alClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading auth method: %v", err)
	}
	if alrr == nil {
		return diag.Errorf("auth method nil after read")
	}

	if err := setFromTargetAliasResponseMap(d, alrr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceTargetAliasUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	alClient := aliases.NewClient(md.client)

	opts := []aliases.Option{}

	if d.HasChange(NameKey) {
		opts = append(opts, aliases.DefaultName())
		if nameVal, ok := d.GetOk(NameKey); ok {
			opts = append(opts, aliases.WithName(nameVal.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, aliases.DefaultDescription())
		if descVal, ok := d.GetOk(DescriptionKey); ok {
			opts = append(opts, aliases.WithDescription(descVal.(string)))
		}
	}

	if d.HasChange(ValueKey) {
		if valVal, ok := d.GetOk(ValueKey); ok {
			opts = append(opts, aliases.WithValue(valVal.(string)))
		} else {
			return diag.Errorf("no value provided")
		}
	}

	if d.HasChange(DestinationIdKey) {
		opts = append(opts, aliases.DefaultDestinationId())
		if destId, ok := d.GetOk(DestinationIdKey); ok {
			opts = append(opts, aliases.WithDestinationId(destId.(string)))
		}
	}

	if d.HasChange(aliasTargetAuthorizeSessionHostIdKey) {
		opts = append(opts, aliases.DefaultTargetAliasAuthorizeSessionArgumentsHostId())
		if authSessHostId, ok := d.GetOk(aliasTargetAuthorizeSessionHostIdKey); ok {
			opts = append(opts, aliases.WithTargetAliasAuthorizeSessionArgumentsHostId(authSessHostId.(string)))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, aliases.WithAutomaticVersioning(true))
		alur, err := alClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating auth method: %v", err)
		}

		if err := setFromTargetAliasResponseMap(d, alur.GetResponse().Map); err != nil {
			return diag.FromErr(err)
		}
		return nil
	}
	return nil
}

func resourceTargetAliasDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := aliases.NewClient(md.client)

	_, err := client.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting alias: %v", err)
	}

	return nil
}
