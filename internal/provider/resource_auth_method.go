package provider

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	// Password auth method keys
	authmethodTypePassword          = "password"
	authmethodMinLoginNameLengthKey = "min_login_name_length"
	authmethodMinPasswordLengthKey  = "min_password_length"

	// OIDC auth method keys
	authmethodTypeOidc                              = "oidc"
	authmethodOidcStateKey                          = "state"
	authmethodOidcDiscoveryUrlKey                   = "discovery_url"
	authmethodOidcClientIdKey                       = "client_id"
	authmethodOidcClientSecretKey                   = "client_secret"
	authmethodOidcClientSecretHmacKey               = "client_secret_hmac"
	authmethodOidcMaxAgeKey                         = "max_age"
	authmethodOidcSigningAlgorithmsKey              = "signing_algorithms"
	authmethodOidcApiUrlPrefixKey                   = "api_url_prefix"
	authmethodOidcCallbackUrlKey                    = "callback_url"
	authmethodOidcCertificatesKey                   = "certificates"
	authmethodOidcAllowedAudiencesKey               = "allowed_audiences"
	authmethodOidcOverrideOidcDiscoveryUrlConfigKey = "override_oidc_discovery_url_config"
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
			},
			authmethodTypePassword: {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						authmethodMinLoginNameLengthKey: {
							Description: "The minimum login name length.",
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
						},
						authmethodMinPasswordLengthKey: {
							Description: "The minimum password length.",
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			authmethodTypeOidc: {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						authmethodOidcStateKey: {
							Description: "OIDC state",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcDiscoveryUrlKey: {
							Description: "OIDC discovery URL",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcClientIdKey: {
							Description: "OIDC client ID",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcClientSecretKey: {
							Description: "OIDC client secret",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcClientSecretHmacKey: {
							Description: "OIDC client secret HMAC",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcMaxAgeKey: {
							Description: "OIDC max age",
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcSigningAlgorithmsKey: {
							Description: "OIDC signing algorithms",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcApiUrlPrefixKey: {
							Description: "OIDC API URL prefix",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcCallbackUrlKey: {
							Description: "OIDC callback URL",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcCertificatesKey: {
							Description: "OIDC certificates",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcAllowedAudiencesKey: {
							Description: "OIDC allowed audiences",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
						authmethodOidcOverrideOidcDiscoveryUrlConfigKey: {
							Description: "OIDC discovery URL override configuration",
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
						},
					},
				},
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

	switch raw[TypeKey].(string) {
	case authmethodTypePassword:
		if attrsVal, ok := raw["attributes"]; ok {
			attrs := attrsVal.(map[string]interface{})

			minLoginNameLength := attrs[authmethodMinLoginNameLengthKey].(json.Number)
			minLoginNameLengthInt, _ := minLoginNameLength.Int64()
			if err := d.Set(authmethodMinLoginNameLengthKey, int(minLoginNameLengthInt)); err != nil {
				return err
			}

			minPasswordLength := attrs[authmethodMinPasswordLengthKey].(json.Number)
			minPasswordLengthInt, _ := minPasswordLength.Int64()
			if err := d.Set(authmethodMinPasswordLengthKey, int(minPasswordLengthInt)); err != nil {
				return err
			}
		}

	case authmethodTypeOidc:
		if attrsVal, ok := raw["attributes"]; ok {
			attrs := attrsVal.(map[string]interface{})

			authmethodOidcState := attrs[authmethodOidcStateKey]
			d.Set(authmethodOidcStateKey, authmethodOidcState.(string))
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

	var minLoginNameLength *int
	if minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey); ok {
		minLength := minLengthVal.(int)
		minLoginNameLength = &minLength
	}

	var minPasswordLength *int
	if minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey); ok {
		minLength := minLengthVal.(int)
		minPasswordLength = &minLength
	}

	opts := []authmethods.Option{}

	var typeStr string
	if typeVal, ok := d.GetOk(TypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}
	switch typeStr {
	case authmethodTypePassword:
		if minLoginNameLength != nil {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(*minLoginNameLength)))
		}
		if minPasswordLength != nil {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(*minPasswordLength)))
		}
	default:
		return diag.Errorf("invalid type provided")
	}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, authmethods.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, authmethods.WithDescription(descStr))
	}

	amClient := authmethods.NewClient(md.client)

	amcr, err := amClient.Create(ctx, typeStr, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating auth method: %v", err)
	}
	if amcr == nil {
		return diag.Errorf("nil auth method after create")
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

	if err := setFromAuthMethodResponseMap(d, amrr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAuthMethodUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	opts := []authmethods.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, authmethods.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, authmethods.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, authmethods.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, authmethods.WithDescription(descStr))
		}
	}

	var minLoginNameLength *int
	if d.HasChange(authmethodMinLoginNameLengthKey) {
		switch d.Get(TypeKey).(string) {
		case authmethodTypePassword:
			opts = append(opts, authmethods.DefaultPasswordAuthMethodMinLoginNameLength())
			minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey)
			if ok {
				minLengthInt := minLengthVal.(int)
				minLoginNameLength = &minLengthInt
				opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(minLengthInt)))
			}
		default:
			return diag.Errorf(`"min_login_name_length" cannot be used with this type of auth method`)
		}
	}

	var minPasswordLength *int
	if d.HasChange(authmethodMinPasswordLengthKey) {
		switch d.Get(TypeKey).(string) {
		case authmethodTypePassword:
			opts = append(opts, authmethods.DefaultPasswordAuthMethodMinPasswordLength())
			minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey)
			if ok {
				minLengthInt := minLengthVal.(int)
				minPasswordLength = &minLengthInt
				opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(minLengthInt)))
			}
		default:
			return diag.Errorf(`"min_password_length" cannot be used with this type of auth method`)
		}
	}

	if len(opts) > 0 {
		opts = append(opts, authmethods.WithAutomaticVersioning(true))
		_, err := amClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating auth method: %v", err)
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
	if d.HasChange(authmethodMinLoginNameLengthKey) {
		if err := d.Set(authmethodMinLoginNameLengthKey, minLoginNameLength); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(authmethodMinPasswordLengthKey) {
		if err := d.Set(authmethodMinPasswordLengthKey, minPasswordLength); err != nil {
			return diag.FromErr(err)
		}
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
