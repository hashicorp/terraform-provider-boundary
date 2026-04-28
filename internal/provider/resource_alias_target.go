// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/aliases"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	aliasTypeTarget = "target"

	aliasTargetAuthorizeSessionHostIdKey = "authorize_session_host_id"
	aliasTargetBaseValueKey              = "base_value"
)

func resourceAliasTarget() *schema.Resource {
	return &schema.Resource{
		Description: "The target alias resource allows you to configure a Boundary target alias.",

		CreateContext: resourceTargetAliasCreate,
		ReadContext:   resourceTargetAliasRead,
		UpdateContext: resourceTargetAliasUpdate,
		DeleteContext: resourceTargetAliasDelete,
		CustomizeDiff: resourceTargetAliasCustomizeDiff,
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
			aliasTargetBaseValueKey: {
				Description: "The base value of the alias returned by Boundary.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			ValueKey: {
				Description: "The value of the alias. Boundary may append a suffix; the returned value is stored in state.",
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

// setFromTargetAliasResponseMap updates d from the raw API response map.
// baseValue must be the user-configured value when it is known to
// have changed (Create, or Update when ValueKey changed), and empty string
// otherwise (Read, or Update when ValueKey is unchanged).  Passing empty
// string causes the function to preserve any existing base_value already in
// state, which prevents the server-appended suffix from corrupting the stored
// base.  On an initial import where no state exists the function falls back to
// the raw API value.
func setFromTargetAliasResponseMap(d *schema.ResourceData, raw map[string]interface{}, baseValue string) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw["scope_id"]); err != nil {
		return err
	}

	// Determine the correct base_value to store.  Priority order:
	//   1. baseValue – set on Create or when the user explicitly
	//      changed value (Update), so we know the un-suffixed base.
	//   2. Existing base_value in state – preserved on Read/refresh and on
	//      Update when the user did NOT change value, preventing the
	//      server-appended suffix from overwriting the correct base.
	//   3. Raw API value – fallback used during import when state is empty.
	switch {
	case baseValue != "":
		if err := d.Set(aliasTargetBaseValueKey, baseValue); err != nil {
			return err
		}
	case d.Get(aliasTargetBaseValueKey).(string) != "":
		// Keep the already-stored base value; no Set needed.
	default:
		if rawValue, ok := raw["value"]; ok && rawValue != nil {
			if err := d.Set(aliasTargetBaseValueKey, rawValue); err != nil {
				return err
			}
		}
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

func resourceTargetAliasCustomizeDiff(_ context.Context, rd *schema.ResourceDiff, _ interface{}) error {
	valueRaw, ok := rd.GetOk(ValueKey)
	if !ok || valueRaw.(string) == "" {
		return fmt.Errorf("value field is required")
	}

	if rd.Id() == "" {
		return nil
	}

	oldBaseRaw, _ := rd.GetChange(aliasTargetBaseValueKey)
	baseValue, _ := oldBaseRaw.(string)
	if baseValue == "" {
		return nil
	}

	oldValueRaw, newValueRaw := rd.GetChange(ValueKey)
	oldValue, _ := oldValueRaw.(string)
	newValue, _ := newValueRaw.(string)

	if newValue == baseValue {
		if err := rd.SetNew(ValueKey, oldValue); err != nil {
			return fmt.Errorf("failed to set value for diff suppression: %w", err)
		}
	}

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

	if valueVal, ok := d.GetOk(ValueKey); ok {
		opts = append(opts, aliases.WithValue(valueVal.(string)))
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

	// Capture the user-supplied value before the API call mutates state.
	var baseValue string
	if val, ok := d.GetOk(ValueKey); ok {
		baseValue = val.(string)
	}

	aliasClient := aliases.NewClient(md.client)

	alcr, err := aliasClient.Create(ctx, typeStr, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating alias: %v", err)
	}
	if alcr == nil {
		return diag.Errorf("nil alias after create")
	}

	if err := setFromTargetAliasResponseMap(d, alcr.GetResponse().Map, baseValue); err != nil {
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

	if err := setFromTargetAliasResponseMap(d, alrr.GetResponse().Map, ""); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceTargetAliasUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	alClient := aliases.NewClient(md.client)

	opts := []aliases.Option{}

	// Capture the new user-supplied value before building opts so we can pass
	// it as the baseValue when calling setFromTargetAliasResponseMap.
	var baseValue string
	if d.HasChange(ValueKey) {
		if val, ok := d.GetOk(ValueKey); ok {
			baseValue = val.(string)
		}
	}

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

		if err := setFromTargetAliasResponseMap(d, alur.GetResponse().Map, baseValue); err != nil {
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
