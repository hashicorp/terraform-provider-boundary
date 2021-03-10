package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
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
		Description: "The host resource allows you to configure a Boundary static host. Hosts are " +
			"always part of a project, so a project resource should be used inline or you should have " +
			"the project ID in hand to successfully configure a host.",

		CreateContext: resourceHostCreate,
		ReadContext:   resourceHostRead,
		UpdateContext: resourceHostUpdate,
		DeleteContext: resourceHostDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the host.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The host name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The host description.",
				Type:        schema.TypeString,
				Optional:    true,
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
				Description: "The static address of the host resource as `<IP>` (note: port assignment occurs in the target resource definition, do not add :port here) or a domain name.",
				Type:        schema.TypeString,
				Optional:    true,
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

	hcr, err := hClient.Create(ctx, hostCatalogId, opts...)
	if err != nil {
		return diag.Errorf("error creating host: %v", err)
	}
	if hcr == nil {
		return diag.Errorf("host nil after create")
	}

	setFromHostResponseMap(d, hcr.GetResponse().Map)

	return nil
}

func resourceHostRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hClient := hosts.NewClient(md.client)

	hrr, err := hClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading host: %v", err)
	}
	if hrr == nil {
		return diag.Errorf("host nil after read")
	}

	setFromHostResponseMap(d, hrr.GetResponse().Map)

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
		_, err := hClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating host: %v", err)
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

	_, err := hClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting host: %v", err)
	}

	return nil
}
