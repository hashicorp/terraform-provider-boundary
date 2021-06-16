package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentialstores"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	credentialStoreVaultAddressKey                  = "address"
	credentialStoreVaultNamespaceKey                = "namespace"
	credentialStoreVaultCaCertKey                   = "ca_cert"
	credentialStoreVaultTlsServerNameKey            = "tls_server_name"
	credentialStoreVaultTlsSkipVerifyKey            = "tls_skip_verify"
	credentialStoreVaultTokenKey                    = "token"
	credentialStoreVaultTokenHmacKey                = "token_hmac"
	credentialStoreVaultClientCertificateKey        = "client_certificate"
	credentialStoreVaultClientCertificateKeyKey     = "client_certificate_key"
	credentialStoreVaultClientCertificateKeyHmacKey = "client_certificate_key_hmac"
	credentialStoreType                             = "vault"
)

var storeVaultAttrs = []string{
	credentialStoreVaultAddressKey,
	credentialStoreVaultNamespaceKey,
	credentialStoreVaultCaCertKey,
	credentialStoreVaultTlsServerNameKey,
	credentialStoreVaultTlsSkipVerifyKey,
	credentialStoreVaultTokenHmacKey,
	credentialStoreVaultClientCertificateKey,
	credentialStoreVaultClientCertificateKeyHmacKey,
}

func resourceCredentialStoreVault() *schema.Resource {
	return &schema.Resource{
		Description: "The credential store for Vault resource allows you to configure a Boundary credential store for Vault.",

		CreateContext: resourceCredentialStoreVaultCreate,
		ReadContext:   resourceCredentialStoreVaultRead,
		UpdateContext: resourceCredentialStoreVaultUpdate,
		DeleteContext: resourceCredentialStoreVaultDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the Vault credential store.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The Vault credential store name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The Vault credential store description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope for this credential store",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			credentialStoreVaultAddressKey: {
				Description: "The address to Vault server",
				Type:        schema.TypeString,
				Required:    true,
			},
			credentialStoreVaultNamespaceKey: {
				Description: "The namespace within Vault to use",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreVaultCaCertKey: {
				Description: "The Vault CA certificate to use",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreVaultTlsServerNameKey: {
				Description: "The Vault TLS server name",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreVaultTlsSkipVerifyKey: {
				Description: "Whether or not to skip TLS verification",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			credentialStoreVaultTokenKey: {
				Description: "The Vault token",
				Type:        schema.TypeString,
				Required:    true,
			},
			credentialStoreVaultTokenHmacKey: {
				Description: "The Vault token hmac",
				Type:        schema.TypeString,
				Computed:    true,
			},
			credentialStoreVaultClientCertificateKey: {
				Description: "The Vault client certificate",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreVaultClientCertificateKeyKey: {
				Description: "The Vault client certificate key",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreVaultClientCertificateKeyHmacKey: {
				Description: "The Vault client certificate key hmac",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func setFromVaultCredentialStoreResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw[NameKey]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw[DescriptionKey]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw[ScopeIdKey]); err != nil {
		return err
	}

	if attrsVal, ok := raw["attributes"]; ok {
		attrs := attrsVal.(map[string]interface{})
		for _, v := range storeVaultAttrs {
			if err := d.Set(v, attrs[v]); err != nil {
				return err
			}
		}
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceCredentialStoreVaultCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var opts []credentialstores.Option
	if v, ok := d.GetOk(NameKey); ok {
		opts = append(opts, credentialstores.WithName(v.(string)))
	}
	if v, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, credentialstores.WithDescription(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultAddressKey); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreAddress(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultNamespaceKey); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreNamespace(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultCaCertKey); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreCaCert(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultTlsServerNameKey); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreTlsServerName(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultTlsSkipVerifyKey); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreTlsSkipVerify(v.(bool)))
	}
	if v, ok := d.GetOk(credentialStoreVaultClientCertificateKey); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreClientCertificate(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultClientCertificateKeyKey); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreClientCertificateKey(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultTokenKey); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreToken(v.(string)))
	}

	var scope string
	gotScope, ok := d.GetOk(ScopeIdKey)
	if ok {
		scope = gotScope.(string)
	} else {
		return diag.Errorf("no scope is set")
	}

	client := credentialstores.NewClient(md.client)
	cr, err := client.Create(ctx, credentialStoreType, scope, opts...)
	if err != nil {
		return diag.Errorf("error creating credential store: %v", err)
	}
	if cr == nil {
		return diag.Errorf("nil credential store after create")
	}

	if err := setFromVaultCredentialStoreResponseMap(d, cr.GetResponse().Map); err != nil {
		return diag.Errorf("error generating credential store from response map: %v", err)
	}

	return nil
}

func resourceCredentialStoreVaultRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentialstores.NewClient(md.client)

	cr, err := client.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading credential store: %v", err)
	}
	if cr == nil {
		return diag.Errorf("credential store nil after read")
	}

	if err := setFromVaultCredentialStoreResponseMap(d, cr.GetResponse().Map); err != nil {
		return diag.Errorf("error generating credential store from response map: %v", err)
	}

	return nil
}

func resourceCredentialStoreVaultUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentialstores.NewClient(md.client)

	var opts []credentialstores.Option
	if d.HasChange(NameKey) {
		opts = append(opts, credentialstores.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			opts = append(opts, credentialstores.WithName(nameVal.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, credentialstores.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			opts = append(opts, credentialstores.WithDescription(descVal.(string)))
		}
	}

	if d.HasChange(credentialStoreVaultAddressKey) {
		opts = append(opts, credentialstores.DefaultVaultCredentialStoreAddress())
		v, ok := d.GetOk(credentialStoreVaultAddressKey)
		if ok {
			opts = append(opts, credentialstores.WithVaultCredentialStoreAddress(v.(string)))
		}
	}

	if d.HasChange(credentialStoreVaultNamespaceKey) {
		opts = append(opts, credentialstores.DefaultVaultCredentialStoreNamespace())
		v, ok := d.GetOk(credentialStoreVaultNamespaceKey)
		if ok {
			opts = append(opts, credentialstores.WithVaultCredentialStoreNamespace(v.(string)))
		}
	}

	if d.HasChange(credentialStoreVaultCaCertKey) {
		opts = append(opts, credentialstores.DefaultVaultCredentialStoreCaCert())
		v, ok := d.GetOk(credentialStoreVaultCaCertKey)
		if ok {
			opts = append(opts, credentialstores.WithVaultCredentialStoreCaCert(v.(string)))
		}
	}

	if d.HasChange(credentialStoreVaultTlsServerNameKey) {
		opts = append(opts, credentialstores.DefaultVaultCredentialStoreTlsServerName())
		v, ok := d.GetOk(credentialStoreVaultTlsServerNameKey)
		if ok {
			opts = append(opts, credentialstores.WithVaultCredentialStoreTlsServerName(v.(string)))
		}
	}

	if d.HasChange(credentialStoreVaultTlsSkipVerifyKey) {
		opts = append(opts, credentialstores.DefaultVaultCredentialStoreTlsSkipVerify())
		v, ok := d.GetOk(credentialStoreVaultTlsSkipVerifyKey)
		if ok {
			opts = append(opts, credentialstores.WithVaultCredentialStoreTlsSkipVerify(v.(bool)))
		}
	}

	if d.HasChange(credentialStoreVaultTokenKey) {
		opts = append(opts, credentialstores.DefaultVaultCredentialStoreToken())
		v, ok := d.GetOk(credentialStoreVaultTokenKey)
		if ok {
			opts = append(opts, credentialstores.WithVaultCredentialStoreToken(v.(string)))
		}
	}

	if d.HasChange(credentialStoreVaultClientCertificateKey) {
		opts = append(opts, credentialstores.DefaultVaultCredentialStoreClientCertificate())
		v, ok := d.GetOk(credentialStoreVaultClientCertificateKey)
		if ok {
			opts = append(opts, credentialstores.WithVaultCredentialStoreClientCertificate(v.(string)))
		}
	}

	if d.HasChange(credentialStoreVaultClientCertificateKeyKey) {
		opts = append(opts, credentialstores.DefaultVaultCredentialStoreClientCertificateKey())
		v, ok := d.GetOk(credentialStoreVaultClientCertificateKeyKey)
		if ok {
			opts = append(opts, credentialstores.WithVaultCredentialStoreClientCertificateKey(v.(string)))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, credentialstores.WithAutomaticVersioning(true))
		aur, err := client.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating credential store: %v", err)
		}
		if aur == nil {
			return diag.Errorf("credential store nil after update")
		}

		if err = setFromVaultCredentialStoreResponseMap(d, aur.GetResponse().Map); err != nil {
			return diag.Errorf("error generating credential store from response map: %v", err)
		}
	}

	return nil
}

func resourceCredentialStoreVaultDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentialstores.NewClient(md.client)

	_, err := client.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting credential store: %v", err)
	}

	return nil
}
