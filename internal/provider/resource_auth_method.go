package provider

import (
	"context"
	"encoding/json"
	"fmt"
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

			// these are always set
			d.Set(authmethodOidcStateKey, attrs[authmethodOidcStateKey].(string))
			d.Set(authmethodOidcIssuerKey, attrs[authmethodOidcIssuerKey].(string))
			d.Set(authmethodOidcClientIdKey, attrs[authmethodOidcClientIdKey].(string))
			d.Set(authmethodOidcClientSecretHmacKey, attrs[authmethodOidcClientSecretHmacKey].(string))
			d.Set(authmethodOidcCaCertificatesKey, attrs[authmethodOidcCaCertificatesKey].([]interface{}))
			d.Set(authmethodOidcAllowedAudiencesKey, attrs[authmethodOidcAllowedAudiencesKey].([]interface{}))

			fmt.Printf("ca certs: %s\n", d.Get(authmethodOidcCaCertificatesKey))

			// TODO(malnick) remove after testing
			/*
				strArys := []string{authmethodOidcCaCertificatesKey, authmethodOidcAllowedAudiencesKey}

				for _, k := range strArys {
					kAry := []string{}
					for _, val := range attrs[k].([]interface{}) {
						kAry = append(kAry, val.(string))
					}
					d.Set(k, kAry)
					fmt.Printf("%s: %s\n", k, d.Get(k))
				}
			*/

			maxAge := attrs[authmethodOidcMaxAgeKey].(json.Number)
			maxAgeInt, _ := maxAge.Int64()
			d.Set(authmethodOidcMaxAgeKey, maxAgeInt)

			// these are set sometimes
			sometimesString := []string{
				authmethodOidcApiUrlPrefixKey,
				authmethodOidcCallbackUrlKey,
				authmethodOidcDisableDiscoveredConfigValidationKey}

			for _, k := range sometimesString {
				if val, ok := attrs[k]; ok {
					d.Set(k, val.(string))
				}
			}

			if val, ok := attrs[authmethodOidcSigningAlgorithmsKey]; ok {
				d.Set(authmethodOidcSigningAlgorithmsKey, val.([]interface{}))
			}
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

	typeStr := d.Get(TypeKey).(string)
	switch typeStr {
	case authmethodTypePassword:
		if d.HasChange(authmethodMinLoginNameLengthKey) {
			opts = append(opts, authmethods.DefaultPasswordAuthMethodMinLoginNameLength())
			minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey)
			if ok {
				opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(minLengthVal.(int))))
			}
		}

		if d.HasChange(authmethodMinPasswordLengthKey) {
			opts = append(opts, authmethods.DefaultPasswordAuthMethodMinPasswordLength())
			minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey)
			if ok {
				opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(minLengthVal.(int))))
			}
		}

	case authmethodTypeOidc:
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
				opts = append(opts, authmethods.WithOidcAuthMethodSigningAlgorithms(algos.([]string)))
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
				opts = append(opts, authmethods.WithOidcAuthMethodAllowedAudiences(val.([]string)))
			}
		}
		if d.HasChange(authmethodOidcCaCertificatesKey) {
			if val, ok := d.GetOk(authmethodOidcCaCertificatesKey); ok {
				opts = append(opts, authmethods.WithOidcAuthMethodCaCerts(val.([]string)))
			}
		}
		if d.HasChange(authmethodOidcDisableDiscoveredConfigValidationKey) {
			if val, ok := d.GetOk(authmethodOidcDisableDiscoveredConfigValidationKey); ok {
				opts = append(opts, authmethods.WithOidcAuthMethodDisableDiscoveredConfigValidation(val.(bool)))
			}
		}
	default:
		return errorInvalidAuthMethodType
	}

	if len(opts) > 0 {
		opts = append(opts, authmethods.WithAutomaticVersioning(true))
		amur, err := amClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating auth method: %v", err)
		}

		return setFromAuthMethodResponseMap(d, amur.GetResponse().Map)
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
