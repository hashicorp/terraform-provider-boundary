package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/watchtower/api/hosts"
)

const (
	hostNameKey        = "name"
	hostDescriptionKey = "description"
	hostProjectIDKey   = "project_id"
	hostTypeKey        = "type"
	hostCatalogIDKey   = "host_catalog_id"

	hostTypeStatic = "Static"
)

var errNotImplemented = func(s string) error { return fmt.Errorf("err: %s not implemented") }

func resourceHost() *schema.Resource {
	return &schema.Resource{
		Create: resourceHostCreate,
		Read:   resourceHostRead,
		Update: resourceHostUpdate,
		Delete: resourceHostDelete,
		Schema: map[string]*schema.Schema{
			hostNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			hostDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			hostTypeKey: {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateHostType,
			},
			hostCatalogIDKey: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}

}

func validateHostType(val interface{}, key string) (warns []string, errs []error) {
	allow := []string{hostTypeStatic}
	v := val.(string)

	for _, a := range allow {
		if a == v {
			return
		}
	}

	errs = append(errs, fmt.Errorf("%s is not a supported host catalog type, please use one of %v", v, allow))
	return
}

// convertHostToResourceData creates a ResourceData type from a Host
func convertHostToResourceData(hostCatalogID, h *hosts.Host, d *schema.ResourceData) error {
	if h.Name != nil {
		if err := d.Set(hostNameKey, h.Name); err != nil {
			return err
		}
	}

	if h.Description != nil {
		if err := d.Set(hostDescriptionKey, h.Description); err != nil {
			return err
		}
	}

	if h.Type != nil {
		if err := d.Set(hostTypeKey, h.Type); err != nil {
			return err
		}
	}

	if hostCatalogID != "" {
		if err := d.Set(hostCatalogIDKey, hostCatalogID); err != nil {
			return err
		}
	}

	d.SetId(h.Id)

	return nil
}

// convertResourceDataToHost returns a localy built Host using the values provided in the ResourceData.
func convertResourceDataToHost(d *schema.ResourceData) (string, *hosts.Host) {
	h := &hosts.Host{}
	if descVal, ok := d.GetOk(hostDescriptionKey); ok {
		desc := descVal.(string)
		h.Description = &desc
	}
	if nameVal, ok := d.GetOk(hostNameKey); ok {
		name := nameVal.(string)
		h.Name = &name
	}
	if typeVal, ok := d.GetOk(hostTypeKey); ok {
		t := typeVal.(string)
		h.Type = &t
	}
	if d.Id() != "" {
		h.Id = d.Id()
	}

	hostCatalogID, ok := d.GetOk(hostCatalogIDKey)
	if !ok {
		hostCatalogID = ""
	}

	return hostCatalogID.(string), h
}

func resourceHostCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	hostCatalogID, h := convertResourceDataToHost(d)

	hc := &hosts.HostCatalog{
		Client: client,
		Id:     hostCatalogID,
	}

	h, apiErr, err := hc.CreateHost(ctx, h)
	if err != nil {
		return fmt.Errorf("error calling new host catalog: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error creating host catalog: %s", *apiErr.Message)
	}

	d.SetId(h.Id)

	return nil
}

func resourceHostRead(d *schema.ResourceData, meta interface{}) error {
	return errNotImplemented("read host")
}

func resourceHostUpdate(d *schema.ResourceData, meta interface{}) error {
	return errNotImplemented("update host")
}

func resourceHostDelete(d *schema.ResourceData, meta interface{}) error {
	return errNotImplemented("delete host")
}
