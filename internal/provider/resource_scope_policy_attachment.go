// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	policyIdKey         = "policy_id"
	storagePolicyPrefix = "pst_"
)

func resourceScopePolicyAttachment() *schema.Resource {
	return &schema.Resource{
		CreateContext:        resourceScopePolicyAttachmentCreate,
		ReadWithoutTimeout:   resourceScopePolicyAttachmentRead,
		DeleteWithoutTimeout: resourceScopePolicyAttachmentDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			ScopeIdKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			policyIdKey: {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
		},
	}
}

func resourceScopePolicyAttachmentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)
	opts := []scopes.Option{scopes.WithAutomaticVersioning(true)}

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	if err := d.Set(ScopeIdKey, scopeId); err != nil {
		return diag.FromErr(err)
	}

	var policyId string
	if policyIdVal, ok := d.GetOk(policyIdKey); ok {
		policyId = policyIdVal.(string)
	} else {
		return diag.Errorf("no policy ID provided")
	}

	// other policy types may be added here and in other switches with this file
	switch {
	case strings.HasPrefix(policyId, storagePolicyPrefix):
		if _, err := scp.AttachStoragePolicy(ctx, scopeId, 0, policyId, opts...); err != nil {
			return diag.FromErr(err)
		}
	default:
		return diag.Errorf("unknown policy type provided.")
	}

	if err := d.Set(policyIdKey, policyId); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s:%s", policyId, scopeId))

	return nil
}

func resourceScopePolicyAttachmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	var policyId string
	if policyIdVal, ok := d.GetOk(policyIdKey); ok {
		policyId = policyIdVal.(string)
	} else {
		return diag.Errorf("no policy ID provided")
	}

	s, err := scp.Read(ctx, scopeId)
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			// the scope is gone, destroy this resource
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading scope: %v", err)
	}
	if s == nil {
		return diag.Errorf("scope nil after read")
	}

	switch {
	case strings.HasPrefix(policyId, storagePolicyPrefix):
		if policyId != s.GetItem().StoragePolicyId {
			// this policy is no longer attached to the scope, destroy this resource
			d.SetId("")
			return nil
		}
	default:
		return diag.Errorf("unknown policy type provided.")
	}

	return nil
}

func resourceScopePolicyAttachmentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)
	opts := []scopes.Option{scopes.WithAutomaticVersioning(true)}

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	var policyId string
	if policyIdVal, ok := d.GetOk(policyIdKey); ok {
		policyId = policyIdVal.(string)
	} else {
		return diag.Errorf("no policy ID provided")
	}

	switch {
	case strings.HasPrefix(policyId, storagePolicyPrefix):
		if _, err := scp.DetachStoragePolicy(ctx, scopeId, 0, opts...); err != nil {
			return diag.FromErr(err)
		}
	default:
		return diag.Errorf("unknown policy type provided.")
	}

	return nil
}
