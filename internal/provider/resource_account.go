package provider

import (
	"context"

	"github.com/hashicorp/boundary/api/accounts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	accountTypePassword = "password"
	accountLoginNameKey = "login_name"
	accountPasswordKey  = "password"
)

func resourceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAccountCreate,
		ReadContext:   resourceAccountRead,
		UpdateContext: resourceAccountUpdate,
		DeleteContext: resourceAccountDelete,
		Schema: map[string]*schema.Schema{
			NameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			DescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			AuthMethodIdKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			TypeKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			accountLoginNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			accountPasswordKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func setFromAccountResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
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

func resourceAccountCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	acr, apiErr, err := aClient.Create(
		ctx,
		authMethodId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling create account: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating account: %s", apiErr.Message)
	}
	if acr == nil {
		return diag.Errorf("nil account after create")
	}

	setFromAccountResponseMap(d, acr.GetResponseMap())

	return nil
}

func resourceAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	aClient := accounts.NewClient(md.client)

	arr, apiErr, err := aClient.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling read account: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading account: %s", apiErr.Message)
	}
	if arr == nil {
		return diag.Errorf("account nil after read")
	}

	setFromAccountResponseMap(d, arr.GetResponseMap())

	return nil
}

func resourceAccountUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	var password *string
	if d.HasChange(accountPasswordKey) {
		switch d.Get(TypeKey).(string) {
		case accountTypePassword:
			opts = append(opts, accounts.DefaultPasswordAccountPassword())
			keyVal, ok := d.GetOk(accountPasswordKey)
			if ok {
				keyStr := keyVal.(string)
				password = &keyStr
				opts = append(opts, accounts.WithPasswordAccountPassword(keyStr))
			}
		default:
			return diag.Errorf(`"password" cannot be used with this type of account`)
		}
	}

	if len(opts) > 0 {
		opts = append(opts, accounts.WithAutomaticVersioning(true))
		_, apiErr, err := aClient.Update(
			ctx,
			d.Id(),
			0,
			opts...)
		if err != nil {
			return diag.Errorf("error calling update account: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating account: %s", apiErr.Message)
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
	if d.HasChange(accountPasswordKey) {
		d.Set(accountPasswordKey, password)
	}

	return nil
}

func resourceAccountDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	aClient := accounts.NewClient(md.client)

	_, apiErr, err := aClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete account: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error deleting account: %s", apiErr.Message)
	}

	return nil
}
