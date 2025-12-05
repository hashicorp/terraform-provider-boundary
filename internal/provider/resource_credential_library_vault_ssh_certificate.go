// Copyright IBM Corp. 2020, 2025
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
	credentialLibraryVaultSshCertificateType                         = "vault-ssh-certificate"
	credentialLibraryVaultSshCertificatePathKey                      = "path"
	credentialLibraryVaultSshCertificateUsernameKey                  = "username"
	credentialLibraryVaultSshCertificateKeyTypeKey                   = "key_type"
	credentialLibraryVaultSshCertificateKeyBitsKey                   = "key_bits"
	credentialLibraryVaultSshCertificateTtlKey                       = "ttl"
	credentialLibraryVaultSshCertificateKeyIdKey                     = "key_id"
	credentialLibraryVaultSshCertificateCriticalOptionsKey           = "critical_options"
	credentialLibraryVaultSshCertificateExtensionsKey                = "extensions"
	credentialLibraryVaultSshCertificateAdditionalValidPrincipalsKey = "additional_valid_principals"
)

var libraryVaultSshCertificateAttrs = []string{
	credentialLibraryVaultSshCertificatePathKey,
	credentialLibraryVaultSshCertificateUsernameKey,
	credentialLibraryVaultSshCertificateKeyTypeKey,
	credentialLibraryVaultSshCertificateKeyBitsKey,
	credentialLibraryVaultSshCertificateTtlKey,
	credentialLibraryVaultSshCertificateKeyIdKey,
	credentialLibraryVaultSshCertificateCriticalOptionsKey,
	credentialLibraryVaultSshCertificateExtensionsKey,
	credentialLibraryVaultSshCertificateAdditionalValidPrincipalsKey,
}

func resourceCredentialLibraryVaultSshCertificate() *schema.Resource {
	return &schema.Resource{
		Description: "The credential library for Vault resource allows you to configure a Boundary credential library for Vault.",

		CreateContext: resourceCredentialLibraryCreateVaultSshCertificate,
		ReadContext:   resourceCredentialLibraryReadVaultSshCertificate,
		UpdateContext: resourceCredentialLibraryUpdateVaultSshCertificate,
		DeleteContext: resourceCredentialLibraryDeleteVaultSshCertificate,
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
			credentialLibraryVaultSshCertificatePathKey: {
				Description: "The path in Vault to request credentials from.",
				Type:        schema.TypeString,
				Required:    true,
			},
			credentialLibraryVaultSshCertificateUsernameKey: {
				Description: "The username to use with the certificate returned by the library.",
				Type:        schema.TypeString,
				Required:    true,
			},
			credentialLibraryVaultSshCertificateKeyTypeKey: {
				Description: "Specifies the desired key type; must be ed25519, ecdsa, or rsa.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialLibraryVaultSshCertificateKeyBitsKey: {
				Description: "Specifies the number of bits to use for the generated keys.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			credentialLibraryVaultSshCertificateTtlKey: {
				Description: "Specifies the requested time to live for a certificate returned from the library.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialLibraryVaultSshCertificateKeyIdKey: {
				Description: "Specifies the key id a certificate should have.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialLibraryVaultSshCertificateCriticalOptionsKey: {
				Description: "Specifies a map of the critical options that the certificate should be signed for.",
				Type:        schema.TypeMap,
				Optional:    true,
			},
			credentialLibraryVaultSshCertificateExtensionsKey: {
				Description: "Specifies a map of the extensions that the certificate should be signed for.",
				Type:        schema.TypeMap,
				Optional:    true,
			},
			credentialLibraryVaultSshCertificateAdditionalValidPrincipalsKey: {
				Description: "Principals to be signed as \"valid_principles\" in addition to username.",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
		},
	}
}

func setFromVaultSshCertificateCredentialLibraryResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw[NameKey]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw[DescriptionKey]); err != nil {
		return err
	}
	if err := d.Set(credentialStoreIdKey, raw[credentialStoreIdKey]); err != nil {
		return err
	}

	if attrsVal, ok := raw["attributes"]; ok {
		attrs := attrsVal.(map[string]interface{})
		for _, v := range libraryVaultSshCertificateAttrs {
			if err := d.Set(v, attrs[v]); err != nil {
				return err
			}
		}
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceCredentialLibraryCreateVaultSshCertificate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var opts []credentiallibraries.Option
	if v, ok := d.GetOk(NameKey); ok {
		opts = append(opts, credentiallibraries.WithName(v.(string)))
	}
	if v, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, credentiallibraries.WithDescription(v.(string)))
	}
	if v, ok := d.GetOk(credentialLibraryVaultSshCertificatePathKey); ok {
		opts = append(opts, credentiallibraries.WithVaultCredentialLibraryPath(v.(string)))
	}
	if v, ok := d.GetOk(credentialLibraryVaultSshCertificateUsernameKey); ok {
		opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryUsername(v.(string)))
	}
	if v, ok := d.GetOk(credentialLibraryVaultSshCertificateKeyTypeKey); ok {
		opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryKeyType(v.(string)))
	}
	if v, ok := d.GetOk(credentialLibraryVaultSshCertificateKeyBitsKey); ok {
		opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryKeyBits(uint32(v.(int))))
	}
	if v, ok := d.GetOk(credentialLibraryVaultSshCertificateTtlKey); ok {
		opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryTtl(v.(string)))
	}
	if v, ok := d.GetOk(credentialLibraryVaultSshCertificateKeyIdKey); ok {
		opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryKeyId(v.(string)))
	}
	if v, ok := d.GetOk(credentialLibraryVaultSshCertificateCriticalOptionsKey); ok {
		if vv, ok := v.(map[string]any); ok {
			co := make(map[string]string, len(vv))
			for k, vvv := range vv {
				co[k] = vvv.(string)
			}
			opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryCriticalOptions(co))
		}
	}
	if v, ok := d.GetOk(credentialLibraryVaultSshCertificateExtensionsKey); ok {
		if vv, ok := v.(map[string]any); ok {
			e := make(map[string]string, len(vv))
			for k, vvv := range vv {
				e[k] = vvv.(string)
			}
			opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryExtensions(e))
		}
	}
	if v, ok := d.GetOk(credentialLibraryVaultSshCertificateAdditionalValidPrincipalsKey); ok {
		avp := []string{}
		for _, vv := range v.([]interface{}) {
			avp = append(avp, vv.(string))
		}
		opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryAdditionalValidPrincipals(avp))
	}

	var credentialStoreId string
	cid, ok := d.GetOk(credentialStoreIdKey)
	if ok {
		credentialStoreId = cid.(string)
	} else {
		return diag.Errorf("no credential store ID is set")
	}

	client := credentiallibraries.NewClient(md.client)
	cr, err := client.Create(ctx, credentialLibraryVaultSshCertificateType, credentialStoreId, opts...)
	if err != nil {
		return diag.Errorf("error creating credential library: %v", err)
	}
	if cr == nil {
		return diag.Errorf("nil credential library after create")
	}

	if err := setFromVaultSshCertificateCredentialLibraryResponseMap(d, cr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceCredentialLibraryReadVaultSshCertificate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	if err := setFromVaultSshCertificateCredentialLibraryResponseMap(d, cr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceCredentialLibraryUpdateVaultSshCertificate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
	if d.HasChange(credentialLibraryVaultSshCertificatePathKey) {
		v, ok := d.GetOk(credentialLibraryVaultSshCertificatePathKey)
		if ok {
			opts = append(opts, credentiallibraries.WithVaultCredentialLibraryPath(v.(string)))
		}
	}
	if d.HasChange(credentialLibraryVaultSshCertificateUsernameKey) {
		v, ok := d.GetOk(credentialLibraryVaultSshCertificateUsernameKey)
		if ok {
			opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryUsername(v.(string)))
		}
	}
	if d.HasChange(credentialLibraryVaultSshCertificateKeyTypeKey) {
		v, ok := d.GetOk(credentialLibraryVaultSshCertificateKeyTypeKey)
		if ok {
			opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryKeyType(v.(string)))
		}
	}
	if d.HasChange(credentialLibraryVaultSshCertificateKeyBitsKey) {
		v, ok := d.GetOk(credentialLibraryVaultSshCertificateKeyBitsKey)
		if ok {
			opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryKeyBits(uint32(v.(int))))
		}
	}
	if d.HasChange(credentialLibraryVaultSshCertificateTtlKey) {
		v, ok := d.GetOk(credentialLibraryVaultSshCertificateTtlKey)
		if ok {
			opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryTtl(v.(string)))
		}
	}
	if d.HasChange(credentialLibraryVaultSshCertificateKeyIdKey) {
		v, ok := d.GetOk(credentialLibraryVaultSshCertificateKeyIdKey)
		if ok {
			opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryKeyId(v.(string)))
		}
	}
	if d.HasChange(credentialLibraryVaultSshCertificateCriticalOptionsKey) {
		var oldCriticalOptions, newCriticalOptions map[string]interface{}
		old, new := d.GetChange(credentialLibraryVaultSshCertificateCriticalOptionsKey)

		if old == nil {
			old = map[string]interface{}{}
		}
		oldCriticalOptions = old.(map[string]interface{})

		if new == nil {
			new = map[string]interface{}{}
		}
		newCriticalOptions = new.(map[string]interface{})

		newAttrs := []string{}
		for k := range newCriticalOptions {
			newAttrs = append(newAttrs, k)
		}

		for oldAttr := range oldCriticalOptions {
			isRemoved := true
			for _, newAttr := range newAttrs {
				if oldAttr == newAttr {
					isRemoved = false
					break
				}
			}
			if isRemoved {
				newCriticalOptions[oldAttr] = nil
			}
		}

		co := make(map[string]string, 0)
		for k, v := range newCriticalOptions {
			if v != nil {
				co[k] = v.(string)
			}
		}
		if len(co) == 0 {
			co = nil
		}
		opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryCriticalOptions(co))

	}
	if d.HasChange(credentialLibraryVaultSshCertificateExtensionsKey) {
		var oldExtensions, newExtensions map[string]interface{}
		old, new := d.GetChange(credentialLibraryVaultSshCertificateExtensionsKey)

		if old == nil {
			old = map[string]interface{}{}
		}
		oldExtensions = old.(map[string]interface{})

		if new == nil {
			new = map[string]interface{}{}
		}
		newExtensions = new.(map[string]interface{})

		newAttrs := []string{}
		for k := range newExtensions {
			newAttrs = append(newAttrs, k)
		}

		for oldAttr := range oldExtensions {
			isRemoved := true
			for _, newAttr := range newAttrs {
				if oldAttr == newAttr {
					isRemoved = false
					break
				}
			}
			if isRemoved {
				newExtensions[oldAttr] = nil
			}
		}

		e := make(map[string]string, 0)
		for k, v := range newExtensions {
			if v != nil {
				e[k] = v.(string)
			}
		}
		if len(e) == 0 {
			e = nil
		}
		opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryExtensions(e))
	}
	if d.HasChange(credentialLibraryVaultSshCertificateAdditionalValidPrincipalsKey) {
		// set defaults first in case the value was omitted and we want to remove it
		opts = append(opts, credentiallibraries.DefaultVaultSSHCertificateCredentialLibraryAdditionalValidPrincipals())
		if v, ok := d.GetOk(credentialLibraryVaultSshCertificateAdditionalValidPrincipalsKey); ok {
			avp := []string{}
			for _, vv := range v.([]interface{}) {
				avp = append(avp, vv.(string))
			}
			opts = append(opts, credentiallibraries.WithVaultSSHCertificateCredentialLibraryAdditionalValidPrincipals(avp))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, credentiallibraries.WithAutomaticVersioning(true))
		aur, err := client.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating credential library: %v", err)
		}

		if err := setFromVaultSshCertificateCredentialLibraryResponseMap(d, aur.GetResponse().Map); err != nil {
			return diag.Errorf("error setting credential library from response: %v", err)
		}
	}

	return nil
}

func resourceCredentialLibraryDeleteVaultSshCertificate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentiallibraries.NewClient(md.client)

	_, err := client.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting credential library: %v", err)
	}

	return nil
}
