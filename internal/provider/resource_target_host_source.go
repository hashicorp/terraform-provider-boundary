// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/targets"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	targetHostSourceTargetIdKey = "target_id"
	targetHostSourceIdKey       = "host_source_id"
	targetHostSourceDescription = "Attaches a host source to an existing Boundary target. " +
		"Use this resource instead of the `host_source_ids` field on `boundary_target` " +
		"when the target and its host sources are managed in separate Terraform configurations. " +
		"Do not set `host_source_ids` on `boundary_target` when using this resource for the same target."
)

func resourceTargetHostSource() *schema.Resource {
	return &schema.Resource{
		Description: targetHostSourceDescription,

		CreateContext: resourceTargetHostSourceCreate,
		ReadContext:   resourceTargetHostSourceRead,
		DeleteContext: resourceTargetHostSourceDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			targetHostSourceTargetIdKey: {
				Description: "The ID of the target to attach the host source to.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			targetHostSourceIdKey: {
				Description: "The ID of the host source to attach to the target.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceTargetHostSourceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	tc := targets.NewClient(md.client)

	targetId := d.Get(targetHostSourceTargetIdKey).(string)
	hostSourceId := d.Get(targetHostSourceIdKey).(string)

	// Read the target to get its current host source list.
	trr, err := tc.Read(ctx, targetId)
	if err != nil {
		return diag.Errorf("error reading target: %v", err)
	}
	if trr == nil {
		return diag.Errorf("target nil after read")
	}

	hostSourceExists := false
	for _, id := range trr.Item.HostSourceIds {
		if id == hostSourceId {
			hostSourceExists = true
			break
		}
	}

	if !hostSourceExists {
		if _, err := tc.AddHostSources(ctx, targetId, 0, []string{hostSourceId}, targets.WithAutomaticVersioning(true)); err != nil {
			return diag.Errorf("error adding host source to target: %v", err)
		}
	}

	d.SetId(fmt.Sprintf("%s:%s", hostSourceId, targetId))
	if err := d.Set(targetHostSourceTargetIdKey, targetId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(targetHostSourceIdKey, hostSourceId); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceTargetHostSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	tc := targets.NewClient(md.client)

	targetId, hostSourceId, err := targetAndHostSourceIdsFromResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}

	trr, err := tc.Read(ctx, targetId)
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

	// If this host source is no longer attached, remove from state.
	for _, id := range trr.Item.HostSourceIds {
		if id == hostSourceId {
			if err := d.Set(targetHostSourceTargetIdKey, targetId); err != nil {
				return diag.FromErr(err)
			}
			if err := d.Set(targetHostSourceIdKey, hostSourceId); err != nil {
				return diag.FromErr(err)
			}
			return nil
		}
	}

	d.SetId("")
	return nil
}

func resourceTargetHostSourceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	tc := targets.NewClient(md.client)

	targetId, hostSourceId, err := targetAndHostSourceIdsFromResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}

	// Read the current list so we can remove just this one ID.
	trr, err := tc.Read(ctx, targetId)
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			return nil // target is already gone
		}
		return diag.Errorf("error reading target before detach: %v", err)
	}
	if trr == nil {
		return nil
	}

	found := false
	for _, id := range trr.Item.HostSourceIds {
		if id == hostSourceId {
			found = true
		}
	}
	if !found {
		return nil // host source is already gone
	}

	if _, err := tc.RemoveHostSources(ctx, targetId, 0, []string{hostSourceId}, targets.WithAutomaticVersioning(true)); err != nil {
		return diag.Errorf("error removing host source from target: %v", err)
	}

	return nil
}

// targetAndHostSourceIdsFromResourceData returns the target ID and host source ID
// from either the resource attributes (normal CRUD) or the composite resource ID (import).
func targetAndHostSourceIdsFromResourceData(d *schema.ResourceData) (targetId, hostSourceId string, err error) {
	if t, ok := d.GetOk(targetHostSourceTargetIdKey); ok {
		targetId = t.(string)
	}
	if h, ok := d.GetOk(targetHostSourceIdKey); ok {
		hostSourceId = h.(string)
	}
	if targetId != "" && hostSourceId != "" {
		return targetId, hostSourceId, nil
	}

	// Fall back to parsing the composite ID — this path is taken on import.
	parts := strings.SplitN(d.Id(), ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid resource ID %q: expected format <host_source_id>:<target_id>", d.Id())
	}
	return parts[1], parts[0], nil
}
