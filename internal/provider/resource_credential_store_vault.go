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
	credentialStoreVaultAddress                  = "address"
	credentialStoreVaultNamespace                = "namespace"
	credentialStoreVaultCaCert                   = "vault_ca_cert"
	credentialStoreVaultTlsServerName            = "tls_server_name"
	credentialStoreVaultTlsSkipVerify            = "tls_skip_verify"
	credentialStoreVaultToken                    = "vault_token"
	credentialStoreVaultTokenHmac                = "vault_token_hmac"
	credentialStoreVaultClientCertificate        = "client_certificate"
	credentialStoreVaultClientCertificateKey     = "client_certificate_key"
	credentialStoreVaultClientCertificateKeyHmac = "client_certificate_key_hmac"
	credentialStoreVaultScope                    = "scope"
	credentialStoreType                          = "vault"
)

var storeVaultAttrs = []string{
	credentialStoreVaultScope,
	credentialStoreVaultAddress,
	credentialStoreVaultNamespace,
	credentialStoreVaultCaCert,
	credentialStoreVaultTlsServerName,
	credentialStoreVaultTlsSkipVerify,
	credentialStoreVaultToken,
	credentialStoreVaultTokenHmac,
	credentialStoreVaultClientCertificate,
	credentialStoreVaultClientCertificateKey,
	credentialStoreVaultClientCertificateKeyHmac}

func resourceCredentialStoreVault() *schema.Resource {
	return &schema.Resource{
		Description: "The credential store for Vault resource allows you to configure a Boundary credential library for Vault.",

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
			credentialStoreVaultScope: {
				Description: "The scope for this credential store",
				Type:        schema.TypeString,
				Required:    true,
			},
			credentialStoreVaultAddress: {
				Description: "The address to Vault server",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreVaultNamespace: {
				Description: "The namespace within Vault to use",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreVaultCaCert: {
				Description: "The Vault CA certificate to use",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreVaultTlsServerName: {
				Description: "The Vault TLS server name",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreVaultTlsSkipVerify: {
				Description: "Whether or not to skip TLS verification",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			credentialStoreVaultToken: {
				Description: "The Vault token",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreVaultTokenHmac: {
				Description: "The Vault token HMAC",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreVaultClientCertificate: {
				Description: "The Vault client certificate",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreVaultClientCertificateKey: {
				Description: "The Vault client certificate key",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreVaultClientCertificateKeyHmac: {
				Description: "The Vault client certificate key HMAC",
				Type:        schema.TypeString,
				Optional:    true,
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

	for _, v := range storeVaultAttrs {
		if err := d.Set(v, raw[v]); err != nil {
			return err
		}
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceCredentialStoreVaultCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	opts := []credentialstores.Option{}

	if v, ok := d.GetOk(NameKey); ok {
		opts = append(opts, credentialstores.WithName(v.(string)))
	}
	if v, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, credentialstores.WithDescription(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultAddress); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreAddress(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultClientCertificate); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreClientCertificate(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultClientCertificateKey); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreClientCertificateKey(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultNamespace); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreNamespace(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultTlsServerName); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreTlsServerName(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultTlsSkipVerify); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreTlsSkipVerify(v.(bool)))
	}
	if v, ok := d.GetOk(credentialStoreVaultCaCert); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreVaultCaCert(v.(string)))
	}
	if v, ok := d.GetOk(credentialStoreVaultToken); ok {
		opts = append(opts, credentialstores.WithVaultCredentialStoreVaultToken(v.(string)))
	}

	var scope string
	gotScope, ok := d.GetOk(credentialStoreIdKey)
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
		return diag.FromErr(err)
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
		return diag.FromErr(err)
	}

	return nil
}

func resourceCredentialStoreVaultUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentialstores.NewClient(md.client)

	opts := []credentialstores.Option{}

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

	if d.HasChange(credentialStoreVaultAddress) {
		opts = append(opts, credentialstores.DefaultVaultCredentialStoreAddress())
		v, ok := d.GetOk(credentialStoreVaultAddress)
		if ok {
			opts = append(opts, credentialstores.WithVaultCredentialStoreAddress(v.(string)))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, credentialstores.WithAutomaticVersioning(true))
		aur, err := client.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating credential store: %v", err)
		}

		setFromVaultCredentialStoreResponseMap(d, aur.GetResponse().Map)
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
