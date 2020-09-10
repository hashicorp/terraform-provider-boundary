package provider

import (
	"errors"
	"fmt"

	"github.com/hashicorp/boundary/api/hostcatalogs"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	hostCatalogNameKey        = "name"
	hostCatalogDescriptionKey = "description"
	hostCatalogScopeIDKey     = "scope_id"
	hostCatalogTypeKey        = "type"
	hostCatalogTypeStatic     = "static"
)

func resourceHostCatalog() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceHostCatalogCreate,
		ReadContext:   resourceHostCatalogRead,
		UpdateContext: resourceHostCatalogUpdate,
		DeleteContext: resourceHostCatalogDelete,
		Schema: map[string]*schema.Schema{
			hostCatalogNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			hostCatalogDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			hostCatalogScopeIDKey: {
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

func validateHostCatalogType(val interface{}, key string) (warns []string, errs []error) {
	allow := []string{hostCatalogTypeStatic}
	v := val.(string)

	for _, a := range allow {
		if a == v {
			return
		}
	}

	errs = append(errs, fmt.Errorf("%s is not a supported host catalog type, please use one of %v", v, allow))
	return
}

// convertHostCatalogToResourceData creates a ResourceData type from a HostCatalog
func convertHostCatalogToResourceData(h *hostcatalogs.HostCatalog, d *schema.ResourceData) error {
	if h.Name != "" {
		if err := d.Set(hostCatalogNameKey, h.Name); err != nil {
			return err
		}
	}

	if h.Description != "" {
		if err := d.Set(hostCatalogDescriptionKey, h.Description); err != nil {
			return err
		}
	}

	if h.Type != "" {
		if err := d.Set(hostCatalogTypeKey, h.Type); err != nil {
			return err
		}
	}

	if h.Scope.Id != "" {
		if err := d.Set(hostCatalogScopeIDKey, h.Scope.Id); err != nil {
			return err
		}
	}

	d.SetId(h.Id)

	return nil
}

// convertResourceDataToHostCatalog returns a localy built HostCatalog using the values provided in the ResourceData.
func convertResourceDataToHostCatalog(d *schema.ResourceData) *hostcatalogs.HostCatalog {
	h := &hostcatalogs.HostCatalog{Scope: &scopes.ScopeInfo{}}

	if descVal, ok := d.GetOk(hostCatalogDescriptionKey); ok {
		h.Description = descVal.(string)
	}

	if nameVal, ok := d.GetOk(hostCatalogNameKey); ok {
		h.Name = nameVal.(string)
	}

	if typeVal, ok := d.GetOk(hostCatalogTypeKey); ok {
		h.Type = typeVal.(string)
	}

	if d.Id() != "" {
		h.Id = d.Id()
	}

	if scopeIDVal, ok := d.GetOk(hostCatalogScopeIDKey); ok {
		// if scope_id is not set, don't override with an empty scope
		if scopeIDVal.(string) != "" {
			h.Scope.Id = scopeIDVal.(string)
		}
	}

	return h
}

func resourceHostCatalogCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	h := convertResourceDataToHostCatalog(d)
	hcClient := hostcatalogs.NewClient(client)

	h, apiErr, err := hcClient.Create(
		ctx,
		h.Type,
		hostcatalogs.WithName(h.Name),
		hostcatalogs.WithDescription(h.Description),
		hostcatalogs.WithScopeId(h.Scope.Id))
	if err != nil {
		return fmt.Errorf("error calling new host catalog: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error creating host catalog: %s", apiErr.Message)
	}

	d.SetId(h.Id)

	return convertHostCatalogToResourceData(h, d)
}

func resourceHostCatalogRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	h := convertResourceDataToHostCatalog(d)
	hcClient := hostcatalogs.NewClient(client)

	h, apiErr, err := hcClient.Read(ctx, h.Id, hostcatalogs.WithScopeId(h.Scope.Id))
	if err != nil {
		return fmt.Errorf("error calling new host catalog: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error reading host catalog: %s", apiErr.Message)
	}

	return convertHostCatalogToResourceData(h, d)
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
