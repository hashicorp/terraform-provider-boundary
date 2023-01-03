// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentiallibraries"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	credentialStoreIdKey                           = "credential_store_id"
	credentialLibraryVaultHttpMethodKey            = "http_method"
	credentialLibraryVaultHttpRequestBodyKey       = "http_request_body"
	credentialLibraryVaultPathKey                  = "path"
	credentialLibraryCredentialTypeKey             = "credential_type"
	credentialLibraryCredentialMappingOverridesKey = "credential_mapping_overrides"
)

var libraryVaultAttrs = []string{
	credentialLibraryVaultHttpMethodKey,
	credentialLibraryVaultHttpRequestBodyKey,
	credentialLibraryVaultPathKey,
}

func resourceCredentialLibraryVault() *schema.Resource {
	return &schema.Resource{
		Description: "The credential library for Vault resource allows you to configure a Boundary credential library for Vault.",

		CreateContext: resourceCredentialLibraryCreateVault,
		ReadContext:   resourceCredentialLibraryReadVault,
		UpdateContext: resourceCredentialLibraryUpdateVault,
		DeleteContext: resourceCredentialLibraryDeleteVault,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the Vault credential library.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The Vault credential library name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The Vault credential library description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreIdKey: {
				Description: "The ID of the credential store that this library belongs to.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			credentialLibraryVaultHttpMethodKey: {
				Description: "The HTTP method the library uses when requesting credentials from Vault. Defaults to 'GET'",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialLibraryVaultHttpRequestBodyKey: {
				Description: "The body of the HTTP request the library sends to Vault when requesting credentials. Only valid if `http_method` is set to `POST`.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialLibraryVaultPathKey: {
				Description: "The path in Vault to request credentials from.",
				Type:        schema.TypeString,
				Required:    true,
			},
			credentialLibraryCredentialTypeKey: {
				Description: "The type of credential the library generates.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialLibraryCredentialMappingOverridesKey: {
				Description: "The credential mapping override.",
				Type:        schema.TypeMap,
				Optional:    true,
			},
		},
	}
}

func setFromVaultCredentialLibraryResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw[NameKey]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw[DescriptionKey]); err != nil {
		return err
	}
	if err := d.Set(credentialStoreIdKey, raw[credentialStoreIdKey]); err != nil {
		return err
	}

	if err := d.Set(credentialLibraryCredentialMappingOverridesKey, raw[credentialLibraryCredentialMappingOverridesKey]); err != nil {
		return err
	}

	if attrsVal, ok := raw["attributes"]; ok {
		attrs := attrsVal.(map[string]interface{})
		for _, v := range libraryVaultAttrs {
			if err := d.Set(v, attrs[v]); err != nil {
				return err
			}
		}
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceCredentialLibraryCreateVault(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var opts []credentiallibraries.Option
	if v, ok := d.GetOk(NameKey); ok {
		opts = append(opts, credentiallibraries.WithName(v.(string)))
	}
	if v, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, credentiallibraries.WithDescription(v.(string)))
	}
	if v, ok := d.GetOk(credentialLibraryVaultHttpMethodKey); ok {
		opts = append(opts, credentiallibraries.WithVaultCredentialLibraryHttpMethod(v.(string)))
	}
	if v, ok := d.GetOk(credentialLibraryVaultHttpRequestBodyKey); ok {
		opts = append(opts, credentiallibraries.WithVaultCredentialLibraryHttpRequestBody(v.(string)))
	}
	if v, ok := d.GetOk(credentialLibraryVaultPathKey); ok {
		opts = append(opts, credentiallibraries.WithVaultCredentialLibraryPath(v.(string)))
	}
	if v, ok := d.GetOk(credentialLibraryCredentialTypeKey); ok {
		opts = append(opts, credentiallibraries.WithCredentialType(v.(string)))
	}
	if v, ok := d.GetOk(credentialLibraryCredentialMappingOverridesKey); ok {
		mappingOverrides := v.(map[string]interface{})
		opts = append(opts, credentiallibraries.WithCredentialMappingOverrides(mappingOverrides))
	}

	var credentialStoreId string
	cid, ok := d.GetOk(credentialStoreIdKey)
	if ok {
		credentialStoreId = cid.(string)
	} else {
		return diag.Errorf("no credential store ID is set")
	}

	client := credentiallibraries.NewClient(md.client)
	cr, err := client.Create(ctx, credentialStoreId, opts...)
	if err != nil {
		return diag.Errorf("error creating credential library: %v", err)
	}
	if cr == nil {
		return diag.Errorf("nil credential library after create")
	}

	if err := setFromVaultCredentialLibraryResponseMap(d, cr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceCredentialLibraryReadVault(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentiallibraries.NewClient(md.client)

	cr, err := client.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading credential library: %v", err)
	}
	if cr == nil {
		return diag.Errorf("credential library nil after read")
	}

	if err := setFromVaultCredentialLibraryResponseMap(d, cr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceCredentialLibraryUpdateVault(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentiallibraries.NewClient(md.client)

	var opts []credentiallibraries.Option
	if d.HasChange(NameKey) {
		opts = append(opts, credentiallibraries.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			opts = append(opts, credentiallibraries.WithName(nameVal.(string)))
		}
	}
	if d.HasChange(DescriptionKey) {
		opts = append(opts, credentiallibraries.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			opts = append(opts, credentiallibraries.WithDescription(descVal.(string)))
		}
	}
	if d.HasChange(credentialLibraryVaultHttpMethodKey) {
		opts = append(opts, credentiallibraries.DefaultVaultCredentialLibraryHttpMethod())
		v, ok := d.GetOk(credentialLibraryVaultHttpMethodKey)
		if ok {
			opts = append(opts, credentiallibraries.WithVaultCredentialLibraryHttpMethod(v.(string)))
		}
	}
	if d.HasChange(credentialLibraryVaultHttpRequestBodyKey) {
		opts = append(opts, credentiallibraries.DefaultVaultCredentialLibraryHttpRequestBody())
		v, ok := d.GetOk(credentialLibraryVaultHttpRequestBodyKey)
		if ok {
			opts = append(opts, credentiallibraries.WithVaultCredentialLibraryHttpRequestBody(v.(string)))
		}
	}
	if d.HasChange(credentialLibraryVaultPathKey) {
		v, ok := d.GetOk(credentialLibraryVaultPathKey)
		if ok {
			opts = append(opts, credentiallibraries.WithVaultCredentialLibraryPath(v.(string)))
		}
	}
	if d.HasChange(credentialLibraryCredentialMappingOverridesKey) {
		var oldMapping, newMapping map[string]interface{}
		old, new := d.GetChange(credentialLibraryCredentialMappingOverridesKey)

		if old == nil {
			old = map[string]interface{}{}
		}
		oldMapping = old.(map[string]interface{})

		if new == nil {
			new = map[string]interface{}{}
		}
		newMapping = new.(map[string]interface{})

		newAttrs := []string{}
		for k := range newMapping {
			newAttrs = append(newAttrs, k)
		}

		for oldAttr := range oldMapping {
			isRemoved := true
			for _, newAttr := range newAttrs {
				if oldAttr == newAttr {
					isRemoved = false
					break
				}
			}
			if isRemoved {
				newMapping[oldAttr] = nil
			}
		}

		opts = append(opts, credentiallibraries.WithCredentialMappingOverrides(newMapping))
	}

	if len(opts) > 0 {
		opts = append(opts, credentiallibraries.WithAutomaticVersioning(true))
		aur, err := client.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating credential library: %v", err)
		}

		if err := setFromVaultCredentialLibraryResponseMap(d, aur.GetResponse().Map); err != nil {
			return diag.Errorf("error setting credential library from response: %v", err)
		}
	}

	return nil
}

func resourceCredentialLibraryDeleteVault(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentiallibraries.NewClient(md.client)

	_, err := client.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting credential library: %v", err)
	}

	return nil
}
