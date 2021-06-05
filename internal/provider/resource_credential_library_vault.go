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
	credentialStoreIdKey                     = "credential_store_id"
	credentialLibraryVaultHttpMethodKey      = "vault_http_method"
	credentialLibraryVaultHttpRequestBodyKey = "vault_http_request_body"
	credentialLibraryVaultPathKey            = "vault_path"
)

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
			//TODO (malnick) do we need type here? if so what value?
			TypeKey: {
				Description: "The resource type.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			credentialStoreIdKey: {
				Description: "The ID of the credential store that this library belongs to.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			credentialLibraryVaultHttpMethodKey: {
				Description: "The HTTP method to use when contacting Vault",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialLibraryVaultHttpRequestBodyKey: {
				Description: "The raw string to use in HTTP request to Vault",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialLibraryVaultPathKey: {
				Description: "The Vault path to query",
				Type:        schema.TypeString,
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
	if err := d.Set(TypeKey, raw[TypeKey]); err != nil {
		return err
	}
	if err := d.Set(credentialStoreIdKey, raw[credentialStoreIdKey]); err != nil {
		return err
	}
	if err := d.Set(credentialLibraryVaultHttpMethodKey, raw[credentialLibraryVaultHttpMethodKey]); err != nil {
		return err
	}
	if err := d.Set(credentialLibraryVaultHttpRequestBodyKey, raw[credentialLibraryVaultHttpRequestBodyKey]); err != nil {
		return err
	}
	if err := d.Set(credentialLibraryVaultPathKey, raw[credentialLibraryVaultPathKey]); err != nil {
		return err
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceCredentialLibraryCreateVault(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	opts := []credentiallibraries.Option{}

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
		opts = append(opts, credentiallibraries.WithVaultCredentialLibraryVaultPath(v.(string)))
	}

	var credentialstoreid string
	cid, ok := d.GetOk(credentialStoreIdKey)
	if ok {
		credentialstoreid = cid.(string)
	} else {
		return diag.Errorf("no credential store ID is set")
	}

	client := credentiallibraries.NewClient(md.client)

	cr, err := client.Create(ctx, credentialstoreid, opts...)
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

	opts := []credentiallibraries.Option{}

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
		opts = append(opts, credentiallibraries.DefaultVaultCredentialLibraryVaultPath())
		v, ok := d.GetOk(credentialLibraryVaultPathKey)
		if ok {
			opts = append(opts, credentiallibraries.WithVaultCredentialLibraryVaultPath(v.(string)))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, credentiallibraries.WithAutomaticVersioning(true))
		aur, err := client.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating credential library: %v", err)
		}

		setFromVaultCredentialLibraryResponseMap(d, aur.GetResponse().Map)
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
