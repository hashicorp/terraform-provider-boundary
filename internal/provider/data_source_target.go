// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/targets"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceTarget() *schema.Resource {
	return &schema.Resource{
		Description: "The boundary_target data source allows you to find a Boundary target.",
		ReadContext: dataSourceTargetRead,

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the target.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The target name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Required:    true,
			},
			DescriptionKey: {
				Description: "The target description.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			TypeKey: {
				Description: "The target resource type.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.",
				Type:        schema.TypeString,
				Required:    true,
			},
			targetDefaultPortKey: {
				Description: "The default port for this target.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			targetDefaultClientPortKey: {
				Description: "The default client port for this target.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			targetHostSourceIdsKey: {
				Description: "A list of host source ID's. Cannot be used alongside address.",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			targetBrokeredCredentialSourceIdsKey: {
				Description: "A list of brokered credential source ID's.",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			targetInjectedAppCredentialSourceIdsKey: {
				Description: "A list of injected application credential source ID's.",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			targetSessionMaxSecondsKey: {
				Type:     schema.TypeInt,
				Computed: true,
			},
			targetSessionConnectionLimitKey: {
				Type:     schema.TypeInt,
				Computed: true,
			},
			targetWorkerFilterKey: {
				Description: "Boolean expression to filter the workers for this target",
				Type:        schema.TypeString,
				Computed:    true,
				Deprecated:  "Deprecated. Use `egress_worker_filter` and `ingress_worker_filter` instead",
			},
			targetWorkerEgressFilterKey: {
				Description: "Boolean expression to filter the workers used to access this target",
				Type:        schema.TypeString,
				Computed:    true,
			},
			targetWorkerIngressFilterKey: {
				Description: "HCP Only. Boolean expression to filter the workers a user will connect to when initiating a session against this target",
				Type:        schema.TypeString,
				Computed:    true,
			},
			targetAddressKey: {
				Description: "Optionally, a valid network address to connect to for this target. Cannot be used alongside host_source_ids.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			targetEnableSessionRecordingKey: {
				Description: "HCP/Ent Only. Enable sessions recording for this target. Only applicable for SSH and RDP targets.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			targetStorageBucketIdKey: {
				Description: "HCP/Ent Only. Storage bucket for this target. Only applicable for SSH and RDP targets.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceTargetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	name := d.Get(NameKey).(string)
	scope_id := d.Get(ScopeIdKey).(string)

	targ := targets.NewClient(md.client)
	targetItems, err := targ.List(
		ctx, scope_id,
		targets.WithFilter(FilterWithItemNameMatches(name)),
	)
	if err != nil {
		return diag.Errorf("error calling list target: %v", err)
	}
	targetList := targetItems.GetItems()
	if targetList == nil {
		return diag.Errorf("no targets found")
	}
	if len(targetList) == 0 {
		return diag.Errorf("no matching target found")
	}
	if len(targetList) > 1 {
		return diag.Errorf("error found more than 1 target")
	}

	arr, err := targ.Read(ctx, targetList[0].Id)
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read target: %v", err)
	}
	if arr == nil {
		return diag.Errorf("target nil after read")
	}

	if err := setFromTargetResponseMap(d, arr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
