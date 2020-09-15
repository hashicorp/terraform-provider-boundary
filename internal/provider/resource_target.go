package provider

import (
	"context"

	"github.com/hashicorp/boundary/api/targets"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	targetHostSetIdsKey  = "host_set_ids"
	targetDefaultPortKey = "default_port"

	targetTypeTcp = "tcp"
)

func resourceTarget() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTargetCreate,
		ReadContext:   resourceTargetRead,
		UpdateContext: resourceTargetUpdate,
		DeleteContext: resourceTargetDelete,
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
			ScopeIdKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			targetDefaultPortKey: {
				Type:     schema.TypeInt,
				Optional: true,
			},
			targetHostSetIdsKey: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func setFromTargetResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(ScopeIdKey, raw["scope_id"])
	d.Set(TypeKey, raw["type"])
	d.Set(targetDefaultPortKey, raw["default_port"])
	d.Set(targetHostSetIdsKey, raw["host_set_ids"])
	d.SetId(raw["id"].(string))
}

func resourceTargetCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
	switch typeStr {
	case targetTypeTcp:
	default:
		return diag.Errorf("invalid type provided")
	}

	opts := []targets.Option{}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, targets.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, targets.WithDescription(descStr))
	}

	var defaultPort *int
	defaultPortVal, ok := d.GetOk(targetDefaultPortKey)
	if ok {
		defaultPortInt := defaultPortVal.(int)
		if defaultPortInt < 0 {
			return diag.Errorf(`"default_port" cannot be less than zero`)
		}
		defaultPort = &defaultPortInt
		opts = append(opts, targets.WithDefaultPort(uint32(*defaultPort)))
	}

	var hostSetIds []string
	if hostSetIdsVal, ok := d.GetOk(targetHostSetIdsKey); ok {
		list := hostSetIdsVal.(*schema.Set).List()
		hostSetIds = make([]string, 0, len(list))
		for _, i := range list {
			hostSetIds = append(hostSetIds, i.(string))
		}
	}

	tc := targets.NewClient(md.client)

	tcr, apiErr, err := tc.Create(
		ctx,
		typeStr,
		scopeId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling create target: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating target: %s", apiErr.Message)
	}
	if tcr == nil {
		return diag.Errorf("target nil after create")
	}

	raw := tcr.GetResponseMap()

	if hostSetIds != nil {
		tur, apiErr, err := tc.SetHostSets(
			ctx,
			tcr.Item.Id,
			tcr.Item.Version,
			hostSetIds)
		if apiErr != nil {
			return diag.Errorf("error setting host sets on target: %s", apiErr.Message)
		}
		if err != nil {
			return diag.Errorf("error setting host sets on target: %v", err)
		}
		d.Set(targetHostSetIdsKey, hostSetIds)
		raw = tur.GetResponseMap()
	}

	setFromTargetResponseMap(d, raw)

	return nil
}

func resourceTargetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	tc := targets.NewClient(md.client)

	trr, apiErr, err := tc.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling read target: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading target: %s", apiErr.Message)
	}
	if trr == nil {
		return diag.Errorf("target nil after read")
	}

	setFromTargetResponseMap(d, trr.GetResponseMap())

	return nil
}

func resourceTargetUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	tc := targets.NewClient(md.client)

	opts := []targets.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, targets.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, targets.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, targets.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, targets.WithDescription(descStr))
		}
	}

	var defaultPort *int
	if d.HasChange(targetDefaultPortKey) {
		opts = append(opts, targets.DefaultDefaultPort())
		defaultPortVal, ok := d.GetOk(targetDefaultPortKey)
		if ok {
			defaultPortInt := defaultPortVal.(int)
			if defaultPortInt < 0 {
				return diag.Errorf(`"default_port" cannot be less than zero`)
			}
			defaultPort = &defaultPortInt
			opts = append(opts, targets.WithDefaultPort(uint32(defaultPortInt)))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, targets.WithAutomaticVersioning(true))
		_, apiErr, err := tc.Update(
			ctx,
			d.Id(),
			0,
			opts...)
		if err != nil {
			return diag.Errorf("error calling update target: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating target: %s", apiErr.Message)
		}
	}

	if d.HasChange(NameKey) {
		d.Set(NameKey, name)
	}
	if d.HasChange(DescriptionKey) {
		d.Set(DescriptionKey, desc)
	}
	if d.HasChange(targetDefaultPortKey) {
		d.Set(targetDefaultPortKey, defaultPort)
	}

	// The above call may not actually happen, so we use d.Id() and automatic
	// versioning here
	if d.HasChange(targetHostSetIdsKey) {
		var hostSetIds []string
		if hostSetIdsVal, ok := d.GetOk(targetHostSetIdsKey); ok {
			hostSets := hostSetIdsVal.(*schema.Set).List()
			for _, hostSet := range hostSets {
				hostSetIds = append(hostSetIds, hostSet.(string))
			}
		}
		_, apiErr, err := tc.SetHostSets(
			ctx,
			d.Id(),
			0,
			hostSetIds,
			targets.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating host sets in target: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating host sets in target: %s", apiErr.Message)
		}
		d.Set(targetHostSetIdsKey, hostSetIds)
	}

	return nil
}

func resourceTargetDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	tc := targets.NewClient(md.client)

	_, apiErr, err := tc.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete target: %s", err.Error())
	}
	if apiErr != nil {
		return diag.Errorf("error deleting target: %s", apiErr.Message)
	}

	return nil
}
