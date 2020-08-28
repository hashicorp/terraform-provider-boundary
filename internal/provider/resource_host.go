package provider

import (
	"fmt"

	"github.com/hashicorp/boundary/api/hosts"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	hostNameKey        = "name"
	hostDescriptionKey = "description"
	hostScopeIDKey     = "scope_id"
	hostCatalogIDKey   = "host_catalog_id"
	hostAddressKey     = "address"
)

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
			hostCatalogIDKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			hostAddressKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			hostScopeIDKey: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
		},
	}
}

// convertHostToResourceData creates a ResourceData type from a Host
func convertHostToResourceData(h *hosts.Host, d *schema.ResourceData) error {
	if h.Name != "" {
		if err := d.Set(hostNameKey, h.Name); err != nil {
			return err
		}
	}

	if h.Description != "" {
		if err := d.Set(hostDescriptionKey, h.Description); err != nil {
			return err
		}
	}

	if h.Scope.Id != "" {
		if err := d.Set(hostScopeIDKey, h.Scope.Id); err != nil {
			return err
		}
	}

	if h.HostCatalogId != "" {
		if err := d.Set(hostCatalogIDKey, h.HostCatalogId); err != nil {
			return err
		}
	}

	if len(h.Attributes) != 0 {
		if addr, ok := h.Attributes["address"]; ok {
			if err := d.Set(hostAddressKey, addr); err != nil {
				return err
			}
		}
	}

	d.SetId(h.Id)

	return nil
}

// convertResourceDataToHost returns a localy built Host using the values provided in the ResourceData.
func convertResourceDataToHost(d *schema.ResourceData) *hosts.Host {
	// if you're manually defining the host in TF, it's always going
	// to be of type "static"
	h := &hosts.Host{
		Scope: &scopes.ScopeInfo{},
		Type:  hostCatalogTypeStatic,
		Attributes: map[string]interface{}{
			"address": "",
		},
	}

	if descVal, ok := d.GetOk(hostDescriptionKey); ok {
		h.Description = descVal.(string)
	}

	if nameVal, ok := d.GetOk(hostNameKey); ok {
		h.Name = nameVal.(string)
	}

	if scopeIDVal, ok := d.GetOk(hostScopeIDKey); ok {
		h.Scope.Id = scopeIDVal.(string)
	}

	if hostCatalogVal, ok := d.GetOk(hostCatalogIDKey); ok {
		h.HostCatalogId = hostCatalogVal.(string)
	}

	if hostAddrVal, ok := d.GetOk(hostAddressKey); ok {
		h.Attributes["address"] = hostAddrVal.(string)
	}

	if d.Id() != "" {
		h.Id = d.Id()
	}

	return h
}

func resourceHostCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	h := convertResourceDataToHost(d)
	usrs := hosts.NewClient(client)

	h, apiErr, err := usrs.Create(
		ctx,
		h.HostCatalogId,
		hosts.WithName(h.Name),
		hosts.WithDescription(h.Description),
		// not checking the key or the type because it's guaranteed set and string
		// when calling convertResourceDataToHost()
		hosts.WithStaticHostAddress(h.Attributes["address"].(string)),
		hosts.WithScopeId(h.Scope.Id))
	if err != nil {
		return fmt.Errorf("error calling new host: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error creating host: %s", apiErr.Message)
	}

	d.SetId(h.Id)

	return convertHostToResourceData(h, d)
}

func resourceHostRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	h := convertResourceDataToHost(d)
	usrs := hosts.NewClient(client)

	h, apiErr, err := usrs.Read(ctx, h.HostCatalogId, h.Id, hosts.WithScopeId(h.Scope.Id))
	if err != nil {
		return fmt.Errorf("error reading host: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error reading host: %s", apiErr.Message)
	}

	return convertHostToResourceData(h, d)
}

func resourceHostUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	h := convertResourceDataToHost(d)
	usrs := hosts.NewClient(client)

	if d.HasChange(hostNameKey) {
		h.Name = d.Get(hostNameKey).(string)
	}

	if d.HasChange(hostDescriptionKey) {
		h.Description = d.Get(hostDescriptionKey).(string)
	}

	if d.HasChange(hostAddressKey) {
		h.Attributes["address"] = d.Get(hostAddressKey).(string)
	}

	h.Scope.Id = d.Get(hostScopeIDKey).(string)

	h, apiErr, err := usrs.Update(
		ctx,
		h.HostCatalogId,
		h.Id,
		0,
		hosts.WithStaticHostAddress(h.Attributes["address"].(string)),
		hosts.WithAutomaticVersioning(),
		hosts.WithName(h.Name),
		hosts.WithDescription(h.Description),
		hosts.WithScopeId(h.Scope.Id))
	if err != nil {
		return err
	}
	if apiErr != nil {
		return fmt.Errorf("error updating host: %s\n   Invalid request fields: %v\n", apiErr.Message, apiErr.Details.RequestFields)
	}

	return convertHostToResourceData(h, d)
}

func resourceHostDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	h := convertResourceDataToHost(d)
	usrs := hosts.NewClient(client)

	_, apiErr, err := usrs.Delete(ctx, h.HostCatalogId, h.Id, hosts.WithScopeId(h.Scope.Id))
	if err != nil {
		return fmt.Errorf("error deleting host: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error deleting host: %s", apiErr.Message)
	}

	return nil
}
