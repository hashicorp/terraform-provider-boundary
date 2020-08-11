package provider

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/boundary/api/hosts"
	"github.com/hashicorp/boundary/api/scopes"
)

const (
	hostCatalogNameKey        = "name"
	hostCatalogDescriptionKey = "description"
	hostCatalogProjectIDKey   = "project_id"
	hostCatalogTypeKey        = "type"
	hostCatalogTypeStatic     = "Static"
)

func resourceHostCatalog() *schema.Resource {
	return &schema.Resource{
		Create: resourceHostCatalogCreate,
		Read:   resourceHostCatalogRead,
		Update: resourceHostCatalogUpdate,
		Delete: resourceHostCatalogDelete,
		Schema: map[string]*schema.Schema{
			hostCatalogNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			hostCatalogDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			hostCatalogProjectIDKey: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			hostCatalogTypeKey: {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateHostCatalogType,
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
func convertHostCatalogToResourceData(projectID string, h *hosts.HostCatalog, d *schema.ResourceData) error {
	if h.Name != nil {
		if err := d.Set(hostCatalogNameKey, h.Name); err != nil {
			return err
		}
	}

	if h.Description != nil {
		if err := d.Set(hostCatalogDescriptionKey, h.Description); err != nil {
			return err
		}
	}

	if h.Type != nil {
		if err := d.Set(hostCatalogTypeKey, h.Type); err != nil {
			return err
		}
	}

	if projectID != "" {
		if err := d.Set(hostCatalogProjectIDKey, projectID); err != nil {
			return err
		}
	}

	d.SetId(h.Id)

	return nil
}

// convertResourceDataToHostCatalog returns a localy built HostCatalog using the values provided in the ResourceData.
func convertResourceDataToHostCatalog(d *schema.ResourceData) (string, *hosts.HostCatalog) {
	h := &hosts.HostCatalog{}
	if descVal, ok := d.GetOk(hostCatalogDescriptionKey); ok {
		desc := descVal.(string)
		h.Description = &desc
	}
	if nameVal, ok := d.GetOk(hostCatalogNameKey); ok {
		name := nameVal.(string)
		h.Name = &name
	}
	if typeVal, ok := d.GetOk(hostCatalogTypeKey); ok {
		t := typeVal.(string)
		h.Type = &t
	}
	if d.Id() != "" {
		h.Id = d.Id()
	}

	projID, ok := d.GetOk(hostCatalogProjectIDKey)
	if !ok {
		projID = ""
	}

	return projID.(string), h
}

func resourceHostCatalogCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	projID, h := convertResourceDataToHostCatalog(d)
	p := &scopes.Project{
		Client: client,
		Id:     projID,
	}

	h, apiErr, err := p.CreateHostCatalog(ctx, h)
	if err != nil {
		return fmt.Errorf("error calling new host catalog: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error creating host catalog: %s", apiErr.Message)
	}

	d.SetId(h.Id)

	return nil
}

func resourceHostCatalogRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	projID, h := convertResourceDataToHostCatalog(d)
	p := &scopes.Project{
		Client: client,
		Id:     projID,
	}

	h, apiErr, err := p.ReadHostCatalog(ctx, h)
	if err != nil {
		return fmt.Errorf("error calling new host catalog: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error reading host catalog: %s", apiErr.Message)
	}

	return convertHostCatalogToResourceData(projID, h, d)
}

func resourceHostCatalogUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	projID, h := convertResourceDataToHostCatalog(d)
	p := &scopes.Project{
		Client: client,
		Id:     projID,
	}

	if d.HasChange(hostCatalogNameKey) {
		n := d.Get(hostCatalogNameKey).(string)
		h.Name = &n
	}

	if d.HasChange(hostCatalogDescriptionKey) {
		d := d.Get(hostCatalogDescriptionKey).(string)
		h.Description = &d
	}

	if d.HasChange(hostCatalogProjectIDKey) {
		id := d.Get(hostCatalogProjectIDKey).(string)
		p.Id = id
	}

	if d.HasChange(hostCatalogTypeKey) {
		return errors.New("error updating host catalog: A host catalog can not have its type modified.")
	}

	// Type is a read-only value that can not be updated. It is added in the method to convert from a
	// resource to a HostCatalog type, so it needs to be unset when calling update.
	h.Type = nil
	h, apiErr, err := p.UpdateHostCatalog(ctx, h)
	if err != nil {
		return err
	}
	if apiErr != nil {
		return fmt.Errorf("error updating host catalog: %s\n   Invalid request fields: %v\n", apiErr.Message, apiErr.Details.RequestFields)
	}

	return convertHostCatalogToResourceData(projID, h, d)
}

func resourceHostCatalogDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	projID, h := convertResourceDataToHostCatalog(d)
	p := &scopes.Project{
		Client: client,
		Id:     projID,
	}

	_, apiErr, err := p.DeleteHostCatalog(ctx, h)
	if err != nil {
		return fmt.Errorf("error calling new host catalog: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error reading host catalog: %s", apiErr.Message)
	}

	return nil
}
