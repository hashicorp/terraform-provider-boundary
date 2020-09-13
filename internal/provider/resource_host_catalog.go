package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/boundary/api/hostcatalogs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	hostCatalogTypeKey    = "type"
	hostCatalogTypeStatic = "static"
)

func resourceHostCatalog() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceHostCatalogCreate,
		ReadContext:   resourceHostCatalogRead,
		UpdateContext: resourceHostCatalogUpdate,
		DeleteContext: resourceHostCatalogDelete,
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
			hostCatalogTypeKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}

}

func resourceHostCatalogCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	var typeStr string
	if typeVal, ok := d.GetOk(hostCatalogTypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}
	switch typeStr {
	case hostCatalogTypeStatic:
	default:
		return diag.Errorf("invalid type provided")
	}

	opts := []hostcatalogs.Option{}

	var name *string
	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		name = &nameStr
		opts = append(opts, hostcatalogs.WithName(nameStr))
	}

	var desc *string
	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		desc = &descStr
		opts = append(opts, hostcatalogs.WithDescription(descStr))
	}

	hcClient := hostcatalogs.NewClient(client)

	hc, apiErr, err := hcClient.Create(
		ctx,
		typeStr,
		scopeId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling new host catalog: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating host catalog: %s", apiErr.Message)
	}

	if name != nil {
		if err := d.Set(NameKey, name); err != nil {
			return diag.FromErr(err)
		}
	}

	if desc != nil {
		if err := d.Set(DescriptionKey, *desc); err != nil {
			return diag.FromErr(err)
		}
	}

	d.Set(hostCatalogTypeKey, hc.Type)
	d.SetId(hc.Id)

	return nil
}

func resourceHostCatalogRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	hcClient := hostcatalogs.NewClient(client)

	hc, apiErr, err := hcClient.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling new host catalog: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading host catalog: %s", apiErr.Message)
	}
	if hc == nil {
		return diag.Errorf("host catalog nil after read")
	}

	raw := hc.LastResponseMap()
	if raw == nil {
		return []diag.Diagnostic{
			{
				Severity: diag.Warning,
				Summary:  "response map empty after read",
			},
		}
	}

	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(ScopeIdKey, raw["scope_id"])
	d.Set(hostCatalogTypeKey, raw["type"])

	return nil
}

func resourceHostCatalogUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	h := convertResourceDataToHostCatalog(d)
	hcClient := hostcatalogs.NewClient(client)

	if d.HasChange(hostCatalogNameKey) {
		h.Name = d.Get(hostCatalogNameKey).(string)
	}

	if d.HasChange(hostCatalogDescriptionKey) {
		h.Description = d.Get(hostCatalogDescriptionKey).(string)
	}

	if d.HasChange(hostCatalogScopeIDKey) {
		h.Scope.Id = d.Get(hostCatalogScopeIDKey).(string)
	}

	if d.HasChange(hostCatalogTypeKey) {
		return errors.New("error updating host catalog: A host catalog can not have its type modified.")
	}

	// Type is a read-only value that can not be updated. It is added in the method to convert from a
	// resource to a HostCatalog type, so it needs to be unset when calling update.
	h.Type = ""
	h, apiErr, err := hcClient.Update(
		ctx,
		h.Id,
		0,
		hostcatalogs.WithScopeId(h.Scope.Id),
		hostcatalogs.WithAutomaticVersioning(),
		hostcatalogs.WithName(h.Name),
		hostcatalogs.WithDescription(h.Description))
	if err != nil {
		return err
	}
	if apiErr != nil {
		return fmt.Errorf("error updating host catalog: %s\n   Invalid request fields: %v\n", apiErr.Message, apiErr.Details.RequestFields)
	}

	return convertHostCatalogToResourceData(h, d)
}

func resourceHostCatalogDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	h := convertResourceDataToHostCatalog(d)
	hcClient := hostcatalogs.NewClient(client)

	_, apiErr, err := hcClient.Delete(ctx, h.Id, hostcatalogs.WithScopeId(h.Scope.Id))
	if err != nil {
		return fmt.Errorf("error calling new host catalog: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error reading host catalog: %s", apiErr.Message)
	}

	return nil
}
