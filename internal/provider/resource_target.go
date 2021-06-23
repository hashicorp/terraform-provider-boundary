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
	targetCredentialLibraryIdsKey   = "credential_library_ids"
	targetDefaultPortKey            = "default_port"
	targetSessionMaxSecondsKey      = "session_max_seconds"
	targetSessionConnectionLimitKey = "session_connection_limit"
	targetWorkerFilterKey           = "worker_filter"

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
			targetCredentialLibraryIdsKey: {
				Description: "A list of credential library ID's.",
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
	if err := d.Set(targetHostSetIdsKey, raw["host_set_ids"]); err != nil {
		return err
	}
	if err := d.Set(targetCredentialLibraryIdsKey, raw["credential_library_ids"]); err != nil {
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

	var credentialLibraryIds []string
	if credentialLibraryIdsVal, ok := d.GetOk(targetCredentialLibraryIdsKey); ok {
		list := credentialLibraryIdsVal.(*schema.Set).List()
		credentialLibraryIds = make([]string, 0, len(list))
		for _, i := range list {
			credentialLibraryIds = append(credentialLibraryIds, i.(string))
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
	if hostSetIds != nil {
		tur, err := tc.SetHostSets(ctx, tcr.Item.Id, version, hostSetIds)
		if err != nil {
			return diag.Errorf("error setting host sets on target: %v", err)
		}
		raw = tur.GetResponse().Map
		version = tur.Item.Version
	}

	if credentialLibraryIds != nil {
		tur, err := tc.SetCredentialLibraries(ctx, tcr.Item.Id, version, credentialLibraryIds)
		if err != nil {
			return diag.Errorf("error setting credential libraries on target: %v", err)
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
		if err := d.Set(targetHostSetIdsKey, hostSetIds); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(targetCredentialLibraryIdsKey) {
		var credentialLibraryIds []string
		if credentialLibraryIdsVal, ok := d.GetOk(targetCredentialLibraryIdsKey); ok {
			credLibsIds := credentialLibraryIdsVal.(*schema.Set).List()
			for _, credLibId := range credLibsIds {
				credentialLibraryIds = append(credentialLibraryIds, credLibId.(string))
			}
		}
		_, err := tc.SetCredentialLibraries(ctx, d.Id(), 0, credentialLibraryIds, targets.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating credential libraries in target: %v", err)
		}
		if err := d.Set(targetCredentialLibraryIdsKey, credentialLibraryIds); err != nil {
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
