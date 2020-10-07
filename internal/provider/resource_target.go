package provider

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/targets"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	targetHostSetIdsKey             = "host_set_ids"
	targetDefaultPortKey            = "default_port"
	targetSessionMaxSecondsKey      = "session_max_seconds"
	targetSessionConnectionLimitKey = "session_connection_limit"

	targetTypeTcp = "tcp"
)

func resourceTarget() *schema.Resource {
	return &schema.Resource{
		Description: "The target resource allows you to configure a Boundary target.",

		CreateContext: resourceTargetCreate,
		ReadContext:   resourceTargetRead,
		UpdateContext: resourceTargetUpdate,
		DeleteContext: resourceTargetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the target.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The target name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The target description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			TypeKey: {
				Description: "The target resource type.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			targetDefaultPortKey: {
				Description: "The default port for this target.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			targetHostSetIdsKey: {
				Description: "A list of host set ID's.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			targetSessionMaxSecondsKey: {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			targetSessionConnectionLimitKey: {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func setFromTargetResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(ScopeIdKey, raw["scope_id"])
	d.Set(TypeKey, raw["type"])
	d.Set(targetHostSetIdsKey, raw["host_set_ids"])
	d.Set(targetSessionMaxSecondsKey, raw["session_max_seconds"])
	d.Set(targetSessionConnectionLimitKey, raw["session_connection_limit"])

	switch raw["type"].(string) {
	case targetTypeTcp:
		if attrsVal, ok := raw["attributes"]; ok {
			attrs := attrsVal.(map[string]interface{})
			if defPort, ok := attrs["default_port"].(json.Number); ok {
				defPortInt, _ := defPort.Int64()
				d.Set(targetDefaultPortKey, int(defPortInt))
			}
		}
	}

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

	defaultPortVal, ok := d.GetOk(targetDefaultPortKey)
	if ok {
		defaultPortInt := defaultPortVal.(int)
		if defaultPortInt < 0 {
			return diag.Errorf(`"default_port" cannot be less than zero`)
		}
		opts = append(opts, targets.WithTcpTargetDefaultPort(uint32(defaultPortInt)))
	}

	sessionMaxSecondsVal, ok := d.GetOk(targetSessionMaxSecondsKey)
	if ok {
		sessionMaxSecondsInt := sessionMaxSecondsVal.(int)
		if sessionMaxSecondsInt <= 0 {
			return diag.Errorf(`"session_max_seconds" must be greater than zero`)
		}
		opts = append(opts, targets.WithSessionMaxSeconds(uint32(sessionMaxSecondsInt)))
	}

	sessionConnectionLimitVal, ok := d.GetOk(targetSessionConnectionLimitKey)
	if ok {
		sessionConnectionLimitInt := sessionConnectionLimitVal.(int)
		if sessionConnectionLimitInt != -1 && sessionConnectionLimitInt <= 0 {
			return diag.Errorf(`"session_connection_limit" must be -1 or greater than zero`)
		}
		opts = append(opts, targets.WithSessionConnectionLimit(int32(sessionConnectionLimitInt)))
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

	tcr, err := tc.Create(ctx, typeStr, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating target: %v", err)
	}
	if tcr == nil {
		return diag.Errorf("target nil after create")
	}
	raw := tcr.GetResponseMap()

	if hostSetIds != nil {
		tur, err := tc.SetHostSets(ctx, tcr.Item.Id, tcr.Item.Version, hostSetIds)
		if err != nil {
			return diag.Errorf("error setting host sets on target: %v", err)
		}
		raw = tur.GetResponseMap()
	}

	setFromTargetResponseMap(d, raw)

	return nil
}

func resourceTargetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	tc := targets.NewClient(md.client)

	trr, err := tc.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Status == int32(http.StatusNotFound) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading target: %v", err)
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
		opts = append(opts, targets.DefaultTcpTargetDefaultPort())
		defaultPortVal, ok := d.GetOk(targetDefaultPortKey)
		if ok {
			defaultPortInt := defaultPortVal.(int)
			if defaultPortInt < 0 {
				return diag.Errorf(`"default_port" cannot be less than zero`)
			}
			defaultPort = &defaultPortInt
			opts = append(opts, targets.WithTcpTargetDefaultPort(uint32(defaultPortInt)))
		}
	}

	var sessionMaxSeconds *int
	if d.HasChange(targetSessionMaxSecondsKey) {
		opts = append(opts, targets.DefaultSessionMaxSeconds())
		sessionMaxSecondsVal, ok := d.GetOk(targetSessionMaxSecondsKey)
		if ok {
			sessionMaxSecondsInt := sessionMaxSecondsVal.(int)
			if sessionMaxSecondsInt <= 0 {
				return diag.Errorf(`"session_max_seconds" must be greater than zero`)
			}
			sessionMaxSeconds = &sessionMaxSecondsInt
			opts = append(opts, targets.WithSessionMaxSeconds(uint32(sessionMaxSecondsInt)))
		}
	}

	var sessionConnectionLimit *int
	if d.HasChange(targetSessionConnectionLimitKey) {
		opts = append(opts, targets.DefaultSessionConnectionLimit())
		sessionConnectionLimitVal, ok := d.GetOk(targetSessionConnectionLimitKey)
		if ok {
			sessionConnectionLimitInt := sessionConnectionLimitVal.(int)
			if sessionConnectionLimitInt != -1 && sessionConnectionLimitInt <= 0 {
				return diag.Errorf(`"session_connection_limit" must be -1 or greater than zero`)
			}
			sessionConnectionLimit = &sessionConnectionLimitInt
			opts = append(opts, targets.WithSessionConnectionLimit(int32(sessionConnectionLimitInt)))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, targets.WithAutomaticVersioning(true))
		_, err := tc.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating target: %v", err)
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
	if d.HasChange(targetSessionMaxSecondsKey) {
		d.Set(targetSessionMaxSecondsKey, sessionMaxSeconds)
	}
	if d.HasChange(targetSessionConnectionLimitKey) {
		d.Set(targetSessionConnectionLimitKey, sessionConnectionLimit)
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
		_, err := tc.SetHostSets(ctx, d.Id(), 0, hostSetIds, targets.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating host sets in target: %v", err)
		}
		d.Set(targetHostSetIdsKey, hostSetIds)
	}

	return nil
}

func resourceTargetDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	tc := targets.NewClient(md.client)

	_, err := tc.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting target: %s", err.Error())
	}

	return nil
}
