// Copyright IBM Corp. 2020, 2026
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
	aliasSuffixKey = "alias_suffix"
)

func resourceScopeAliasSuffix() *schema.Resource {
	return &schema.Resource{
		Description: "The scope alias suffix resource allows you to set an alias suffix for an org or project scope. " +
			"Global scopes are not valid for alias suffix operations.",

		CreateContext:        resourceScopeAliasSuffixCreate,
		UpdateWithoutTimeout: resourceScopeAliasSuffixUpdate,
		ReadWithoutTimeout:   resourceScopeAliasSuffixRead,
		DeleteWithoutTimeout: resourceScopeAliasSuffixDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			ScopeIdKey: {
				Description: "The scope ID. Alias suffixes are supported for org and project scopes.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			aliasSuffixKey: {
				Description: "The alias suffix value for this scope.",
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}

func resourceScopeAliasSuffixCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)
	opts := []scopes.Option{scopes.WithAutomaticVersioning(true)}

	scopeId, err := scopeIdFromAliasSuffixResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := validateAliasSuffixScope(scopeId); err != nil {
		return diag.FromErr(err)
	}

	var aliasSuffix string
	if aliasSuffixVal, ok := d.GetOk(aliasSuffixKey); ok {
		aliasSuffix = aliasSuffixVal.(string)
	} else {
		return diag.Errorf("no alias suffix provided")
	}

	if _, err := scp.SetAliasSuffix(ctx, scopeId, 0, aliasSuffix, opts...); err != nil {
		return diag.FromErr(err)
	}

	if err := setScopeAliasSuffixState(d, scopeId, aliasSuffix); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceScopeAliasSuffixRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)

	scopeId, err := scopeIdFromAliasSuffixResourceData(d)
	if err != nil {
		return diag.FromErr(err)
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
	if s.GetResponse() == nil {
		return diag.Errorf("scope response nil after read")
	}

	serverAliasSuffix, err := aliasSuffixFromScopeResponseMap(s.GetResponse().Map)
	if err != nil {
		return diag.FromErr(err)
	}

	if serverAliasSuffix == "" {
		// no alias suffix is currently set on this scope, destroy this resource
		d.SetId("")
		return nil
	}

	if err := setScopeAliasSuffixState(d, scopeId, serverAliasSuffix); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceScopeAliasSuffixUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)
	opts := []scopes.Option{scopes.WithAutomaticVersioning(true)}

	scopeId, err := scopeIdFromAliasSuffixResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := validateAliasSuffixScope(scopeId); err != nil {
		return diag.FromErr(err)
	}

	var aliasSuffix string
	if aliasSuffixVal, ok := d.GetOk(aliasSuffixKey); ok {
		aliasSuffix = aliasSuffixVal.(string)
	} else {
		return diag.Errorf("no alias suffix provided")
	}

	if _, err := scp.SetAliasSuffix(ctx, scopeId, 0, aliasSuffix, opts...); err != nil {
		return diag.FromErr(err)
	}

	if err := setScopeAliasSuffixState(d, scopeId, aliasSuffix); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceScopeAliasSuffixDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)
	opts := []scopes.Option{scopes.WithAutomaticVersioning(true)}

	scopeId, err := scopeIdFromAliasSuffixResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := validateAliasSuffixScope(scopeId); err != nil {
		return diag.FromErr(err)
	}

	if _, err := scp.RemoveAliasSuffix(ctx, scopeId, 0, opts...); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func aliasSuffixFromScopeResponseMap(raw map[string]interface{}) (string, error) {
	if raw == nil {
		return "", fmt.Errorf("scope response map nil after read")
	}

	aliasSuffixRaw, ok := raw[aliasSuffixKey]
	if !ok || aliasSuffixRaw == nil {
		return "", nil
	}

	aliasSuffix, ok := aliasSuffixRaw.(string)
	if !ok {
		return "", fmt.Errorf("alias suffix in scope response has unexpected type %T", aliasSuffixRaw)
	}

	return aliasSuffix, nil
}

func setScopeAliasSuffixState(d *schema.ResourceData, scopeId, aliasSuffix string) error {
	if err := d.Set(ScopeIdKey, scopeId); err != nil {
		return err
	}
	if err := d.Set(aliasSuffixKey, aliasSuffix); err != nil {
		return err
	}

	// One alias suffix may be set per scope, so scope ID is the stable TF ID.
	d.SetId(scopeId)

	return nil
}

func validateAliasSuffixScope(scopeId string) error {
	if scopeId == globalScopeId {
		return fmt.Errorf("alias suffixes are not supported for the global scope; use an org or project scope ID")
	}
	return nil
}

func scopeIdFromAliasSuffixResourceData(d *schema.ResourceData) (string, error) {
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId := scopeIdVal.(string)
		if scopeId != "" {
			return scopeId, nil
		}
	}

	id := d.Id()
	if id == "" {
		return "", fmt.Errorf("no scope ID provided")
	}

	// Backward-compatibility for any prior state that used alias_suffix:scope_id.
	if index := strings.LastIndex(id, ":"); index >= 0 && index < len(id)-1 {
		return id[index+1:], nil
	}

	return id, nil
}
