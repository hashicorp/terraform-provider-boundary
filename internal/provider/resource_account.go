package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/accounts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAccount() *schema.Resource {
	return &schema.Resource{
		Description: "The account resource allows you to configure a Boundary account.",

		CreateContext: resourceAccountCreate,
		ReadContext:   resourceAccountRead,
		UpdateContext: resourceAccountUpdate,
		DeleteContext: resourceAccountDelete,
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
				Required:    true,
				ForceNew:    true,
			},
			accountLoginNameKey: {
				Description: "The login name for this account.",
				Type:        schema.TypeString,
				Optional:    true,
				Deprecated:  "Will be removed in favor of using attributes parameter",
			},
			accountPasswordKey: {
				Description: "The account password. Only set on create, changes will not be reflected when updating account.",
				Type:        schema.TypeString,
				Optional:    true,
				Deprecated:  "Will be removed in favor of using attributes parameter",

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

func setFromAccountResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(AuthMethodIdKey, raw["auth_method_id"])
	d.Set(TypeKey, raw["type"])

	// TODO(malnick) - remove after deprecation cycle in favor of attributes
	switch raw["type"].(string) {
	case accountTypePassword:
		if attrsVal, ok := raw["attributes"]; ok {
			attrs := attrsVal.(map[string]interface{})
			d.Set(accountLoginNameKey, attrs["login_name"])
		}
	}

	d.SetId(raw["id"].(string))
}

func resourceAccountCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var authMethodId string
	if authMethodIdVal, ok := d.GetOk(AuthMethodIdKey); ok {
		authMethodId = authMethodIdVal.(string)
	} else {
		return diag.Errorf("no auth method ID provided")
	}

	opts := []accounts.Option{}

	if nameVal, ok := d.GetOk(NameKey); ok {
		opts = append(opts, accounts.WithName(nameVal.(string)))
	}

	if descVal, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, accounts.WithDescription(descVal.(string)))
	}

	// TODO(malnick) - remove after deprecation cycle
	if name, ok := d.GetOk(accountLoginNameKey); ok {
		opts = append(opts, accounts.WithPasswordAccountLoginName(name.(string)))
	}

	// TODO(malnick) - remove after deprecation cycle
	if pass, ok := d.GetOk(accountPasswordKey); ok {
		opts = append(opts, accounts.WithPasswordAccountPassword(pass.(string)))
	}

	aClient := accounts.NewClient(md.client)

	acr, err := aClient.Create(ctx, authMethodId, opts...)
	if err != nil {
		return diag.Errorf("error creating account: %v", err)
	}
	if acr == nil {
		return diag.Errorf("nil account after create")
	}

	setFromAccountResponseMap(d, acr.GetResponse().Map)

	return nil
}

func resourceAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	setFromAccountResponseMap(d, arr.GetResponse().Map)

	return nil
}

func resourceAccountUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	aClient := accounts.NewClient(md.client)

	opts := []accounts.Option{}

	if d.HasChange(NameKey) {
		opts = append(opts, accounts.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			opts = append(opts, accounts.WithName(nameVal.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, accounts.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			opts = append(opts, accounts.WithDescription(descVal.(string)))
		}
	}

	// TODO(malnick) - remove after deprecation cycle
	if d.HasChange(accountLoginNameKey) {
		opts = append(opts, accounts.DefaultPasswordAccountLoginName())
		if keyVal, ok := d.GetOk(accountLoginNameKey); ok {
			opts = append(opts, accounts.WithPasswordAccountLoginName(keyVal.(string)))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, accounts.WithAutomaticVersioning(true))
		aur, err := aClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating account: %v", err)
		}

		setFromAccountResponseMap(d, aur.GetResponse().Map)
	}

	return nil
}

func resourceAccountDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	aClient := accounts.NewClient(md.client)

	_, err := aClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting account: %v", err)
	}

	return nil
}
