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

func setFromAuthMethodResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
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
			d.Set(authmethodOidcCertificatesKey, attrs[authmethodOidcCertificatesKey].(string))
			d.Set(authmethodOidcAllowedAudiencesKey, attrs[authmethodOidcAllowedAudiencesKey].(string))
			d.Set(authmethodOidcOverrideOidcDiscoveryUrlConfigKey, attrs[authmethodOidcOverrideOidcDiscoveryUrlConfigKey].(string))
		}
	}

	d.SetId(raw["id"].(string))
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
			opts = append(opts, authmethods.WithOidcAuthMethodMaxAge(maxAge.(uint32)))
		}
		if algos, ok := d.GetOk(authmethodOidcSigningAlgorithmsKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodSigningAlgorithms(algos.([]string)))
		}
		if prefix, ok := d.GetOk(authmethodOidcApiUrlPrefixKey); ok {
			opts = append(opts, authmethods.WithOidcAuthMethodApiUrlPrefix(prefix.(string)))
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

	setFromAuthMethodResponseMap(d, amcr.GetResponse().Map)

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

	setFromAuthMethodResponseMap(d, amrr.GetResponse().Map)

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

	typeStr := d.Get(TypeKey).(string)
	switch typeStr {
	case authmethodTypePassword:
		var minLoginNameLength *int
		if d.HasChange(authmethodMinLoginNameLengthKey) {
			opts = append(opts, authmethods.DefaultPasswordAuthMethodMinLoginNameLength())
			minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey)
			if ok {
				minLengthInt := minLengthVal.(int)
				minLoginNameLength = &minLengthInt
				opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(minLengthInt)))
			}
		}

		var minPasswordLength *int
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
	if d.HasChange(authmethodMinLoginNameLengthKey) {
		d.Set(authmethodMinLoginNameLengthKey, minLoginNameLength)
	}
	if d.HasChange(authmethodMinPasswordLengthKey) {
		d.Set(authmethodMinPasswordLengthKey, minPasswordLength)
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
