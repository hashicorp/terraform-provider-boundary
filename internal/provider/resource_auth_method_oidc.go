package provider

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	authmethodTypeOidc                                 = "oidc"
	authmethodOidcIssuerKey                            = "issuer"
	authmethodOidcClientIdKey                          = "client_id"
	authmethodOidcClientSecretKey                      = "client_secret"
	authmethodOidcMaxAgeKey                            = "max_age"
	authmethodOidcApiUrlPrefixKey                      = "api_url_prefix"
	authmethodOidcIdpCaCertsKey                        = "idp_ca_certs"
	authmethodOidcAllowedAudiencesKey                  = "allowed_audiences"
	authmethodOidcDisableDiscoveredConfigValidationKey = "disable_discovered_config_validation"
	authmethodOidcSigningAlgorithmsKey                 = "signing_algorithms"
	authmethodOidcIsPrimaryAuthMethodForScope          = "is_primary_for_scope"
	authmethodOidcAccountClaimMapsKey                  = "account_claim_maps"
	authmethodOidcClaimsScopesKey                      = "claims_scopes"

	// computed-only parameters
	authmethodOidcCallbackUrlKey      = "callback_url"
	authmethodOidcClientSecretHmacKey = "client_secret_hmac"
	authmethodOidcStateKey            = "state"
)

func resourceAuthMethodOidc() *schema.Resource {
	return &schema.Resource{
		Description: "The OIDC auth method resource allows you to configure a Boundary auth_method_oidc.",

		CreateContext: resourceAuthMethodOidcCreate,
		ReadContext:   resourceAuthMethodOidcRead,
		UpdateContext: resourceAuthMethodOidcUpdate,
		DeleteContext: resourceAuthMethodOidcDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the auth method.",
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

			// OIDC specific configurable parameters
			authmethodOidcAllowedAudiencesKey: {
				Description: "Audiences for which the provider responses will be allowed",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			authmethodOidcApiUrlPrefixKey: {
				Description: "The API prefix to use when generating callback URLs for the provider. Should be set to an address at which the provider can reach back to the controller.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authmethodOidcIdpCaCertsKey: {
				Description: "A list of CA certificates to trust when validating the IdP's token signatures.",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			authmethodOidcClientIdKey: {
				Description: "The client ID assigned to this auth method from the provider.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authmethodOidcClientSecretKey: {
				Description: "The secret key assigned to this auth method from the provider. Once set, only the hash will be kept and the original value can be removed from configuration.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authmethodOidcIssuerKey: {
				Description: "The issuer corresponding to the provider, which must match the issuer field in generated tokens.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authmethodOidcDisableDiscoveredConfigValidationKey: {
				Description: "Disables validation logic ensuring that the OIDC provider's information from its discovery endpoint matches the information here. The validation is only performed at create or update time.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			authmethodOidcMaxAgeKey: {
				Description: "The max age to provide to the provider, indicating how much time is allowed to have passed since the last authentication before the user is challenged again.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			authmethodOidcSigningAlgorithmsKey: {
				Description: "Allowed signing algorithms for the provider's issued tokens.",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			authmethodOidcAccountClaimMapsKey: {
				Description: "Account claim maps for the to_claim of sub.",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				// per comment in https://github.com/hashicorp/boundary/pull/1186
				ForceNew: true,
			},
			authmethodOidcClaimsScopesKey: {
				Description: "Claims scopes.",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},

			// OIDC specific immutable and computed parameters
			authmethodOidcClientSecretHmacKey: {
				Description: "The HMAC of the client secret returned by the Boundary controller, which is used for comparison after initial setting of the value.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcStateKey: {
				Description: "Can be one of 'inactive', 'active-private', or 'active-public'. Currently automatically set to active-public.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcCallbackUrlKey: {
				Description: "The URL that should be provided to the IdP for callbacks.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcIsPrimaryAuthMethodForScope: {
				Description: "When true, makes this auth method the primary auth method for the scope in which it resides. The primary auth method for a scope means the the user will be automatically created when they login using an OIDC account.",
				Type:        schema.TypeBool,
				Optional:    true,
			},

			TypeKey: {
				Description: "The type of auth method; hardcoded.",
				Type:        schema.TypeString,
				Default:     authmethodTypeOidc,
				Optional:    true,
			},
		},
	}
}

func setFromOidcAuthMethodResponseMap(d *schema.ResourceData, raw map[string]interface{}) diag.Diagnostics {
	d.Set(NameKey, raw[NameKey])
	d.Set(DescriptionKey, raw[DescriptionKey])
	d.Set(ScopeIdKey, raw[ScopeIdKey])
	d.Set(TypeKey, raw[TypeKey])

	if attrsVal, ok := raw["attributes"]; ok {
		attrs := attrsVal.(map[string]interface{})

		// these are always set
		d.Set(authmethodOidcStateKey, attrs[authmethodOidcStateKey].(string))
		d.Set(authmethodOidcIssuerKey, attrs[authmethodOidcIssuerKey].(string))
		d.Set(authmethodOidcClientIdKey, attrs[authmethodOidcClientIdKey].(string))
		d.Set(authmethodOidcClientSecretHmacKey, attrs[authmethodOidcClientSecretHmacKey].(string))

		if certs, ok := attrs[authmethodOidcIdpCaCertsKey]; ok {
			d.Set(authmethodOidcIdpCaCertsKey, certs.([]interface{}))
		}

		if aud, ok := attrs[authmethodOidcAllowedAudiencesKey]; ok {
			d.Set(authmethodOidcAllowedAudiencesKey, aud.([]interface{}))
		}

		if m, ok := attrs[authmethodOidcMaxAgeKey]; ok {
			maxAge := m.(json.Number)
			maxAgeInt, _ := maxAge.Int64()
			d.Set(authmethodOidcMaxAgeKey, maxAgeInt)
		}

		if val, ok := attrs[authmethodOidcApiUrlPrefixKey]; ok {
			d.Set(authmethodOidcApiUrlPrefixKey, val.(string))
		}

		if val, ok := attrs[authmethodOidcCallbackUrlKey]; ok {
			d.Set(authmethodOidcCallbackUrlKey, val.(string))
		}

		if val, ok := attrs[authmethodOidcSigningAlgorithmsKey]; ok {
			d.Set(authmethodOidcSigningAlgorithmsKey, val.([]interface{}))
		}

		if p, ok := attrs[authmethodOidcIsPrimaryAuthMethodForScope]; ok {
			d.Set(authmethodOidcIsPrimaryAuthMethodForScope, p.(bool))
		}

		if p, ok := attrs[authmethodOidcDisableDiscoveredConfigValidationKey]; ok {
			d.Set(authmethodOidcDisableDiscoveredConfigValidationKey, p.(bool))
		}

		if p, ok := attrs[authmethodOidcAccountClaimMapsKey]; ok {
			d.Set(authmethodOidcAccountClaimMapsKey, p.([]interface{}))
		}

		if p, ok := attrs[authmethodOidcClaimsScopesKey]; ok {
			d.Set(authmethodOidcClaimsScopesKey, p.([]interface{}))
		}
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceAuthMethodOidcCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var typeStr string
	if typeVal, ok := d.GetOk(TypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}

	opts := []authmethods.Option{}

	opts = append(opts, authmethods.WithAutomaticVersioning(true))

	if issuer, ok := d.GetOk(authmethodOidcIssuerKey); ok {
		opts = append(opts, authmethods.WithOidcAuthMethodIssuer(issuer.(string)))
	}
	if clientId, ok := d.GetOk(authmethodOidcClientIdKey); ok {
		opts = append(opts, authmethods.WithOidcAuthMethodClientId(clientId.(string)))
	}
	if clientSecret, ok := d.GetOk(authmethodOidcClientSecretKey); ok {
		opts = append(opts, authmethods.WithOidcAuthMethodClientSecret(clientSecret.(string)))
	}
	if maxAge, ok := d.GetOk(authmethodOidcMaxAgeKey); ok {
		opts = append(opts, authmethods.WithOidcAuthMethodMaxAge(uint32(maxAge.(int))))
	}
	if prefix, ok := d.GetOk(authmethodOidcApiUrlPrefixKey); ok {
		opts = append(opts, authmethods.WithOidcAuthMethodApiUrlPrefix(prefix.(string)))
	}
	if certs, ok := d.GetOk(authmethodOidcIdpCaCertsKey); ok {
		certList := []string{}
		for _, c := range certs.([]interface{}) {
			certList = append(certList, c.(string))
		}

		opts = append(opts, authmethods.WithOidcAuthMethodIdpCaCerts(certList))
	}
	if aud, ok := d.GetOk(authmethodOidcAllowedAudiencesKey); ok {
		audList := []string{}
		for _, c := range aud.([]interface{}) {
			audList = append(audList, c.(string))
		}
		opts = append(opts, authmethods.WithOidcAuthMethodAllowedAudiences(audList))
	}
	if dis, ok := d.GetOk(authmethodOidcDisableDiscoveredConfigValidationKey); ok {
		opts = append(opts, authmethods.WithOidcAuthMethodDisableDiscoveredConfigValidation(dis.(bool)))
	}
	if algos, ok := d.GetOk(authmethodOidcSigningAlgorithmsKey); ok {
		algoList := []string{}
		for _, c := range algos.([]interface{}) {
			algoList = append(algoList, c.(string))
		}
		opts = append(opts, authmethods.WithOidcAuthMethodSigningAlgorithms(algoList))
	}

	if claims, ok := d.GetOk(authmethodOidcAccountClaimMapsKey); ok {
		cList := []string{}
		for _, c := range claims.([]interface{}) {
			cList = append(cList, c.(string))
		}
		opts = append(opts, authmethods.WithOidcAuthMethodAccountClaimMaps(cList))
	}

	if claimsScopes, ok := d.GetOk(authmethodOidcClaimsScopesKey); ok {
		cList := []string{}
		for _, c := range claimsScopes.([]interface{}) {
			cList = append(cList, c.(string))
		}
		opts = append(opts, authmethods.WithOidcAuthMethodClaimsScopes(cList))
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

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	amClient := authmethods.NewClient(md.client)

	amcr, err := amClient.Create(ctx, typeStr, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating auth method: %v", err)
	}
	if amcr == nil {
		return diag.Errorf("nil auth method after create")
	}
	// need to update state is create was successful to avoid loosing state if the change state or update
	// scope step that follows errors out
	if err := setFromOidcAuthMethodResponseMap(d, amcr.GetResponse().Map); err != nil {
		return diag.Errorf("%v", err)
	}

	amid := amcr.GetResponse().Map["id"].(string)

	// auto set to active-public state
	_, err = amClient.ChangeState(ctx, amid, 0, "active-public", authmethods.WithAutomaticVersioning(true))
	if err != nil {
		return diag.Errorf("%v", err)
	}

	// update scope when set to primary
	if p, ok := d.GetOk(authmethodOidcIsPrimaryAuthMethodForScope); ok {
		if p.(bool) {
			if err := updateScopeWithPrimaryAuthMethodId(ctx, scopeId, amid, meta); err != nil {
				return diag.Errorf("%v", err)
			}

			amcr.GetResponse().Map[authmethodOidcIsPrimaryAuthMethodForScope] = true
		}
	}

	return setFromOidcAuthMethodResponseMap(d, amcr.GetResponse().Map)
}

func updateScopeWithPrimaryAuthMethodId(ctx context.Context, scopeId, authmethodId string, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)

	opts := []scopes.Option{}
	opts = append(opts, scopes.WithAutomaticVersioning(true))
	opts = append(opts, scopes.WithPrimaryAuthMethodId(authmethodId))

	_, err := scp.Update(ctx, scopeId, 0, opts...)
	if err != nil {
		return diag.Errorf("error updating scope: %v", err)
	}

	return nil
}

func readScopeIsPrimaryAuthMethodId(ctx context.Context, scopeId, authmethodId string, meta interface{}) (diag.Diagnostics, bool) {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)

	srr, err := scp.Read(ctx, scopeId)
	if err != nil {
		return diag.Errorf("%s", err), false
	}

	if p, ok := srr.GetResponse().Map["primary_auth_method_id"]; ok {
		if p == authmethodId {
			return nil, true
		}
	}

	return nil, false
}

func resourceAuthMethodOidcRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
	if err != nil {
		return diag.Errorf("%v", serr)
	}

	if isPrimary {
		amrr.GetResponse().Map[authmethodOidcIsPrimaryAuthMethodForScope] = true
	}

	return setFromOidcAuthMethodResponseMap(d, amrr.GetResponse().Map)
}

func resourceAuthMethodOidcUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	opts := []authmethods.Option{}

	if d.HasChange(NameKey) {
		opts = append(opts, authmethods.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			opts = append(opts, authmethods.WithName(nameVal.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, authmethods.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			opts = append(opts, authmethods.WithDescription(descVal.(string)))
		}
	}

	if d.HasChange(authmethodOidcIssuerKey) {
		if issuer, ok := d.GetOk(authmethodOidcIssuerKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodIssuer(issuer.(string)))
		}
	}
	if d.HasChange(authmethodOidcClientIdKey) {
		if clientId, ok := d.GetOk(authmethodOidcClientIdKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodClientId(clientId.(string)))
		}
	}
	if d.HasChange(authmethodOidcClientSecretKey) {
		if clientSecret, ok := d.GetOk(authmethodOidcClientSecretKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodClientSecret(clientSecret.(string)))
		}
	}
	if d.HasChange(authmethodOidcMaxAgeKey) {
		if maxAge, ok := d.GetOk(authmethodOidcMaxAgeKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodMaxAge(uint32(maxAge.(int))))
		}
	}
	if d.HasChange(authmethodOidcSigningAlgorithmsKey) {
		if algos, ok := d.GetOk(authmethodOidcSigningAlgorithmsKey); ok {
			var signingAlgs []string
			for _, alg := range algos.([]interface{}) {
				signingAlgs = append(signingAlgs, alg.(string))
			}
			opts = append(opts, authmethods.WithOidcAuthMethodSigningAlgorithms(signingAlgs))
		}
	}
	if d.HasChange(authmethodOidcApiUrlPrefixKey) {
		if prefix, ok := d.GetOk(authmethodOidcApiUrlPrefixKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodApiUrlPrefix(prefix.(string)))
		}
	}
	if d.HasChange(authmethodOidcClientSecretHmacKey) {
		if sec, ok := d.GetOk(authmethodOidcClientSecretHmacKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodClientSecret(sec.(string)))
		}
	}
	if d.HasChange(authmethodOidcAllowedAudiencesKey) {
		if val, ok := d.GetOk(authmethodOidcAllowedAudiencesKey); ok {
			var audiences []string
			for _, aud := range val.([]interface{}) {
				audiences = append(audiences, aud.(string))
			}
			opts = append(opts, authmethods.WithOidcAuthMethodAllowedAudiences(audiences))
		}
	}
	if d.HasChange(authmethodOidcIdpCaCertsKey) {
		if val, ok := d.GetOk(authmethodOidcIdpCaCertsKey); ok {
			certs := []string{}
			for _, cert := range val.([]interface{}) {
				certs = append(certs, cert.(string))
			}
			opts = append(opts, authmethods.WithOidcAuthMethodIdpCaCerts(certs))
		}
	}
	if d.HasChange(authmethodOidcDisableDiscoveredConfigValidationKey) {
		if val, ok := d.GetOk(authmethodOidcDisableDiscoveredConfigValidationKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodDisableDiscoveredConfigValidation(val.(bool)))
		}
	}
	if d.HasChange(authmethodOidcClaimsScopesKey) {
		if val, ok := d.GetOk(authmethodOidcClaimsScopesKey); ok {
			claimsScopes := []string{}
			for _, c := range val.([]interface{}) {
				claimsScopes = append(claimsScopes, c.(string))
			}
			opts = append(opts, authmethods.WithOidcAuthMethodClaimsScopes(claimsScopes))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, authmethods.WithAutomaticVersioning(true))
		amur, err := amClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating auth method: %v", err)
		}

		if d.HasChange(authmethodOidcIsPrimaryAuthMethodForScope) {
			if p, ok := d.GetOk(authmethodOidcIsPrimaryAuthMethodForScope); ok {
				if p.(bool) {
					if err := updateScopeWithPrimaryAuthMethodId(
						ctx,
						amur.GetResponse().Map["scope_id"].(string),
						amur.GetResponse().Map["id"].(string),
						meta); err != nil {
						return diag.Errorf("%v", err)
					}

					amur.GetResponse().Map[authmethodOidcIsPrimaryAuthMethodForScope] = true
				}
			}
		}

		return setFromOidcAuthMethodResponseMap(d, amur.GetResponse().Map)
	}
	return nil
}

func resourceAuthMethodOidcDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	_, err := amClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting auth method: %v", err)
	}

	return nil
}
