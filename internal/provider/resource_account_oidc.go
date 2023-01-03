// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/accounts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	accountTypeOidc       = "password"
	accountOidcIssuerKey  = "issuer"
	accountOidcSubjectKey = "subject"
)

func resourceAccountOidc() *schema.Resource {
	return &schema.Resource{
		Description: "The account resource allows you to configure a Boundary account.",

		CreateContext: resourceAccountOidcCreate,
		ReadContext:   resourceAccountOidcRead,
		UpdateContext: resourceAccountOidcUpdate,
		DeleteContext: resourceAccountOidcDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the account.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The account name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The account description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			AuthMethodIdKey: {
				Description: "The resource ID for the auth method.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			accountOidcIssuerKey: {
				Description: "The OIDC issuer.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			accountOidcSubjectKey: {
				Description: "The OIDC subject.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
		},
	}
}

func setFromAccountOidcResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(AuthMethodIdKey, raw["auth_method_id"])
	if attrsVal, ok := raw["attributes"]; ok {
		attrs := attrsVal.(map[string]interface{})
		d.Set(accountOidcIssuerKey, attrs["issuer"])
		d.Set(accountOidcSubjectKey, attrs["subject"])
	}
	d.SetId(raw["id"].(string))
}

func resourceAccountOidcCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var authMethodId string
	if authMethodIdVal, ok := d.GetOk(AuthMethodIdKey); ok {
		authMethodId = authMethodIdVal.(string)
	} else {
		return diag.Errorf("no auth method ID provided")
	}

	opts := []accounts.Option{}

	if i, ok := d.GetOk(accountOidcIssuerKey); ok {
		opts = append(opts, accounts.WithOidcAccountIssuer(i.(string)))
	}

	if s, ok := d.GetOk(accountOidcSubjectKey); ok {
		opts = append(opts, accounts.WithOidcAccountSubject(s.(string)))
	}

	if n, ok := d.GetOk(NameKey); ok {
		opts = append(opts, accounts.WithName(n.(string)))
	}

	if d, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, accounts.WithDescription(d.(string)))
	}

	aClient := accounts.NewClient(md.client)

	acr, err := aClient.Create(ctx, authMethodId, opts...)
	if err != nil {
		return diag.Errorf("error creating account: %v", err)
	}
	if acr == nil {
		return diag.Errorf("nil account after create")
	}

	setFromAccountOidcResponseMap(d, acr.GetResponse().Map)

	return nil
}

func resourceAccountOidcRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	aClient := accounts.NewClient(md.client)

	arr, err := aClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading account: %v", err)
	}
	if arr == nil {
		return diag.Errorf("account nil after read")
	}

	setFromAccountOidcResponseMap(d, arr.GetResponse().Map)

	return nil
}

func resourceAccountOidcUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	aClient := accounts.NewClient(md.client)

	opts := []accounts.Option{}

	if d.HasChange(NameKey) {
		opts = append(opts, accounts.DefaultName())
		if n, ok := d.GetOk(NameKey); ok {
			opts = append(opts, accounts.WithName(n.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, accounts.DefaultDescription())
		if d, ok := d.GetOk(DescriptionKey); ok {
			opts = append(opts, accounts.WithDescription(d.(string)))
		}
	}

	if d.HasChange(accountOidcIssuerKey) {
		opts = append(opts, accounts.DefaultOidcAccountIssuer())
		if i, ok := d.GetOk(accountOidcIssuerKey); ok {
			opts = append(opts, accounts.WithOidcAccountIssuer(i.(string)))
		}
	}

	if d.HasChange(accountOidcSubjectKey) {
		opts = append(opts, accounts.DefaultOidcAccountSubject())
		if i, ok := d.GetOk(accountOidcSubjectKey); ok {
			opts = append(opts, accounts.WithOidcAccountSubject(i.(string)))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, accounts.WithAutomaticVersioning(true))
		aur, err := aClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating account: %v", err)
		}

		setFromAccountOidcResponseMap(d, aur.GetResponse().Map)
	}

	return nil
}

func resourceAccountOidcDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	aClient := accounts.NewClient(md.client)

	_, err := aClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting account: %v", err)
	}

	return nil
}
