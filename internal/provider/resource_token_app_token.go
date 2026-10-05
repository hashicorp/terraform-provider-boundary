// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"math"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/apptokens"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	AppTokenTimeToLiveSecondsKey         = "time_to_live_seconds"
	AppTokenTimeToStaleSecondsKey        = "time_to_stale_seconds"
	AppTokenPermissionsKey               = "permissions"
	AppTokenPermissionGrantsKey          = "grants"
	AppTokenPermissionLabelKey           = "label"
	AppTokenPermissionGrantScopesKey     = "grant_scopes"
	AppTokenPermissionDeletedScopesKey   = "deleted_scopes"
	AppTokenDeletedScopeScopeIdKey       = "scope_id"
	AppTokenDeletedScopeDeletedAtKey     = "deleted_at"
	AppTokenTokenKey                     = "token"
	AppTokenStatusKey                    = "status"
	AppTokenExpirationTimeKey            = "expiration_time"
	AppTokenCreatedByUserIdKey           = "created_by_user_id"
	AppTokenApproximateLastAccessTimeKey = "approximate_last_access_time"
	AppTokenActiveStatus                 = "active"
)

func resourceAppToken() *schema.Resource {
	return &schema.Resource{
		Description: "The app token resource allows you to configure a Boundary app_token. App tokens are immutable once created; any change to a field will destroy and recreate the resource.",

		CreateContext: resourceAppTokenCreate,
		ReadContext:   resourceAppTokenRead,
		DeleteContext: resourceAppTokenDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the app token.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The app token name. Must be unique within the scope, if provided.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			DescriptionKey: {
				Description: "The app token description.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID of the app token.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			AppTokenTimeToLiveSecondsKey: {
				Description: "The time-to-live of the app token, in seconds.",
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				// Between 1 second and 3 years (in seconds).
				ValidateFunc: validation.IntBetween(1, 3*365*24*60*60),
			},
			AppTokenTimeToStaleSecondsKey: {
				Description: "The app token must be used within this many seconds, reset each time the token is used. Set to 0 for no timeout. Defaults to no timeout.",
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				// Upper bound is the uint32 max, matching the API/server field width; values above it would overflow on conversion.
				ValidateFunc: validation.IntBetween(0, math.MaxUint32),
			},
			AppTokenPermissionsKey: {
				Description: "The app token permissions.",
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						AppTokenPermissionGrantsKey: {
							Description: "The grants for this permission.",
							Type:        schema.TypeList,
							Required:    true,
							ForceNew:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						AppTokenPermissionLabelKey: {
							Description: "The label for this permission",
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
						},
						AppTokenPermissionGrantScopesKey: {
							Description: "The grant scopes for this permission.",
							Type:        schema.TypeList,
							Required:    true,
							ForceNew:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						AppTokenPermissionDeletedScopesKey: {
							Description: "Scopes that were directly granted to this permission but have since been deleted. Populated by the server; read-only.",
							Type:        schema.TypeList,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									AppTokenDeletedScopeScopeIdKey: {
										Description: "The ID of the deleted scope.",
										Type:        schema.TypeString,
										Computed:    true,
									},
									AppTokenDeletedScopeDeletedAtKey: {
										Description: "The time the scope was deleted.",
										Type:        schema.TypeString,
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
			AppTokenTokenKey: {
				Description: "The app token value. Only populated at creation time.",
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
			},
			AppTokenStatusKey: {
				Description: "The current status of the app token (active, expired, stale, or revoked).",
				Type:        schema.TypeString,
				Computed:    true,
			},
			AppTokenExpirationTimeKey: {
				Description: "The time the app token expires.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			AppTokenCreatedByUserIdKey: {
				Description: "The ID of the user that created the app token.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			AppTokenApproximateLastAccessTimeKey: {
				Description: "The approximate time the app token was last used.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func setFromAppTokenResponseMap(d *schema.ResourceData, raw map[string]any) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw["scope_id"]); err != nil {
		return err
	}
	if err := d.Set(AppTokenStatusKey, raw["status"]); err != nil {
		return err
	}
	if err := d.Set(AppTokenExpirationTimeKey, raw["expiration_time"]); err != nil {
		return err
	}
	if err := d.Set(AppTokenCreatedByUserIdKey, raw["created_by_user_id"]); err != nil {
		return err
	}
	if err := d.Set(AppTokenApproximateLastAccessTimeKey, raw["approximate_last_access_time"]); err != nil {
		return err
	}
	if v, ok := raw["time_to_live_seconds"].(json.Number); ok {
		ttl, _ := v.Int64()
		if err := d.Set(AppTokenTimeToLiveSecondsKey, int(ttl)); err != nil {
			return err
		}
	}
	if v, ok := raw["time_to_stale_seconds"].(json.Number); ok {
		tts, _ := v.Int64()
		if err := d.Set(AppTokenTimeToStaleSecondsKey, int(tts)); err != nil {
			return err
		}
	}
	if err := d.Set(AppTokenPermissionsKey, raw["permissions"]); err != nil {
		return err
	}

	d.SetId(raw["id"].(string))
	return nil
}

func resourceAppTokenCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	md := meta.(*metaData)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	// TODO: I couldnt find anything in the rfc about this, but TTL doesnt seem to have a special
	// zero value like timeToStale? If it does, replace this with
	// the pattern used for TimeToStale thats below.
	var timeToLive int
	if ttlVal, ok := d.GetOk(AppTokenTimeToLiveSecondsKey); ok {
		timeToLive = ttlVal.(int)
	} else {
		return diag.Errorf("no time_to_live_seconds provided")
	}

	permsRaw, ok := d.GetOk(AppTokenPermissionsKey)
	if !ok {
		return diag.Errorf("no permissions provided")
	}

	var permissions []*apptokens.AppTokenPermission
	for _, p := range permsRaw.([]any) {
		permMap := p.(map[string]any)

		var grants []string
		for _, g := range permMap[AppTokenPermissionGrantsKey].([]any) {
			grants = append(grants, g.(string))
		}

		var grantScopes []string
		for _, gs := range permMap[AppTokenPermissionGrantScopesKey].([]any) {
			grantScopes = append(grantScopes, gs.(string))
		}

		var label string
		if permMap[AppTokenPermissionLabelKey] != nil {
			label = permMap[AppTokenPermissionLabelKey].(string)
		}

		permissions = append(permissions, &apptokens.AppTokenPermission{
			Label:       label,
			Grants:      grants,
			GrantScopes: grantScopes,
		})
	}

	opts := []apptokens.Option{
		apptokens.WithTimeToLive(uint32(timeToLive)),
		apptokens.WithPermissions(permissions),
	}

	if nameVal, ok := d.GetOk(NameKey); ok {
		opts = append(opts, apptokens.WithName(nameVal.(string)))
	}

	if descVal, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, apptokens.WithDescription(descVal.(string)))
	}

	// We can't use GetOk here since it throws away zero values, and the zero value of AppTokenTimeToStale
	// represents "no time to stale."
	if ttsAttr := d.GetRawConfig().GetAttr(AppTokenTimeToStaleSecondsKey); ttsAttr.IsKnown() && !ttsAttr.IsNull() {
		ttsBigFloat := ttsAttr.AsBigFloat()
		ttsVal, _ := ttsBigFloat.Int64()
		opts = append(opts, apptokens.WithTimeToStale(uint32(ttsVal)))
	}

	atClient := apptokens.NewClient(md.client)

	atcr, err := atClient.Create(ctx, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating app token: %v", err)
	}
	if atcr == nil {
		return diag.Errorf("nil app token after create")
	}

	if tokenVal, ok := atcr.GetResponse().Map["token"]; ok {
		if err := d.Set(AppTokenTokenKey, tokenVal); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := setFromAppTokenResponseMap(d, atcr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAppTokenDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	md := meta.(*metaData)
	atClient := apptokens.NewClient(md.client)

	_, err := atClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting app token: %v", err)
	}

	return nil
}

// Since the token value is a secret, all reads done after token creation should not return the token value.
// This is implemented as read is required to be implemented for resources in a terraform provider.
func resourceAppTokenRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	md := meta.(*metaData)
	atClient := apptokens.NewClient(md.client)

	atrr, err := atClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading app token: %v", err)
	}
	if atrr == nil {
		return diag.Errorf("app token nil after read")
	}

	if err := setFromAppTokenResponseMap(d, atrr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
