package provider

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	authmethodTypePassword          = "password"
	authmethodMinLoginNameLengthKey = "min_login_name_length"
	authmethodMinPasswordLengthKey  = "min_password_length"
)

func resourceAuthMethodPassword() *schema.Resource {
	return &schema.Resource{
		Description: "The auth method resource allows you to configure a Boundary auth_method_password.",

		CreateContext: resourceAuthMethodPasswordCreate,
		ReadContext:   resourceAuthMethodPasswordRead,
		UpdateContext: resourceAuthMethodPasswordUpdate,
		DeleteContext: resourceAuthMethodPasswordDelete,
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
				Description: "The auth method name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The auth method description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			TypeKey: {
				Description: "The resource type, hardcoded per resource",
				Type:        schema.TypeString,
				Default:     authmethodTypePassword,
				Optional:    true,
			},
			authmethodMinLoginNameLengthKey: {
				Description: "The minimum login name length.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			authmethodMinPasswordLengthKey: {
				Description: "The minimum password length.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func setFromPasswordAuthMethodResponseMap(d *schema.ResourceData, raw map[string]interface{}) diag.Diagnostics {
	d.Set(NameKey, raw[NameKey])
	d.Set(DescriptionKey, raw[DescriptionKey])
	d.Set(ScopeIdKey, raw[ScopeIdKey])
	d.Set(TypeKey, raw[TypeKey])

	if attrsVal, ok := raw["attributes"]; ok {
		attrs := attrsVal.(map[string]interface{})

		minLoginNameLength := attrs[authmethodMinLoginNameLengthKey].(json.Number)
		minLoginNameLengthInt, _ := minLoginNameLength.Int64()
		d.Set(authmethodMinLoginNameLengthKey, int(minLoginNameLengthInt))

		minPasswordLength := attrs[authmethodMinPasswordLengthKey].(json.Number)
		minPasswordLengthInt, _ := minPasswordLength.Int64()
		d.Set(authmethodMinPasswordLengthKey, int(minPasswordLengthInt))
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceAuthMethodPasswordCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var typeStr string
	if typeVal, ok := d.GetOk(TypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}

	opts := []authmethods.Option{}

	var minLoginNameLength *int
	if minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey); ok {
		minLength := minLengthVal.(int)
		minLoginNameLength = &minLength
	}
	if minLoginNameLength != nil {
		opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(*minLoginNameLength)))
	}

	var minPasswordLength *int
	if minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey); ok {
		minLength := minLengthVal.(int)
		minPasswordLength = &minLength
	}
	if minPasswordLength != nil {
		opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(*minPasswordLength)))
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

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	amClient := authmethods.NewClient(md.client)

	amcr, err := amClient.Create(ctx, typeStr, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating auth method: %v", err)
	}
	if amcr == nil {
		return diag.Errorf("nil auth method after create")
	}

	return setFromPasswordAuthMethodResponseMap(d, amcr.GetResponse().Map)
}

func resourceAuthMethodPasswordRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	amrr, err := amClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading auth method: %v", err)
	}
	if amrr == nil {
		return diag.Errorf("auth method nil after read")
	}

	return setFromPasswordAuthMethodResponseMap(d, amrr.GetResponse().Map)
}

func resourceAuthMethodPasswordUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	opts := []authmethods.Option{}

	if d.HasChange(NameKey) {
		opts = append(opts, authmethods.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			opts = append(opts, authmethods.WithName(nameVal.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, authmethods.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			opts = append(opts, authmethods.WithDescription(descVal.(string)))
		}
	}

	if d.HasChange(authmethodMinLoginNameLengthKey) {
		opts = append(opts, authmethods.DefaultPasswordAuthMethodMinLoginNameLength())
		minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey)
		if ok {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(minLengthVal.(int))))
		}
	}

	if d.HasChange(authmethodMinPasswordLengthKey) {
		opts = append(opts, authmethods.DefaultPasswordAuthMethodMinPasswordLength())
		minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey)
		if ok {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(minLengthVal.(int))))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, authmethods.WithAutomaticVersioning(true))
		amur, err := amClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating auth method: %v", err)
		}

		return setFromPasswordAuthMethodResponseMap(d, amur.GetResponse().Map)
	}
	return nil
}

func resourceAuthMethodPasswordDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	_, err := amClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting auth method: %v", err)
	}

	return nil
}
