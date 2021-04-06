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

var errorInvalidAuthMethodType = diag.Errorf("invalid auth method type, must be 'password' or 'oidc'")

func setFromAuthMethodResponseMap(d *schema.ResourceData, raw map[string]interface{}) diag.Diagnostics {
	d.Set(NameKey, raw[NameKey])
	d.Set(DescriptionKey, raw[DescriptionKey])
	d.Set(ScopeIdKey, raw[ScopeIdKey])
	d.Set(TypeKey, raw[TypeKey])

	switch raw[TypeKey].(string) {
	case authmethodTypePassword:
		if attrsVal, ok := raw["attributes"]; ok {
			attrs := attrsVal.(map[string]interface{})

			minLoginNameLength := attrs[authmethodMinLoginNameLengthKey].(json.Number)
			minLoginNameLengthInt, _ := minLoginNameLength.Int64()
			d.Set(authmethodMinLoginNameLengthKey, int(minLoginNameLengthInt))

			minPasswordLength := attrs[authmethodMinPasswordLengthKey].(json.Number)
			minPasswordLengthInt, _ := minPasswordLength.Int64()
			d.Set(authmethodMinPasswordLengthKey, int(minPasswordLengthInt))
		}

	case authmethodTypeOidc:
		if attrsVal, ok := raw["attributes"]; ok {
			attrs := attrsVal.(map[string]interface{})

			d.Set(authmethodOidcStateKey, attrs[authmethodOidcStateKey].(string))
			d.Set(authmethodOidcIssuerKey, attrs[authmethodOidcIssuerKey].(string))
			d.Set(authmethodOidcClientIdKey, attrs[authmethodOidcClientIdKey].(string))
			d.Set(authmethodOidcClientSecretKey, attrs[authmethodOidcClientSecretKey].(string))
			d.Set(authmethodOidcClientSecretHmacKey, attrs[authmethodOidcClientSecretHmacKey].(string))
			d.Set(authmethodOidcMaxAgeKey, attrs[authmethodOidcMaxAgeKey].(string))
			d.Set(authmethodOidcSigningAlgorithmsKey, attrs[authmethodOidcSigningAlgorithmsKey].(string))
			d.Set(authmethodOidcApiUrlPrefixKey, attrs[authmethodOidcApiUrlPrefixKey].(string))
			d.Set(authmethodOidcCallbackUrlKey, attrs[authmethodOidcCallbackUrlKey].(string))
			d.Set(authmethodOidcCaCertificatesKey, attrs[authmethodOidcCaCertificatesKey].(string))
			d.Set(authmethodOidcAllowedAudiencesKey, attrs[authmethodOidcAllowedAudiencesKey].(string))
			d.Set(authmethodOidcDisableDiscoveredConfigValidationKey, attrs[authmethodOidcDisableDiscoveredConfigValidationKey].(string))
		}
	default:
		return errorInvalidAuthMethodType
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceAuthMethodCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var typeStr string
	if typeVal, ok := d.GetOk(TypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}

	opts := []authmethods.Option{}
	switch typeStr {
	case authmethodTypePassword:
		var minLoginNameLength *int
		if minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey); ok {
			minLength := minLengthVal.(int)
			minLoginNameLength = &minLength
		}
		if minLoginNameLength != nil {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(*minLoginNameLength)))
		}

		var minPasswordLength *int
		if minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey); ok {
			minLength := minLengthVal.(int)
			minPasswordLength = &minLength
		}
		if minPasswordLength != nil {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(*minPasswordLength)))
		}

	case authmethodTypeOidc:
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
		if certs, ok := d.GetOk(authmethodOidcCaCertificatesKey); ok {
			certList := []string{}
			for _, c := range certs.([]interface{}) {
				certList = append(certList, c.(string))
			}

			opts = append(opts, authmethods.WithOidcAuthMethodCaCerts(certList))
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

	default:
		return errorInvalidAuthMethodType
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

	return setFromAuthMethodResponseMap(d, amcr.GetResponse().Map)
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

	return setFromAuthMethodResponseMap(d, amrr.GetResponse().Map)
}

func resourceAuthMethodUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	opts := []authmethods.Option{}

	var (
		// password auth method values for updating
		name               *string
		desc               *string
		minLoginNameLength *int
		minPasswordLength  *int

		// oidc auth method values for updating
		oidcIssuer       *string
		oidcClientId     *string
		oidcClientSecret *string
		oidcMaxAge       *int
		oidcSigningAlgos *[]string
		oidcUrlPrefix    *string
	)

	if d.HasChange(NameKey) {
		opts = append(opts, authmethods.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, authmethods.WithName(nameStr))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, authmethods.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, authmethods.WithDescription(descStr))
		}
	}

	typeStr := d.Get(TypeKey).(string)
	switch typeStr {
	case authmethodTypePassword:
		if d.HasChange(authmethodMinLoginNameLengthKey) {
			opts = append(opts, authmethods.DefaultPasswordAuthMethodMinLoginNameLength())
			minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey)
			if ok {
				minLengthInt := minLengthVal.(int)
				minLoginNameLength = &minLengthInt
				opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(minLengthInt)))
			}
		}

		if d.HasChange(authmethodMinPasswordLengthKey) {
			opts = append(opts, authmethods.DefaultPasswordAuthMethodMinPasswordLength())
			minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey)
			if ok {
				minLengthInt := minLengthVal.(int)
				minPasswordLength = &minLengthInt
				opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(minLengthInt)))
			}
		}

	case authmethodTypeOidc:
		if d.HasChange(authmethodOidcIssuerKey) {
			if issuer, ok := d.GetOk(authmethodOidcIssuerKey); ok {
				issuerStr := issuer.(string)
				oidcIssuer = &issuerStr
				opts = append(opts, authmethods.WithOidcAuthMethodIssuer(*oidcIssuer))
			}
		}
		if d.HasChange(authmethodOidcClientIdKey) {
			if clientId, ok := d.GetOk(authmethodOidcClientIdKey); ok {
				oidcClientIdStr := clientId.(string)
				oidcClientId = &oidcClientIdStr
				opts = append(opts, authmethods.WithOidcAuthMethodClientId(*oidcClientId))
			}
		}
		if d.HasChange(authmethodOidcClientSecretKey) {
			if clientSecret, ok := d.GetOk(authmethodOidcClientSecretKey); ok {
				oidcClientSecretStr := clientSecret.(string)
				oidcClientSecret = &oidcClientSecretStr
				opts = append(opts, authmethods.WithOidcAuthMethodClientSecret(*oidcClientSecret))
			}
		}
		if d.HasChange(authmethodOidcMaxAgeKey) {
			if maxAge, ok := d.GetOk(authmethodOidcMaxAgeKey); ok {
				oidcMaxAgeStr := maxAge.(int)
				oidcMaxAge = &oidcMaxAgeStr
				opts = append(opts, authmethods.WithOidcAuthMethodMaxAge(uint32(*oidcMaxAge)))
			}
		}
		if d.HasChange(authmethodOidcSigningAlgorithmsKey) {
			if algos, ok := d.GetOk(authmethodOidcSigningAlgorithmsKey); ok {
				oidcSigningAlgosAry := algos.([]string)
				oidcSigningAlgos = &oidcSigningAlgosAry
				opts = append(opts, authmethods.WithOidcAuthMethodSigningAlgorithms(*oidcSigningAlgos))
			}
		}
		if d.HasChange(authmethodOidcApiUrlPrefixKey) {
			if prefix, ok := d.GetOk(authmethodOidcApiUrlPrefixKey); ok {
				oidcUrlPrefixStr := prefix.(string)
				oidcUrlPrefix = &oidcUrlPrefixStr
				opts = append(opts, authmethods.WithOidcAuthMethodApiUrlPrefix(*oidcUrlPrefix))
			}
		}

	default:
		return errorInvalidAuthMethodType
	}

	if len(opts) > 0 {
		opts = append(opts, authmethods.WithAutomaticVersioning(true))
		_, err := amClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating auth method: %v", err)
		}
	}

	if d.HasChange(NameKey) {
		d.Set(NameKey, name)
	}
	if d.HasChange(DescriptionKey) {
		d.Set(DescriptionKey, desc)
	}

	switch typeStr {
	case authmethodTypePassword:
		if d.HasChange(authmethodMinLoginNameLengthKey) {
			d.Set(authmethodMinLoginNameLengthKey, minLoginNameLength)
		}
		if d.HasChange(authmethodMinPasswordLengthKey) {
			d.Set(authmethodMinPasswordLengthKey, minPasswordLength)
		}
	case authmethodTypeOidc:
		if d.HasChange(authmethodOidcIssuerKey) {
			d.Set(authmethodOidcIssuerKey, oidcIssuer)
		}
		if d.HasChange(authmethodOidcClientIdKey) {
			d.Set(authmethodOidcClientIdKey, oidcClientId)
		}
		if d.HasChange(authmethodOidcClientSecretKey) {
			d.Set(authmethodOidcClientSecretKey, oidcClientSecret)
		}
		if d.HasChange(authmethodOidcMaxAgeKey) {
			d.Set(authmethodOidcMaxAgeKey, oidcMaxAge)
		}
		if d.HasChange(authmethodOidcSigningAlgorithmsKey) {
			d.Set(authmethodOidcSigningAlgorithmsKey, oidcSigningAlgos)
		}
		if d.HasChange(authmethodOidcApiUrlPrefixKey) {
			d.Set(authmethodOidcApiUrlPrefixKey, oidcUrlPrefix)
		}

	default:
		return errorInvalidAuthMethodType
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
