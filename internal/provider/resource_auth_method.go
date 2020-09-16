package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	authmethodTypePassword          = "password"
	authmethodMinLoginNameLengthKey = "min_login_name_length"
	authmethodMinPasswordLengthKey  = "min_password_length"
)

func resourceAuthMethod() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAuthMethodCreate,
		ReadContext:   resourceAuthMethodRead,
		UpdateContext: resourceAuthMethodUpdate,
		DeleteContext: resourceAuthMethodDelete,
		Schema: map[string]*schema.Schema{
			NameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			DescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			ScopeIdKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			TypeKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			authmethodMinLoginNameLengthKey: {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			authmethodMinPasswordLengthKey: {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func setFromAuthMethodResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(ScopeIdKey, raw["scope_id"])
	d.Set(TypeKey, raw["type"])

	switch raw["type"].(string) {
	case authmethodTypePassword:
		if attrsVal, ok := raw["attributes"]; ok {
			attrs := attrsVal.(map[string]interface{})

			minLoginNameLength := attrs["min_login_name_length"].(json.Number)
			minLoginNameLengthInt, _ := minLoginNameLength.Int64()
			d.Set(authmethodMinLoginNameLengthKey, int(minLoginNameLengthInt))

			minPasswordLength := attrs["min_password_length"].(json.Number)
			minPasswordLengthInt, _ := minPasswordLength.Int64()
			d.Set(authmethodMinPasswordLengthKey, int(minPasswordLengthInt))
		}
	}

	d.SetId(raw["id"].(string))
}

func resourceAuthMethodCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	var minLoginNameLength *int
	if minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey); ok {
		minLength := minLengthVal.(int)
		minLoginNameLength = &minLength
	}

	var minPasswordLength *int
	if minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey); ok {
		minLength := minLengthVal.(int)
		minPasswordLength = &minLength
	}

	opts := []authmethods.Option{}

	var typeStr string
	if typeVal, ok := d.GetOk(TypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}
	switch typeStr {
	case authmethodTypePassword:
		if minLoginNameLength != nil {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(*minLoginNameLength)))
		}
		if minPasswordLength != nil {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(*minPasswordLength)))
		}
	default:
		return diag.Errorf("invalid type provided")
	}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, authmethods.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, authmethods.WithDescription(descStr))
	}

	amClient := authmethods.NewClient(md.client)

	amcr, apiErr, err := amClient.Create(
		ctx,
		typeStr,
		scopeId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling create auth method: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating auth method: %s", apiErr.Message)
	}
	if amcr == nil {
		return diag.Errorf("nil auth method after create")
	}

	setFromAuthMethodResponseMap(d, amcr.GetResponseMap())

	return nil
}

func resourceAuthMethodRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	amrr, apiErr, err := amClient.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling read auth method: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading auth method: %s", apiErr.Message)
	}
	if amrr == nil {
		return diag.Errorf("auth method nil after read")
	}

	setFromAuthMethodResponseMap(d, amrr.GetResponseMap())

	return nil
}

func resourceAuthMethodUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	opts := []authmethods.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, authmethods.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, authmethods.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, authmethods.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, authmethods.WithDescription(descStr))
		}
	}

	var minLoginNameLength *int
	if d.HasChange(authmethodMinLoginNameLengthKey) {
		switch d.Get(TypeKey).(string) {
		case authmethodTypePassword:
			opts = append(opts, authmethods.DefaultPasswordAuthMethodMinLoginNameLength())
			minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey)
			if ok {
				minLengthInt := minLengthVal.(int)
				minLoginNameLength = &minLengthInt
				opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(minLengthInt)))
			}
		default:
			return diag.Errorf(`"min_login_name_length" cannot be used with this type of auth method`)
		}
	}

	var minPasswordLength *int
	if d.HasChange(authmethodMinPasswordLengthKey) {
		switch d.Get(TypeKey).(string) {
		case authmethodTypePassword:
			opts = append(opts, authmethods.DefaultPasswordAuthMethodMinPasswordLength())
			minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey)
			if ok {
				minLengthInt := minLengthVal.(int)
				minPasswordLength = &minLengthInt
				opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(minLengthInt)))
			}
		default:
			return diag.Errorf(`"min_password_length" cannot be used with this type of auth method`)
		}
	}

	if len(opts) > 0 {
		opts = append(opts, authmethods.WithAutomaticVersioning(true))
		_, apiErr, err := amClient.Update(
			ctx,
			d.Id(),
			0,
			opts...)
		if err != nil {
			return diag.Errorf("error calling update auth method: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating auth method: %s", apiErr.Message)
		}
	}

	if d.HasChange(NameKey) {
		d.Set(NameKey, name)
	}
	if d.HasChange(DescriptionKey) {
		d.Set(DescriptionKey, desc)
	}
	if d.HasChange(authmethodMinLoginNameLengthKey) {
		d.Set(authmethodMinLoginNameLengthKey, minLoginNameLength)
	}
	if d.HasChange(authmethodMinPasswordLengthKey) {
		d.Set(authmethodMinPasswordLengthKey, minPasswordLength)
	}

	return nil
}

func resourceAuthMethodDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	_, apiErr, err := amClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete auth method: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error deleting auth method: %s", apiErr.Message)
	}

	return nil
}
