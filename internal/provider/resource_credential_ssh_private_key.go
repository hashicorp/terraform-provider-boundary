package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentials"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	credentialSshPrivateKeyUsernameKey       = "username"
	credentialSshPrivateKeyPrivateKeyKey     = "private_key"
	credentialSshPrivateKeyPrivateKeyHmacKey = "private_key_hmac"
	credentialSshPrivateKeyPassphraseKey     = "private_key_passphrase"
	credentialSshPrivateKeyPassphraseHmacKey = "private_key_passphrase_hmac"
	credentialSshPrivateKeyCredentialType    = "ssh_private_key"
)

func resourceCredentialSshPrivateKey() *schema.Resource {
	return &schema.Resource{
		Description: "The SSH private key credential resource allows you to configure a credential using a username, private key and optional passphrase.",

		CreateContext: resourceCredentialSshPrivateKeyCreate,
		ReadContext:   resourceCredentialSshPrivateKeyRead,
		UpdateContext: resourceCredentialSshPrivateKeyUpdate,
		DeleteContext: resourceCredentialSshPrivateKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the credential.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The name of the credential. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The description of the credential.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreIdKey: {
				Description: "ID of the credential store this credential belongs to.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			credentialSshPrivateKeyUsernameKey: {
				Description: "The username associated with the credential.",
				Type:        schema.TypeString,
				Required:    true,
			},
			credentialSshPrivateKeyPrivateKeyKey: {
				Description: "The private key associated with the credential.",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			credentialSshPrivateKeyPrivateKeyHmacKey: {
				Description: "The private key hmac.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			credentialSshPrivateKeyPassphraseKey: {
				Description: "The passphrase of the private key associated with the credential.",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
			},
			credentialSshPrivateKeyPassphraseHmacKey: {
				Description: "The private key passphrase hmac.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func setFromCredentialSshPrivateKeyResponseMap(d *schema.ResourceData, raw map[string]interface{}, fromRead bool) error {
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
		if err := d.Set(credentialSshPrivateKeyUsernameKey, attrs[credentialSshPrivateKeyUsernameKey]); err != nil {
			return err
		}

		statePrivKeyHmac := d.Get(credentialSshPrivateKeyPrivateKeyHmacKey)
		boundaryPrivKeyHmac := attrs[credentialSshPrivateKeyPrivateKeyHmacKey].(string)
		if statePrivKeyHmac.(string) != boundaryPrivKeyHmac && fromRead {
			// PrivateKeyHmac has changed in Boundary, therefore the private key has changed.
			// Update private key value to force tf to attempt update.
			if err := d.Set(credentialSshPrivateKeyPrivateKeyKey, "(changed in Boundary)"); err != nil {
				return err
			}
		}
		if err := d.Set(credentialSshPrivateKeyPrivateKeyHmacKey, boundaryPrivKeyHmac); err != nil {
			return err
		}

		var statePassphraseHmac, boundaryPassphraseHmac string
		if v := d.Get(credentialSshPrivateKeyPassphraseHmacKey); v != nil {
			statePassphraseHmac = v.(string)
		}
		if v := attrs[credentialSshPrivateKeyPassphraseHmacKey]; v != nil {
			boundaryPassphraseHmac = v.(string)
		}
		if statePassphraseHmac != boundaryPassphraseHmac && fromRead {
			// PassphraseHmac has changed in Boundary, therefore the private key passphrase has changed.
			// Update private key passphrase value to force tf to attempt update.
			if err := d.Set(credentialSshPrivateKeyPassphraseKey, "(changed in Boundary)"); err != nil {
				return err
			}
		}
		if err := d.Set(credentialSshPrivateKeyPassphraseHmacKey, boundaryPassphraseHmac); err != nil {
			return err
		}
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceCredentialSshPrivateKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var opts []credentials.Option
	if v, ok := d.GetOk(NameKey); ok {
		opts = append(opts, credentials.WithName(v.(string)))
	}
	if v, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, credentials.WithDescription(v.(string)))
	}
	if v, ok := d.GetOk(credentialSshPrivateKeyUsernameKey); ok {
		opts = append(opts, credentials.WithSshPrivateKeyCredentialUsername(v.(string)))
	}
	if v, ok := d.GetOk(credentialSshPrivateKeyPrivateKeyKey); ok {
		opts = append(opts, credentials.WithSshPrivateKeyCredentialPrivateKey(v.(string)))
	}
	if v, ok := d.GetOk(credentialSshPrivateKeyPassphraseKey); ok {
		opts = append(opts, credentials.WithSshPrivateKeyCredentialPrivateKeyPassphrase(v.(string)))
	}

	var credentialStoreId string
	retrievedStoreId, ok := d.GetOk(credentialStoreIdKey)
	if ok {
		credentialStoreId = retrievedStoreId.(string)
	} else {
		return diag.Errorf("credential store id is unset")
	}

	client := credentials.NewClient(md.client)
	cr, err := client.Create(ctx, credentialSshPrivateKeyCredentialType, credentialStoreId, opts...)
	if err != nil {
		return diag.Errorf("error creating credential: %v", err)
	}
	if cr == nil {
		return diag.Errorf("nil credential after create")
	}

	if err := setFromCredentialSshPrivateKeyResponseMap(d, cr.GetResponse().Map, false); err != nil {
		return diag.Errorf("error generating credential from response map: %v", err)
	}

	return nil
}

func resourceCredentialSshPrivateKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentials.NewClient(md.client)

	cr, err := client.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading credential: %v", err)
	}
	if cr == nil {
		return diag.Errorf("credential nil after read")
	}

	if err := setFromCredentialSshPrivateKeyResponseMap(d, cr.GetResponse().Map, true); err != nil {
		return diag.Errorf("error generating credential from response map: %v", err)
	}

	return nil
}

func resourceCredentialSshPrivateKeyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentials.NewClient(md.client)

	var opts []credentials.Option
	if d.HasChange(NameKey) {
		opts = append(opts, credentials.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			opts = append(opts, credentials.WithName(nameVal.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, credentials.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			opts = append(opts, credentials.WithDescription(descVal.(string)))
		}
	}

	if d.HasChange(credentialSshPrivateKeyUsernameKey) {
		usernameVal, ok := d.GetOk(credentialSshPrivateKeyUsernameKey)
		if ok {
			opts = append(opts, credentials.WithSshPrivateKeyCredentialUsername(usernameVal.(string)))
		}
	}

	if d.HasChange(credentialSshPrivateKeyPrivateKeyKey) {
		privKeyVal, ok := d.GetOk(credentialSshPrivateKeyPrivateKeyKey)
		if ok {
			opts = append(opts, credentials.WithSshPrivateKeyCredentialPrivateKey(privKeyVal.(string)))
		}
	}

	if d.HasChange(credentialSshPrivateKeyPassphraseKey) {
		passVal, ok := d.GetOk(credentialSshPrivateKeyPassphraseKey)
		if ok {
			opts = append(opts, credentials.WithSshPrivateKeyCredentialPrivateKeyPassphrase(passVal.(string)))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, credentials.WithAutomaticVersioning(true))
		crUpdate, err := client.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating credential: %v", err)
		}
		if crUpdate == nil {
			return diag.Errorf("credential nil after update")
		}

		if err = setFromCredentialSshPrivateKeyResponseMap(d, crUpdate.GetResponse().Map, false); err != nil {
			return diag.Errorf("error generating credential from response map: %v", err)
		}
	}

	return nil
}

func resourceCredentialSshPrivateKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentials.NewClient(md.client)

	_, err := client.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting credential: %v", err)
	}

	return nil
}
