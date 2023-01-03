// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	credentialUsernamePasswordUsernameKey     = "username"
	credentialUsernamePasswordPasswordKey     = "password"
	credentialUsernamePasswordPasswordHmacKey = "password_hmac"
	credentialUsernamePasswordCredentialType  = "username_password"
)

func resourceCredentialUsernamePassword() *schema.Resource {
	return &schema.Resource{
		Description: "The username/password credential resource allows you to configure a credential using a username and password pair.",

		CreateContext: resourceCredentialUsernamePasswordCreate,
		ReadContext:   resourceCredentialUsernamePasswordRead,
		UpdateContext: resourceCredentialUsernamePasswordUpdate,
		DeleteContext: resourceCredentialUsernamePasswordDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of this username/password credential.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The name of this username/password credential. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The description of this username/password credential.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreIdKey: {
				Description: "The credential store in which to save this username/password credential.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			credentialUsernamePasswordUsernameKey: {
				Description: "The username of this username/password credential.",
				Type:        schema.TypeString,
				Required:    true,
			},
			credentialUsernamePasswordPasswordKey: {
				Description: "The password of this username/password credential.",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			credentialUsernamePasswordPasswordHmacKey: {
				Description: "The password hmac.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func setFromCredentialUsernamePasswordResponseMap(d *schema.ResourceData, raw map[string]interface{}, fromRead bool) error {
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
		if err := d.Set(credentialUsernamePasswordUsernameKey, attrs[credentialUsernamePasswordUsernameKey]); err != nil {
			return err
		}

		statePasswordHmac := d.Get(credentialUsernamePasswordPasswordHmacKey)
		boundaryPasswordHmac := attrs[credentialUsernamePasswordPasswordHmacKey].(string)
		if statePasswordHmac.(string) != boundaryPasswordHmac && fromRead {
			// PasswordHmac has changed in Boundary, therefore the password has changed.
			// Update password value to force tf to attempt update.
			if err := d.Set(credentialUsernamePasswordPasswordKey, "(changed in Boundary)"); err != nil {
				return err
			}
		}
		if err := d.Set(credentialUsernamePasswordPasswordHmacKey, boundaryPasswordHmac); err != nil {
			return err
		}
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceCredentialUsernamePasswordCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var opts []credentials.Option
	if v, ok := d.GetOk(NameKey); ok {
		opts = append(opts, credentials.WithName(v.(string)))
	}
	if v, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, credentials.WithDescription(v.(string)))
	}
	if v, ok := d.GetOk(credentialUsernamePasswordUsernameKey); ok {
		opts = append(opts, credentials.WithUsernamePasswordCredentialUsername(v.(string)))
	}
	if v, ok := d.GetOk(credentialUsernamePasswordPasswordKey); ok {
		opts = append(opts, credentials.WithUsernamePasswordCredentialPassword(v.(string)))
	}

	var credentialStoreId string
	retrievedStoreId, ok := d.GetOk(credentialStoreIdKey)
	if ok {
		credentialStoreId = retrievedStoreId.(string)
	} else {
		return diag.Errorf("credential store id is unset")
	}

	client := credentials.NewClient(md.client)
	cr, err := client.Create(ctx, credentialUsernamePasswordCredentialType, credentialStoreId, opts...)
	if err != nil {
		return diag.Errorf("error creating credential: %v", err)
	}
	if cr == nil {
		return diag.Errorf("nil credential after create")
	}

	if err := setFromCredentialUsernamePasswordResponseMap(d, cr.GetResponse().Map, false); err != nil {
		return diag.Errorf("error generating credential from response map: %v", err)
	}

	return nil
}

func resourceCredentialUsernamePasswordRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	if err := setFromCredentialUsernamePasswordResponseMap(d, cr.GetResponse().Map, true); err != nil {
		return diag.Errorf("error generating credential from response map: %v", err)
	}

	return nil
}

func resourceCredentialUsernamePasswordUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	if d.HasChange(credentialUsernamePasswordUsernameKey) {
		usernameVal, ok := d.GetOk(credentialUsernamePasswordUsernameKey)
		if ok {
			opts = append(opts, credentials.WithUsernamePasswordCredentialUsername(usernameVal.(string)))
		}
	}

	if d.HasChange(credentialUsernamePasswordPasswordKey) {
		passwordVal, ok := d.GetOk(credentialUsernamePasswordPasswordKey)
		if ok {
			opts = append(opts, credentials.WithUsernamePasswordCredentialPassword(passwordVal.(string)))
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

		if err = setFromCredentialUsernamePasswordResponseMap(d, crUpdate.GetResponse().Map, false); err != nil {
			return diag.Errorf("error generating credential from response map: %v", err)
		}
	}

	return nil
}

func resourceCredentialUsernamePasswordDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentials.NewClient(md.client)

	_, err := client.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting credential: %v", err)
	}

	return nil
}
