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
	accountTypePassword = "password"
	accountLoginNameKey = "login_name"
	accountPasswordKey  = "password"
)

func resourceAccountPassword() *schema.Resource {
	return &schema.Resource{
		Description: "The account resource allows you to configure a Boundary account.",

		CreateContext: resourceAccountPasswordCreate,
		ReadContext:   resourceAccountPasswordRead,
		UpdateContext: resourceAccountPasswordUpdate,
		DeleteContext: resourceAccountPasswordDelete,
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
			TypeKey: {
				Description: "The resource type.",
				Type:        schema.TypeString,
				Deprecated:  "The value for this field will be infered since 'password' is the only possible value.",
				Default:     accountTypePassword,
				Optional:    true,
				ForceNew:    true,
			},
			accountLoginNameKey: {
				Description: "The login name for this account.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			accountPasswordKey: {
				Description: "The account password. Only set on create, changes will not be reflected when updating account.",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if d.Id() == "" {
						// This is a new resource do not suppress password diff
						return false
					}
					return true
				},
			},
		},
	}
}

func setFromAccountPasswordResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(AuthMethodIdKey, raw["auth_method_id"])
	d.Set(TypeKey, raw["type"])

	switch raw["type"].(string) {
	case accountTypePassword:
		if attrsVal, ok := raw["attributes"]; ok {
			attrs := attrsVal.(map[string]interface{})
			d.Set(accountLoginNameKey, attrs["login_name"])
		}
	}

	d.SetId(raw["id"].(string))
}

func resourceAccountPasswordCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var authMethodId string
	if authMethodIdVal, ok := d.GetOk(AuthMethodIdKey); ok {
		authMethodId = authMethodIdVal.(string)
	} else {
		return diag.Errorf("no auth method ID provided")
	}

	var loginName *string
	if keyVal, ok := d.GetOk(accountLoginNameKey); ok {
		key := keyVal.(string)
		loginName = &key
	}

	var password *string
	if keyVal, ok := d.GetOk(accountPasswordKey); ok {
		key := keyVal.(string)
		password = &key
	}

	opts := []accounts.Option{}

	var typeStr string
	if typeVal, ok := d.GetOk(TypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}
	switch typeStr {
	case accountTypePassword:
		if loginName != nil {
			opts = append(opts, accounts.WithPasswordAccountLoginName(*loginName))
		}
		if password != nil {
			opts = append(opts, accounts.WithPasswordAccountPassword(*password))
			d.Set(accountPasswordKey, *password)
		}
	default:
		return diag.Errorf("invalid type provided")
	}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, accounts.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, accounts.WithDescription(descStr))
	}

	aClient := accounts.NewClient(md.client)

	acr, err := aClient.Create(ctx, authMethodId, opts...)
	if err != nil {
		return diag.Errorf("error creating account: %v", err)
	}
	if acr == nil {
		return diag.Errorf("nil account after create")
	}

	setFromAccountPasswordResponseMap(d, acr.GetResponse().Map)

	return nil
}

func resourceAccountPasswordRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	setFromAccountPasswordResponseMap(d, arr.GetResponse().Map)

	return nil
}

func resourceAccountPasswordUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	aClient := accounts.NewClient(md.client)

	opts := []accounts.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, accounts.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, accounts.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, accounts.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, accounts.WithDescription(descStr))
		}
	}

	var loginName *string
	if d.HasChange(accountLoginNameKey) {
		switch d.Get(TypeKey).(string) {
		case accountTypePassword:
			opts = append(opts, accounts.DefaultPasswordAccountLoginName())
			keyVal, ok := d.GetOk(accountLoginNameKey)
			if ok {
				keyStr := keyVal.(string)
				loginName = &keyStr
				opts = append(opts, accounts.WithPasswordAccountLoginName(keyStr))
			}
		default:
			return diag.Errorf(`"login_name" cannot be used with this type of account`)
		}
	}

	if len(opts) > 0 {
		opts = append(opts, accounts.WithAutomaticVersioning(true))
		_, err := aClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating account: %v", err)
		}
	}

	if d.HasChange(NameKey) {
		d.Set(NameKey, name)
	}
	if d.HasChange(DescriptionKey) {
		d.Set(DescriptionKey, desc)
	}
	if d.HasChange(accountLoginNameKey) {
		d.Set(accountLoginNameKey, loginName)
	}

	return nil
}

func resourceAccountPasswordDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	aClient := accounts.NewClient(md.client)

	_, err := aClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting account: %v", err)
	}

	return nil
}
