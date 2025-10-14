// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceAuthMethod() *schema.Resource {
	return &schema.Resource{
		Description: "The auth method resource allows you to configure a Boundary auth_method.",

		CreateContext: resourceAuthMethodCreate,
		ReadContext:   resourceAuthMethodRead,
		UpdateContext: resourceAuthMethodUpdate,
		DeleteContext: resourceAuthMethodDelete,
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
				Description: "The auth method name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The auth method description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			TypeKey: {
				Description: "The resource type.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				ValidateFunc: validation.StringInSlice([]string{
					authmethodTypeOidc,
					authmethodTypePassword,
				}, false),
			},
			authmethodMinLoginNameLengthKey: {
				Description: "The minimum login name length.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Deprecated:  "Will be removed in favor of using attributes parameter",
			},
			authmethodMinPasswordLengthKey: {
				Description: "The minimum password length.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Deprecated:  "Will be removed in favor of using attributes parameter",
			},
			authmethodIsPrimaryAuthMethodForScopeKey: {
				Description: "When true, makes this auth method the primary auth method for the scope in which it resides.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
		},
	}
}

func setFromAuthMethodResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw["scope_id"]); err != nil {
		return err
	}
	if err := d.Set(TypeKey, raw["type"]); err != nil {
		return err
	}

	if p, ok := raw[authmethodIsPrimaryAuthMethodForScopeKey]; ok {
		d.Set(authmethodIsPrimaryAuthMethodForScopeKey, p.(bool))
	}

	switch raw["type"].(string) {
	case authmethodTypePassword:
		if attrsVal, ok := raw["attributes"]; ok {
			attrs := attrsVal.(map[string]interface{})

			minLoginNameLength := attrs["min_login_name_length"].(json.Number)
			minLoginNameLengthInt, _ := minLoginNameLength.Int64()
			if err := d.Set(authmethodMinLoginNameLengthKey, int(minLoginNameLengthInt)); err != nil {
				return err
			}

			minPasswordLength := attrs["min_password_length"].(json.Number)
			minPasswordLengthInt, _ := minPasswordLength.Int64()
			if err := d.Set(authmethodMinPasswordLengthKey, int(minPasswordLengthInt)); err != nil {
				return err
			}
		}
	}
	d.SetId(raw["id"].(string))
	return nil
}

func resourceAuthMethodCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	opts := []authmethods.Option{}

	if nameVal, ok := d.GetOk(NameKey); ok {
		opts = append(opts, authmethods.WithName(nameVal.(string)))
	}

	if descVal, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, authmethods.WithDescription(descVal.(string)))
	}

	// TODO(malnick) - deprecate
	if minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey); ok {
		opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(minLengthVal.(int))))
	}

	// TODO(malnick) - deprecate
	if minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey); ok {
		opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(minLengthVal.(int))))
	}

	amClient := authmethods.NewClient(md.client)

	amcr, err := amClient.Create(ctx, typeStr, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating auth method: %v", err)
	}
	if amcr == nil {
		return diag.Errorf("nil auth method after create")
	}

	amid := amcr.GetResponse().Map["id"].(string)

	// update scope when set to primary
	if p, ok := d.GetOk(authmethodIsPrimaryAuthMethodForScopeKey); ok {
		if p.(bool) {
			if err := updateScopeWithPrimaryAuthMethodId(ctx, scopeId, amid, meta); err != nil {
				return diag.Errorf("%v", err)
			}

			amcr.GetResponse().Map[authmethodIsPrimaryAuthMethodForScopeKey] = true
		}
	}

	if err := setFromAuthMethodResponseMap(d, amcr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAuthMethodRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	amrr, err := amClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading auth method: %v", err)
	}
	if amrr == nil {
		return diag.Errorf("auth method nil after read")
	}

	serr, isPrimary := readScopeIsPrimaryAuthMethodId(ctx, amrr.GetResponse().Map["scope_id"].(string), amrr.GetResponse().Map["id"].(string), meta)
	if serr != nil {
		return diag.Errorf("%v", serr)
	}

	amrr.GetResponse().Map[authmethodIsPrimaryAuthMethodForScopeKey] = isPrimary

	if err := setFromAuthMethodResponseMap(d, amrr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAuthMethodUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	opts := []authmethods.Option{}

	if d.HasChange(NameKey) {
		opts = append(opts, authmethods.DefaultName())
		if nameVal, ok := d.GetOk(NameKey); ok {
			opts = append(opts, authmethods.WithName(nameVal.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, authmethods.DefaultDescription())
		if descVal, ok := d.GetOk(DescriptionKey); ok {
			opts = append(opts, authmethods.WithDescription(descVal.(string)))
		}
	}

	// TODO(malnick) - deprecate
	if d.HasChange(authmethodMinPasswordLengthKey) {
		opts = append(opts, authmethods.DefaultPasswordAuthMethodMinPasswordLength())
		if minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey); ok {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(minLengthVal.(int))))
		}
	}
	// TODO(malnick) - deprecate
	if d.HasChange(authmethodMinLoginNameLengthKey) {
		opts = append(opts, authmethods.DefaultPasswordAuthMethodMinLoginNameLength())
		if minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey); ok {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(minLengthVal.(int))))
		}
	}

	if d.HasChange(authmethodIsPrimaryAuthMethodForScopeKey) {
		amrr, err := amClient.Read(ctx, d.Id())
		if err != nil {
			return diag.Errorf("error updating auth method: %v", err)
		}
		if amrr == nil {
			return diag.Errorf("error updating auth method: nil resource")
		}
		scopeId := amrr.GetResponse().Map["scope_id"].(string)
		authMethodId := amrr.GetResponse().Map["id"].(string)

		isPrimary := d.Get(authmethodIsPrimaryAuthMethodForScopeKey).(bool)

		if isPrimary {
			if err := updateScopeWithPrimaryAuthMethodId(ctx, scopeId, authMethodId, meta); err != nil {
				return diag.Errorf("%v", err)
			}
		} else {
			if err := updateScopeWithPrimaryAuthMethodId(ctx, scopeId, "", meta); err != nil {
				return diag.Errorf("%v", err)
			}
		}
	}

	if len(opts) > 0 {
		opts = append(opts, authmethods.WithAutomaticVersioning(true))
		amu, err := amClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating auth method: %v", err)
		}

		if d.HasChange(authmethodIsPrimaryAuthMethodForScopeKey) {
			amu.GetResponse().Map[authmethodIsPrimaryAuthMethodForScopeKey] = d.Get(authmethodIsPrimaryAuthMethodForScopeKey).(bool)
		}

		setFromAuthMethodResponseMap(d, amu.GetResponse().Map)
	}

	// If only is_primary_for_scope changed
	if d.HasChange(authmethodIsPrimaryAuthMethodForScopeKey) {
		return resourceAuthMethodPasswordRead(ctx, d, meta)
	}

	return nil
}

func resourceAuthMethodDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	_, err := amClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting auth method: %v", err)
	}

	return nil
}
