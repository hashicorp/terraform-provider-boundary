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
	targetHostSourceIdsKey                  = "host_source_ids"
	targetApplicationCredentialSourceIdsKey = "application_credential_source_ids"
	targetDefaultPortKey                    = "default_port"
	targetSessionMaxSecondsKey              = "session_max_seconds"
	targetSessionConnectionLimitKey         = "session_connection_limit"
	targetWorkerFilterKey                   = "worker_filter"

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
			targetHostSourceIdsKey: {
				Description: "A list of host source ID's.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			targetApplicationCredentialSourceIdsKey: {
				Description: "A list of application credential source ID's.",
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
			targetWorkerFilterKey: {
				Description: "Boolean expression to filter the workers for this target",
				Type:        schema.TypeString,
				Optional:    true,
			},
		},
	}
}

func setFromTargetResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw["scope_id"]); err != nil {
		return err
	}
	if err := d.Set(TypeKey, raw["type"]); err != nil {
		return err
	}
	if err := d.Set(targetHostSourceIdsKey, raw["host_source_ids"]); err != nil {
		return err
	}
	if err := d.Set(targetApplicationCredentialSourceIdsKey, raw["application_credential_source_ids"]); err != nil {
		return err
	}
	if err := d.Set(targetSessionMaxSecondsKey, raw["session_max_seconds"]); err != nil {
		return err
	}
	if err := d.Set(targetSessionConnectionLimitKey, raw["session_connection_limit"]); err != nil {
		return err
	}
	if err := d.Set(targetWorkerFilterKey, raw["worker_filter"]); err != nil {
		return err
	}

	switch raw["type"].(string) {
	case targetTypeTcp:
		if attrsVal, ok := raw["attributes"]; ok {
			attrs := attrsVal.(map[string]interface{})
			if defPort, ok := attrs["default_port"].(json.Number); ok {
				defPortInt, _ := defPort.Int64()
				if err := d.Set(targetDefaultPortKey, int(defPortInt)); err != nil {
					return err
				}
			}
		}
	}

	d.SetId(raw["id"].(string))
	return nil
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

	var opts []targets.Option
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

	var hostSourceIds []string
	if hostSourceIdsVal, ok := d.GetOk(targetHostSourceIdsKey); ok {
		list := hostSourceIdsVal.(*schema.Set).List()
		hostSourceIds = make([]string, 0, len(list))
		for _, i := range list {
			hostSourceIds = append(hostSourceIds, i.(string))
		}
	}

	var credentialSourceIds []string
	if credentialSourceIdsVal, ok := d.GetOk(targetApplicationCredentialSourceIdsKey); ok {
		list := credentialSourceIdsVal.(*schema.Set).List()
		credentialSourceIds = make([]string, 0, len(list))
		for _, i := range list {
			credentialSourceIds = append(credentialSourceIds, i.(string))
		}
	}

	workerFilterVal, ok := d.GetOk(targetWorkerFilterKey)
	if ok {
		workerFilterStr := workerFilterVal.(string)
		opts = append(opts, targets.WithWorkerFilter(workerFilterStr))
	}

	tc := targets.NewClient(md.client)

	tcr, err := tc.Create(ctx, typeStr, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating target: %v", err)
	}
	if tcr == nil {
		return diag.Errorf("target nil after create")
	}
	raw := tcr.GetResponse().Map

	version := tcr.Item.Version
	if hostSourceIds != nil {
		tur, err := tc.SetHostSources(ctx, tcr.Item.Id, version, hostSourceIds)
		if err != nil {
			return diag.Errorf("error setting host sources on target: %v", err)
		}
		raw = tur.GetResponse().Map
		version = tur.Item.Version
	}

	if credentialSourceIds != nil {
		tur, err := tc.SetCredentialSources(ctx, tcr.Item.Id, version, targets.WithApplicationCredentialSourceIds(credentialSourceIds))
		if err != nil {
			return diag.Errorf("error setting credential sources on target: %v", err)
		}
		raw = tur.GetResponse().Map
	}

	if err := setFromTargetResponseMap(d, raw); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceTargetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	tc := targets.NewClient(md.client)

	trr, err := tc.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading target: %v", err)
	}
	if trr == nil {
		return diag.Errorf("target nil after read")
	}

	if err := setFromTargetResponseMap(d, trr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceTargetUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	tc := targets.NewClient(md.client)

	var opts []targets.Option

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

	var workerFilter *string
	if d.HasChange(targetWorkerFilterKey) {
		opts = append(opts, targets.DefaultWorkerFilter())
		workerFilterVal, ok := d.GetOk(targetWorkerFilterKey)
		if ok {
			workerFilterStr := workerFilterVal.(string)
			workerFilter = &workerFilterStr
			opts = append(opts, targets.WithWorkerFilter(workerFilterStr))
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
		if err := d.Set(NameKey, name); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(DescriptionKey) {
		if err := d.Set(DescriptionKey, desc); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(targetDefaultPortKey) {
		if err := d.Set(targetDefaultPortKey, defaultPort); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(targetSessionMaxSecondsKey) {
		if err := d.Set(targetSessionMaxSecondsKey, sessionMaxSeconds); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(targetSessionConnectionLimitKey) {
		if err := d.Set(targetSessionConnectionLimitKey, sessionConnectionLimit); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(targetWorkerFilterKey) {
		if err := d.Set(targetWorkerFilterKey, workerFilter); err != nil {
			return diag.FromErr(err)
		}
	}

	// The above call may not actually happen, so we use d.Id() and automatic
	// versioning here
	if d.HasChange(targetHostSourceIdsKey) {
		var hostSourceIds []string
		if hostSourceIdsVal, ok := d.GetOk(targetHostSourceIdsKey); ok {
			hostSources := hostSourceIdsVal.(*schema.Set).List()
			for _, hostSource := range hostSources {
				hostSourceIds = append(hostSourceIds, hostSource.(string))
			}
		}
		_, err := tc.SetHostSources(ctx, d.Id(), 0, hostSourceIds, targets.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating host sources in target: %v", err)
		}
		if err := d.Set(targetHostSourceIdsKey, hostSourceIds); err != nil {
			return diag.FromErr(err)
		}
	}

	// The above calls may not actually happen, so we use d.Id() and automatic
	// versioning here
	if d.HasChange(targetApplicationCredentialSourceIdsKey) {
		var credentialSourceIds []string
		if credentialSourceIdsVal, ok := d.GetOk(targetApplicationCredentialSourceIdsKey); ok {
			credSourceIds := credentialSourceIdsVal.(*schema.Set).List()
			for _, credSourceId := range credSourceIds {
				credentialSourceIds = append(credentialSourceIds, credSourceId.(string))
			}
		}

		opts := []targets.Option{
			targets.WithAutomaticVersioning(true),
			targets.DefaultApplicationCredentialSourceIds(),
		}
		if len(credentialSourceIds) > 0 {
			opts = append(opts, targets.WithApplicationCredentialSourceIds(credentialSourceIds))
		}

		_, err := tc.SetCredentialSources(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating credential sources in target: %v", err)
		}
		if err := d.Set(targetApplicationCredentialSourceIdsKey, credentialSourceIds); err != nil {
			return diag.FromErr(err)
		}
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
