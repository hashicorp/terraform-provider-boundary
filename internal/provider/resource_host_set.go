package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api/hostsets"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	hostsetHostIdsKey = "host_ids"
	hostsetTypeStatic = "static"
)

func resourceHostset() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceHostsetCreate,
		ReadContext:   resourceHostsetRead,
		UpdateContext: resourceHostsetUpdate,
		DeleteContext: resourceHostsetDelete,
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
			hostsetHostIdsKey: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func setFromHostSetResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(HostCatalogIdKey, raw["host_catalog_id"])
	d.Set(TypeKey, raw["type"])
	d.Set(hostsetHostIdsKey, raw["host_ids"])
	d.SetId(raw["id"].(string))
}

func resourceHostsetCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var hostsetHostCatalogId string
	if hostsetHostCatalogIdVal, ok := d.GetOk(HostCatalogIdKey); ok {
		hostsetHostCatalogId = hostsetHostCatalogIdVal.(string)
	} else {
		return diag.Errorf("no host catalog ID provided")
	}

	var hostIds []string
	if hostIdsVal, ok := d.GetOk(hostsetHostIdsKey); ok {
		list := hostIdsVal.(*schema.Set).List()
		hostIds = make([]string, 0, len(list))
		for _, i := range list {
			hostIds = append(hostIds, i.(string))
		}
	}

	opts := []hostsets.Option{}

	var typeStr string
	if typeVal, ok := d.GetOk(TypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}
	switch typeStr {
	// NOTE: When other types are added, ensure they don't accept hostSetIds if
	// it's not allowed
	case hostsetTypeStatic:
	default:
		return diag.Errorf("invalid type provided")
	}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, hostsets.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, hostsets.WithDescription(descStr))
	}

	hsClient := hostsets.NewClient(md.client)

	hscr, apiErr, err := hsClient.Create(
		ctx,
		hostsetHostCatalogId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling create host set: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating host set: %s", apiErr.Message)
	}
	if hscr == nil {
		return diag.Errorf("nil host set after create")
	}
	raw := hscr.GetResponseMap()

	if hostIds != nil {
		hsshr, apiErr, err := hsClient.SetHosts(
			ctx,
			hscr.Item.Id,
			hscr.Item.Version,
			hostIds)
		if apiErr != nil {
			return diag.Errorf("error setting hosts on host set: %s", apiErr.Message)
		}
		if err != nil {
			return diag.Errorf("error setting hosts on host set: %v", err)
		}
		if hsshr == nil {
			return diag.Errorf("nil host set after setting hosts")
		}
		raw = hsshr.GetResponseMap()
	}

	setFromHostSetResponseMap(d, raw)

	return nil
}

func resourceHostsetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hsClient := hostsets.NewClient(md.client)

	hsrr, apiErr, err := hsClient.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling read host set: %v", err)
	}
	if apiErr != nil {
		if apiErr.Status == int32(http.StatusNotFound) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading host set: %s", apiErr.Message)
	}
	if hsrr == nil {
		return diag.Errorf("host set nil after read")
	}

	setFromHostSetResponseMap(d, hsrr.GetResponseMap())

	return nil
}

func resourceHostsetUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hsClient := hostsets.NewClient(md.client)

	opts := []hostsets.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, hostsets.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, hostsets.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, hostsets.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, hostsets.WithDescription(descStr))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, hostsets.WithAutomaticVersioning(true))
		_, apiErr, err := hsClient.Update(
			ctx,
			d.Id(),
			0,
			opts...)
		if err != nil {
			return diag.Errorf("error calling update host set: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating host set: %s", apiErr.Message)
		}
	}

	if d.HasChange(NameKey) {
		d.Set(NameKey, name)
	}
	if d.HasChange(DescriptionKey) {
		d.Set(DescriptionKey, desc)
	}

	// The above call may not actually happen, so we use d.Id() and automatic
	// versioning here
	if d.HasChange(hostsetHostIdsKey) {
		var hostIds []string
		if hostIdsVal, ok := d.GetOk(hostsetHostIdsKey); ok {
			hosts := hostIdsVal.(*schema.Set).List()
			for _, host := range hosts {
				hostIds = append(hostIds, host.(string))
			}
		}
		_, apiErr, err := hsClient.SetHosts(
			ctx,
			d.Id(),
			0,
			hostIds,
			hostsets.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating hosts in host set: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating hosts in host set: %s", apiErr.Message)
		}
		d.Set(hostsetHostIdsKey, hostIds)
	}

	return nil
}

func resourceHostsetDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hsClient := hostsets.NewClient(md.client)

	_, apiErr, err := hsClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete host set: %s", err.Error())
	}
	if apiErr != nil {
		return diag.Errorf("error deleting host set: %s", apiErr.Message)
	}

	return nil
}
