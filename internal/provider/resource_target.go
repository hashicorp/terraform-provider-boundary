// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	targetBrokeredCredentialSourceIdsKey    = "brokered_credential_source_ids"
	targetInjectedAppCredentialSourceIdsKey = "injected_application_credential_source_ids"
	targetDefaultPortKey                    = "default_port"
	targetSessionMaxSecondsKey              = "session_max_seconds"
	targetSessionConnectionLimitKey         = "session_connection_limit"
	targetWorkerFilterKey                   = "worker_filter"
	targetWorkerEgressFilterKey             = "egress_worker_filter"
	targetWorkerIngressFilterKey            = "ingress_worker_filter"

	targetTypeTcp = "tcp"
	targetTypeSsh = "ssh"
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
			targetBrokeredCredentialSourceIdsKey: {
				Description: "A list of brokered credential source ID's.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			targetInjectedAppCredentialSourceIdsKey: {
				Description: "A list of injected application credential source ID's.",
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
				Deprecated:  "Deprecated. Use `egress_worker_filter` and `ingress_worker_filter` instead",
			},
			targetWorkerEgressFilterKey: {
				Description: "Boolean expression to filter the workers used to access this target",
				Type:        schema.TypeString,
				Optional:    true,
			},
			targetWorkerIngressFilterKey: {
				Description: "HCP Only. Boolean expression to filter the workers a user will connect to when initiating a session against this target",
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
	if err := d.Set(targetBrokeredCredentialSourceIdsKey, raw["brokered_credential_source_ids"]); err != nil {
		return err
	}
	if err := d.Set(targetInjectedAppCredentialSourceIdsKey, raw["injected_application_credential_source_ids"]); err != nil {
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
	if err := d.Set(targetWorkerEgressFilterKey, raw["egress_worker_filter"]); err != nil {
		return err
	}
	if err := d.Set(targetWorkerIngressFilterKey, raw["ingress_worker_filter"]); err != nil {
		return err
	}

	switch raw["type"].(string) {
	case targetTypeTcp, targetTypeSsh:
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

func resourceTargetCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) (errs diag.Diagnostics) {
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
	case targetTypeTcp, targetTypeSsh:
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
		if defaultPortInt < 0 || defaultPortInt > 65535 {
			return diag.Errorf(`"default_port" must be a valid tcp port`)
		}
		switch typeStr {
		case targetTypeTcp:
			opts = append(opts, targets.WithTcpTargetDefaultPort(uint32(defaultPortInt)))
		case targetTypeSsh:
			opts = append(opts, targets.WithSshTargetDefaultPort(uint32(defaultPortInt)))
		}
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

	var brokeredCreds []string
	if credentialSourceIdsVal, ok := d.GetOk(targetBrokeredCredentialSourceIdsKey); ok {
		list := credentialSourceIdsVal.(*schema.Set).List()
		brokeredCreds = make([]string, 0, len(list))
		for _, i := range list {
			brokeredCreds = append(brokeredCreds, i.(string))
		}
	}

	var injectedCreds []string
	if credentialSourceIdsVal, ok := d.GetOk(targetInjectedAppCredentialSourceIdsKey); ok {
		list := credentialSourceIdsVal.(*schema.Set).List()
		injectedCreds = make([]string, 0, len(list))
		for _, i := range list {
			injectedCreds = append(injectedCreds, i.(string))
		}
	}

	workerFilterVal, ok := d.GetOk(targetWorkerFilterKey)
	if ok {
		workerFilterStr := workerFilterVal.(string)
		opts = append(opts, targets.WithWorkerFilter(workerFilterStr))
	}

	workerEgressFilterVal, ok := d.GetOk(targetWorkerEgressFilterKey)
	if ok {
		workerEgressFilterStr := workerEgressFilterVal.(string)
		opts = append(opts, targets.WithEgressWorkerFilter(workerEgressFilterStr))
	}

	workerIngressFilterVal, ok := d.GetOk(targetWorkerIngressFilterKey)
	if ok {
		workerIngressFilterStr := workerIngressFilterVal.(string)
		opts = append(opts, targets.WithIngressWorkerFilter(workerIngressFilterStr))
	}

	tc := targets.NewClient(md.client)
	tcr, err := tc.Create(ctx, typeStr, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating target: %v", err)
	}
	if tcr == nil {
		return diag.Errorf("target nil after create")
	}
	apiResponse := tcr.GetResponse().Map
	defer func() {
		if err := setFromTargetResponseMap(d, apiResponse); err != nil {
			errs = append(errs, diag.FromErr(err)...)
		}
	}()

	version := tcr.Item.Version
	if hostSourceIds != nil {
		tur, err := tc.SetHostSources(ctx, tcr.Item.Id, version, hostSourceIds)
		if err != nil {
			return diag.Errorf("error setting host sources on target: %v", err)
		}
		if tur == nil {
			return diag.Errorf("nil target after setting host sources")
		}
		apiResponse = tur.GetResponse().Map
		version = tur.Item.Version
	}

	var credOpts []targets.Option
	if brokeredCreds != nil {
		credOpts = append(credOpts, targets.WithBrokeredCredentialSourceIds(brokeredCreds))
	}
	if injectedCreds != nil {
		credOpts = append(credOpts, targets.WithInjectedApplicationCredentialSourceIds(injectedCreds))
	}
	if len(credOpts) > 0 {
		tur, err := tc.SetCredentialSources(ctx, tcr.Item.Id, version, credOpts...)
		if err != nil {
			return diag.Errorf("error setting credential sources on target: %v", err)
		}
		if tur == nil {
			return diag.Errorf("nil target after setting credential sources")
		}
		apiResponse = tur.GetResponse().Map
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

	var typeStr string
	if typeVal, ok := d.GetOk(TypeKey); ok {
		typeStr = typeVal.(string)
	}
	switch typeStr {
	case targetTypeTcp, targetTypeSsh:
	default:
		return diag.Errorf("invalid type provided")
	}

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
		switch typeStr {
		case targetTypeTcp:
			opts = append(opts, targets.DefaultTcpTargetDefaultPort())
			defaultPortVal, ok := d.GetOk(targetDefaultPortKey)
			if ok {
				defaultPortInt := defaultPortVal.(int)
				if defaultPortInt < 0 || defaultPortInt > 65535 {
					return diag.Errorf(`"default_port" must be a valid tcp port`)
				}
				defaultPort = &defaultPortInt
				opts = append(opts, targets.WithTcpTargetDefaultPort(uint32(defaultPortInt)))
			}

		case targetTypeSsh:
			opts = append(opts, targets.DefaultSshTargetDefaultPort())
			defaultPortVal, ok := d.GetOk(targetDefaultPortKey)
			if ok {
				defaultPortInt := defaultPortVal.(int)
				if defaultPortInt < 0 || defaultPortInt > 65535 {
					return diag.Errorf(`"default_port" must be a valid tcp port`)
				}
				defaultPort = &defaultPortInt
				opts = append(opts, targets.WithSshTargetDefaultPort(uint32(defaultPortInt)))
			}
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

	var workerEgressFilter *string
	if d.HasChange(targetWorkerEgressFilterKey) {
		opts = append(opts, targets.DefaultEgressWorkerFilter())
		workerEgressFilterVal, ok := d.GetOk(targetWorkerEgressFilterKey)
		if ok {
			workerEgressFilterStr := workerEgressFilterVal.(string)
			workerEgressFilter = &workerEgressFilterStr
			opts = append(opts, targets.WithEgressWorkerFilter(workerEgressFilterStr))
		}
	}

	var workerIngressFilter *string
	if d.HasChange(targetWorkerIngressFilterKey) {
		opts = append(opts, targets.DefaultIngressWorkerFilter())
		workerIngressFilterVal, ok := d.GetOk(targetWorkerIngressFilterKey)
		if ok {
			workerIngressFilterStr := workerIngressFilterVal.(string)
			workerIngressFilter = &workerIngressFilterStr
			opts = append(opts, targets.WithIngressWorkerFilter(workerIngressFilterStr))
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
	if d.HasChange(targetWorkerEgressFilterKey) {
		if err := d.Set(targetWorkerEgressFilterKey, workerEgressFilter); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(targetWorkerIngressFilterKey) {
		if err := d.Set(targetWorkerIngressFilterKey, workerIngressFilter); err != nil {
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

	// if any of the credential types are changed, then all credential ids must be gathered
	// because the SetCredentialSources function will remove ids that are not present.
	if d.HasChange(targetBrokeredCredentialSourceIdsKey) || d.HasChange(targetInjectedAppCredentialSourceIdsKey) {
		credOpts := []targets.Option{
			targets.WithAutomaticVersioning(true),
		}

		if credentialSourceIdsVal, ok := d.GetOk(targetBrokeredCredentialSourceIdsKey); ok {
			var credentialSourceIds []string
			credSourceIds := credentialSourceIdsVal.(*schema.Set).List()
			for _, credSourceId := range credSourceIds {
				credentialSourceIds = append(credentialSourceIds, credSourceId.(string))
			}
			credOpts = append(credOpts, targets.WithBrokeredCredentialSourceIds(credentialSourceIds))
		}

		if credentialSourceIdsVal, ok := d.GetOk(targetInjectedAppCredentialSourceIdsKey); ok {
			var credentialSourceIds []string
			credSourceIds := credentialSourceIdsVal.(*schema.Set).List()
			for _, credSourceId := range credSourceIds {
				credentialSourceIds = append(credentialSourceIds, credSourceId.(string))
			}
			credOpts = append(credOpts, targets.WithInjectedApplicationCredentialSourceIds(credentialSourceIds))
		}

		result, err := tc.SetCredentialSources(ctx, d.Id(), 0, credOpts...)
		if err != nil {
			return diag.Errorf("error updating credential sources in target: %v", err)
		}

		if d.HasChange(targetBrokeredCredentialSourceIdsKey) {
			if err := d.Set(targetBrokeredCredentialSourceIdsKey, result.Item.BrokeredCredentialSourceIds); err != nil {
				return diag.FromErr(err)
			}
		}

		if d.HasChange(targetInjectedAppCredentialSourceIdsKey) {
			if err := d.Set(targetInjectedAppCredentialSourceIdsKey, result.Item.InjectedApplicationCredentialSourceIds); err != nil {
				return diag.FromErr(err)
			}
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
