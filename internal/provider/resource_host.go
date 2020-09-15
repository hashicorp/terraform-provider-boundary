package provider

import (
	"context"

	"github.com/hashicorp/boundary/api/hosts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	hostTypeStatic = "static"
	hostAddressKey = "address"
)

func resourceHost() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceHostCreate,
		ReadContext:   resourceHostRead,
		UpdateContext: resourceHostUpdate,
		DeleteContext: resourceHostDelete,
		Schema: map[string]*schema.Schema{
			NameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			DescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			TypeKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			HostCatalogIdKey: {
				Type:     schema.TypeString,
				Required: true,
			},
			hostAddressKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func setFromHostResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(HostCatalogIdKey, raw["host_catalog_id"])
	d.Set(TypeKey, raw["type"])

	switch raw["type"].(string) {
	case hostTypeStatic:
		if attrsVal, ok := raw["attributes"]; ok {
			attrs := attrsVal.(map[string]interface{})
			d.Set(hostAddressKey, attrs["address"])
		}
	}

	d.SetId(raw["id"].(string))
}

func resourceHostCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var hostCatalogId string
	if hostCatalogIdVal, ok := d.GetOk(HostCatalogIdKey); ok {
		hostCatalogId = hostCatalogIdVal.(string)
	} else {
		return diag.Errorf("no host catalog ID provided")
	}

	var address *string
	if addressVal, ok := d.GetOk(hostAddressKey); ok {
		hostAddress := addressVal.(string)
		address = &hostAddress
	}

	opts := []hosts.Option{}

	var typeStr string
	if typeVal, ok := d.GetOk(TypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}
	switch typeStr {
	// NOTE: When other types are added, ensure they don't accept address if
	// it's not allowed
	case hostTypeStatic:
		if address != nil {
			opts = append(opts, hosts.WithStaticHostAddress(*address))
		} else {
			return diag.Errorf("no address provided")
		}

	default:
		return diag.Errorf("invalid type provided")
	}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, hosts.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, hosts.WithDescription(descStr))
	}

	hClient := hosts.NewClient(md.client)

	hcr, apiErr, err := hClient.Create(
		ctx,
		hostCatalogId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling create host: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating host: %s", apiErr.Message)
	}
	if hcr == nil {
		return diag.Errorf("host nil after create")
	}

	setFromHostResponseMap(d, hcr.GetResponseMap())

	return nil
}

func resourceHostRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hClient := hosts.NewClient(md.client)

	hrr, apiErr, err := hClient.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling read host: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading host: %s", apiErr.Message)
	}
	if hrr == nil {
		return diag.Errorf("host nil after read")
	}

	setFromHostResponseMap(d, hrr.GetResponseMap())

	return nil
}

func resourceHostUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hClient := hosts.NewClient(md.client)

	opts := []hosts.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, hosts.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, hosts.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, hosts.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, hosts.WithDescription(descStr))
		}
	}

	var address *string
	if d.HasChange(hostAddressKey) {
		switch d.Get(TypeKey).(string) {
		case hostTypeStatic:
			opts = append(opts, hosts.DefaultStaticHostAddress())
			addrVal, ok := d.GetOk(hostAddressKey)
			if ok {
				addrStr := addrVal.(string)
				address = &addrStr
				opts = append(opts, hosts.WithStaticHostAddress(addrStr))
			}
		default:
			return diag.Errorf("address cannot be used with this type of host")
		}
	}

	if len(opts) > 0 {
		opts = append(opts, hosts.WithAutomaticVersioning(true))
		_, apiErr, err := hClient.Update(
			ctx,
			d.Id(),
			0,
			opts...)
		if err != nil {
			return diag.Errorf("error calling update host: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating host: %s", apiErr.Message)
		}
	}

	if d.HasChange(NameKey) {
		d.Set(NameKey, name)
	}
	if d.HasChange(DescriptionKey) {
		d.Set(DescriptionKey, desc)
	}
	if d.HasChange(hostAddressKey) {
		d.Set(hostAddressKey, *address)
	}

	return nil
}

func resourceHostDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hClient := hosts.NewClient(md.client)

	_, apiErr, err := hClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete host: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error deleting host: %s", apiErr.Message)
	}

	return nil
}
