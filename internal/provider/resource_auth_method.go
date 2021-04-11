package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	authmethodAttributesKey = "attributes"
)

func resourceAuthMethod() *schema.Resource {
	return &schema.Resource{
		Description: "The auth method resource allows you to configure a Boundary auth_method.",

		CreateContext: resourceAuthMethodCreate,
		ReadContext:   resourceAuthMethodRead,
		UpdateContext: resourceAuthMethodUpdate,
		DeleteContext: resourceAuthMethodDelete,
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
				Description: "The resource type.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			authmethodAttributesKey: {
				Description: "Arbitrary attributes map for auth method configuration.",
				Type:        schema.TypeMap,
				// The elem isn't actually needed, and these values can be arbitrary so
				// leaving this unset
				//				Elem: &schema.Schema{
				//					Type: schema.TypeString,
				//				},
				Optional: true,
				Computed: true,
			},
			authmethodMinLoginNameLengthKey: {
				Description: "The minimum login name length.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Deprecated:  "Will be removed in favor of using attributes parameter",
			},
			authmethodMinPasswordLengthKey: {
				Description: "The minimum password length.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Deprecated:  "Will be removed in favor of using attributes parameter",
			},
		},
	}
}

func setFromAuthMethodResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(ScopeIdKey, raw["scope_id"])
	d.Set(TypeKey, raw["type"])

	if attrsVal, ok := raw["attributes"]; ok {
		// need to switch on type and convert from strings when neccessary
		d.Set(authmethodAttributesKey, attrsVal.(map[string]interface{}))
	}

	fmt.Printf("after set %+v\n", d.Get(authmethodAttributesKey))

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

	var typeStr string
	if typeVal, ok := d.GetOk(TypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}

	opts := []authmethods.Option{}

	if nameVal, ok := d.GetOk(NameKey); ok {
		opts = append(opts, authmethods.WithName(nameVal.(string)))
	}

	if descVal, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, authmethods.WithDescription(descVal.(string)))
	}

	if attrs, ok := d.GetOk(authmethodAttributesKey); ok {
		opts = append(opts, authmethods.WithAttributes(attrs.(map[string]interface{})))
	}

	// TODO(malnick) - deprecate
	if minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey); ok {
		opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(minLengthVal.(int))))
	}

	// TODO(malnick) - deprecate
	if minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey); ok {
		opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(minLengthVal.(int))))
	}

	amClient := authmethods.NewClient(md.client)

	amcr, err := amClient.Create(ctx, typeStr, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating auth method: %v", err)
	}
	if amcr == nil {
		return diag.Errorf("nil auth method after create")
	}

	setFromAuthMethodResponseMap(d, amcr.GetResponse().Map)

	return nil
}

func resourceAuthMethodRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	setFromAuthMethodResponseMap(d, amrr.GetResponse().Map)

	return nil
}

func resourceAuthMethodUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	opts := []authmethods.Option{}

	if d.HasChange(NameKey) {
		opts = append(opts, authmethods.DefaultName())
		if nameVal, ok := d.GetOk(NameKey); ok {
			opts = append(opts, authmethods.WithName(nameVal.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, authmethods.DefaultDescription())
		if descVal, ok := d.GetOk(DescriptionKey); ok {
			opts = append(opts, authmethods.WithDescription(descVal.(string)))
		}
	}

	// TODO(malnick) - when polling the value in this block we get 0
	// it does not appear that TF sees the changes to the map
	if d.HasChange(authmethodAttributesKey) {
		if attrs, ok := d.GetOk(authmethodMinLoginNameLengthKey); ok {
			opts = append(opts, authmethods.WithAttributes(attrs.(map[string]interface{})))
		}

	}

	// TODO(malnick) - deprecate
	if d.HasChange(authmethodMinPasswordLengthKey) {
		opts = append(opts, authmethods.DefaultPasswordAuthMethodMinPasswordLength())
		if minLengthVal, ok := d.GetOk(authmethodMinPasswordLengthKey); ok {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinPasswordLength(uint32(minLengthVal.(int))))
		}
	}
	// TODO(malnick) - deprecate
	if d.HasChange(authmethodMinLoginNameLengthKey) {
		opts = append(opts, authmethods.DefaultPasswordAuthMethodMinLoginNameLength())
		if minLengthVal, ok := d.GetOk(authmethodMinLoginNameLengthKey); ok {
			opts = append(opts, authmethods.WithPasswordAuthMethodMinLoginNameLength(uint32(minLengthVal.(int))))
		}
	}

	opts = append(opts, authmethods.WithAutomaticVersioning(true))
	amu, err := amClient.Update(ctx, d.Id(), 0, opts...)
	if err != nil {
		return diag.Errorf("error updating auth method: %v", err)
	}

	setFromAuthMethodResponseMap(d, amu.GetResponse().Map)

	return nil
}

func resourceAuthMethodDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	_, err := amClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting auth method: %v", err)
	}

	return nil
}
