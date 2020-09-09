package provider

import (
	"fmt"

	"github.com/hashicorp/boundary/api/hostsets"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	hostsetNameKey        = "name"
	hostsetDescriptionKey = "description"
	hostsetCatalogIDKey   = "host_catalog_id"
	hostsetHostIDsKey     = "host_ids"
)

func resourceHostset() *schema.Resource {
	return &schema.Resource{
		Create: resourceHostsetCreate,
		Read:   resourceHostsetRead,
		Update: resourceHostsetUpdate,
		Delete: resourceHostsetDelete,
		Schema: map[string]*schema.Schema{
			hostsetNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			hostsetDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			hostsetCatalogIDKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			hostsetHostIDsKey: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

// convertHostsetToResourceData creates a ResourceData type from a Host
func convertHostsetToResourceData(h *hostsets.HostSet, d *schema.ResourceData) error {
	if h.Name != "" {
		if err := d.Set(hostsetNameKey, h.Name); err != nil {
			return err
		}
	}

	if h.Description != "" {
		if err := d.Set(hostsetDescriptionKey, h.Description); err != nil {
			return err
		}
	}

	if h.HostCatalogId != "" {
		if err := d.Set(hostsetCatalogIDKey, h.HostCatalogId); err != nil {
			return err
		}
	}

	if h.HostIds != nil {
		if err := d.Set(hostsetHostIDsKey, h.HostIds); err != nil {
			return err
		}
	}

	d.SetId(h.Id)

	return nil
}

// convertResourceDataToHostset returns a localy built Host using the values provided in the ResourceData.
func convertResourceDataToHostset(d *schema.ResourceData) *hostsets.HostSet {
	// if you're manually defining the hostset in TF, it's always going
	// to be of type "static"
	h := &hostsets.HostSet{
		Scope:      &scopes.ScopeInfo{},
		HostIds:    []string{},
		Attributes: map[string]interface{}{},
	}

	if descVal, ok := d.GetOk(hostsetDescriptionKey); ok {
		h.Description = descVal.(string)
	}

	if nameVal, ok := d.GetOk(hostsetNameKey); ok {
		h.Name = nameVal.(string)
	}

	if hostsetCatalogVal, ok := d.GetOk(hostCatalogIDKey); ok {
		h.HostCatalogId = hostsetCatalogVal.(string)
	}

	if val, ok := d.GetOk(hostsetHostIDsKey); ok {
		hostIds := val.(*schema.Set).List()
		for _, i := range hostIds {
			h.HostIds = append(h.HostIds, i.(string))
		}
	}

	if d.Id() != "" {
		h.Id = d.Id()
	}

	return h
}

func resourceHostsetCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	h := convertResourceDataToHostset(d)
	hst := hostsets.NewClient(client)

	hostIDs := h.HostIds

	h, apiErr, err := hst.Create2(
		ctx,
		h.HostCatalogId,
		hostsets.WithName(h.Name),
		hostsets.WithDescription(h.Description))
	if err != nil {
		return fmt.Errorf("error calling new hostset: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error creating hostset: %s", apiErr.Message)
	}

	d.SetId(h.Id)

	if len(hostIDs) != 0 {
		h, apiErr, err = hst.SetHosts2(
			ctx,
			h.Id,
			0,
			hostIDs,
			hostsets.WithAutomaticVersioning(),
			hostsets.WithName(h.Name),
			hostsets.WithDescription(h.Description))
		if err != nil {
			return fmt.Errorf("error setting hosts on hostset: %s", err.Error())
		}
		if apiErr != nil {
			return fmt.Errorf("error setting hosts on hostset: %s", apiErr.Message)
		}
	}

	return convertHostsetToResourceData(h, d)
}

func resourceHostsetRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	h := convertResourceDataToHostset(d)
	hst := hostsets.NewClient(client)

	h, apiErr, err := hst.Read2(ctx, h.Id)
	if err != nil {
		return fmt.Errorf("error reading hostset: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error reading hostset: %s", apiErr.Message)
	}

	return convertHostsetToResourceData(h, d)
}

func resourceHostsetUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	h := convertResourceDataToHostset(d)
	hst := hostsets.NewClient(client)

	if d.HasChange(hostsetNameKey) {
		h.Name = d.Get(hostsetNameKey).(string)
	}

	if d.HasChange(hostsetDescriptionKey) {
		h.Description = d.Get(hostsetDescriptionKey).(string)
	}

	if d.HasChange(hostsetHostIDsKey) {
		hostIDs := []string{}
		hosts := d.Get(hostsetHostIDsKey).(*schema.Set).List()

		for _, host := range hosts {
			hostIDs = append(hostIDs, host.(string))
		}

		_, apiErr, err := hst.SetHosts2(
			ctx,
			h.Id,
			0,
			hostIDs,
			hostsets.WithAutomaticVersioning(),
			hostsets.WithName(h.Name),
			hostsets.WithDescription(h.Description))
		if err != nil {
			return fmt.Errorf("error setting hosts on hostset: %s", err.Error())
		}
		if apiErr != nil {
			return fmt.Errorf("error setting hosts on hostset: %s", apiErr.Message)
		}
	}

	h, apiErr, err := hst.Update2(
		ctx,
		h.Id,
		0,
		hostsets.WithAutomaticVersioning(),
		hostsets.WithName(h.Name),
		hostsets.WithDescription(h.Description))
	if err != nil {
		return err
	}
	if apiErr != nil {
		return fmt.Errorf("error updating hostset: %s\n   Invalid request fields: %v\n", apiErr.Message, apiErr.Details.RequestFields)
	}

	return convertHostsetToResourceData(h, d)
}

func resourceHostsetDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	h := convertResourceDataToHostset(d)
	hst := hostsets.NewClient(client)

	_, apiErr, err := hst.Delete2(ctx, h.Id)
	if err != nil {
		return fmt.Errorf("error deleting hostset: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error deleting hostset: %s", apiErr.Message)
	}

	return nil
}
