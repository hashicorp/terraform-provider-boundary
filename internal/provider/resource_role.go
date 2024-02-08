// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/roles"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	roleGrantScopeIdKey  = "grant_scope_id"
	roleGrantScopeIdsKey = "grant_scope_ids"
	rolePrincipalIdsKey  = "principal_ids"
	roleGrantStringsKey  = "grant_strings"
)

func resourceRole() *schema.Resource {
	return &schema.Resource{
		Description: "The role resource allows you to configure a Boundary role.",

		CreateContext: resourceRoleCreate,
		ReadContext:   resourceRoleRead,
		UpdateContext: resourceRoleUpdate,
		DeleteContext: resourceRoleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the role.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The role name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The role description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			rolePrincipalIdsKey: {
				Description: "A list of principal (user or group) IDs to add as principals on the role.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			roleGrantStringsKey: {
				Description: "A list of stringified grants for the role.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			roleGrantScopeIdKey: {
				Description: "For Boundary 0.15+, use `grant_scope_ids` instead. The scope for which the grants in the role should apply.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Deprecated:  "In Boundary 0.15+, please use `grant_scope_ids` instead. This field will be removed in a future release.",
			},
			roleGrantScopeIdsKey: {
				Description: `A list of scopes for which the grants in this role should apply, which can include the special values "this", "children", or "descendants"`,
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
			},
		},
	}
}

func setFromRoleResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw["scope_id"]); err != nil {
		return err
	}
	if err := d.Set(rolePrincipalIdsKey, raw["principal_ids"]); err != nil {
		return err
	}
	if err := d.Set(roleGrantStringsKey, raw["grant_strings"]); err != nil {
		return err
	}
	if err := d.Set(roleGrantScopeIdsKey, raw["grant_scope_ids"]); err != nil {
		return err
	}
	if err := d.Set(roleGrantScopeIdKey, raw["grant_scope_id"]); err != nil {
		return err
	}

	d.SetId(raw["id"].(string))
	return nil
}

func resourceRoleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) (errs diag.Diagnostics) {
	md := meta.(*metaData)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []roles.Option{}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, roles.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, roles.WithDescription(descStr))
	}

	grantScopeIdVal := d.Get(roleGrantScopeIdKey)
	if grantScopeIdVal != "" {
		grantScopeIdStr := grantScopeIdVal.(string)
		opts = append(opts, roles.WithGrantScopeId(grantScopeIdStr))
	}
	var grantScopeIds []string
	if grantScopeIdsVal, ok := d.GetOk(roleGrantScopeIdsKey); ok {
		list := grantScopeIdsVal.(*schema.Set).List()
		grantScopeIds = make([]string, 0, len(list))
		for _, i := range list {
			grantScopeIds = append(grantScopeIds, i.(string))
		}
	}

	var principalIds []string
	if principalIdsVal, ok := d.GetOk(rolePrincipalIdsKey); ok {
		list := principalIdsVal.(*schema.Set).List()
		principalIds = make([]string, 0, len(list))
		for _, i := range list {
			principalIds = append(principalIds, i.(string))
		}
	}

	var grantStrings []string
	if grantStringsVal, ok := d.GetOk(roleGrantStringsKey); ok {
		list := grantStringsVal.(*schema.Set).List()
		grantStrings = make([]string, 0, len(list))
		for _, i := range list {
			grantStrings = append(grantStrings, i.(string))
		}
	}

	rc := roles.NewClient(md.client)

	tcr, err := rc.Create(ctx, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error calling create role: %v", err)
	}
	if tcr == nil {
		return diag.Errorf("nil role after create")
	}
	apiResponse := tcr.GetResponse().Map
	defer func() {
		if err := setFromRoleResponseMap(d, apiResponse); err != nil {
			errs = append(errs, diag.FromErr(err)...)
		}
	}()

	if principalIds != nil {
		tspr, err := rc.SetPrincipals(ctx, tcr.Item.Id, 0, principalIds, roles.WithAutomaticVersioning(true))
		switch {
		case err != nil:
			errs = append(errs, diag.Diagnostic{Severity: diag.Error, Summary: "error setting principals", Detail: err.Error()})
		case tspr == nil:
			errs = append(errs, diag.Diagnostic{Severity: diag.Error, Summary: "nil role after setting principals"})
		default:
			apiResponse = tspr.GetResponse().Map
		}
	}

	if grantStrings != nil {
		tsgr, err := rc.SetGrants(ctx, tcr.Item.Id, 0, grantStrings, roles.WithAutomaticVersioning(true))
		switch {
		case err != nil:
			errs = append(errs, diag.Diagnostic{Severity: diag.Error, Summary: "error setting grants", Detail: err.Error()})
		case tsgr == nil:
			errs = append(errs, diag.Diagnostic{Severity: diag.Error, Summary: "nil role after setting grants"})
		default:
			apiResponse = tsgr.GetResponse().Map
		}
	}

	if grantScopeIds != nil {
		tsgr, err := rc.SetGrantScopes(ctx, tcr.Item.Id, 0, grantScopeIds, roles.WithAutomaticVersioning(true))
		switch {
		case err != nil:
			errs = append(errs, diag.Diagnostic{Severity: diag.Error, Summary: "error setting grant scopes", Detail: err.Error()})
		case tsgr == nil:
			errs = append(errs, diag.Diagnostic{Severity: diag.Error, Summary: "nil role after setting grant scope ids"})
		default:
			apiResponse = tsgr.GetResponse().Map
		}
	}

	return errs
}

func resourceRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	rc := roles.NewClient(md.client)

	trr, err := rc.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read role: %v", err)
	}
	if trr == nil {
		return diag.Errorf("role nil after read")
	}

	if err := setFromRoleResponseMap(d, trr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceRoleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	rc := roles.NewClient(md.client)

	opts := []roles.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, roles.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, roles.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, roles.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, roles.WithDescription(descStr))
		}
	}

	var grantScopeId *string
	if d.HasChange(roleGrantScopeIdKey) {
		// If it is an update to use the multi-value grant_scope_ids, and this
		// has changed to "", then don't set it on the API. Instead let it fall
		// through and below it'll be updated to be "".
		if !d.HasChange(roleGrantScopeIdsKey) {
			opts = append(opts, roles.DefaultGrantScopeId())
			grantScopeIdVal, ok := d.GetOk(roleGrantScopeIdKey)
			if ok {
				grantScopeIdStr := grantScopeIdVal.(string)
				grantScopeId = &grantScopeIdStr
				opts = append(opts, roles.WithGrantScopeId(grantScopeIdStr))
			}
		}
	}

	if len(opts) > 0 {
		opts = append(opts, roles.WithAutomaticVersioning(true))
		_, err := rc.Update(ctx, d.Id(), 0, opts...)
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
	if d.HasChange(roleGrantScopeIdKey) {
		if err := d.Set(roleGrantScopeIdKey, grantScopeId); err != nil {
			return diag.FromErr(err)
		}
	}

	var diags diag.Diagnostics
	if d.HasChange(roleGrantStringsKey) {
		var grantStrings []string
		if grantStringsVal, ok := d.GetOk(roleGrantStringsKey); ok {
			grants := grantStringsVal.(*schema.Set).List()
			for _, grant := range grants {
				grantStrings = append(grantStrings, grant.(string))
			}
		}
		_, err := rc.SetGrants(ctx, d.Id(), 0, grantStrings, roles.WithAutomaticVersioning(true))
		if err != nil {
			diags = append(diags, diag.Diagnostic{Severity: diag.Error, Summary: "error setting grants", Detail: err.Error()})
		} else {
			if err := d.Set(roleGrantStringsKey, grantStrings); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange(rolePrincipalIdsKey) {
		var principalIds []string
		if principalIdsVal, ok := d.GetOk(rolePrincipalIdsKey); ok {
			principals := principalIdsVal.(*schema.Set).List()
			for _, principal := range principals {
				principalIds = append(principalIds, principal.(string))
			}
		}
		_, err := rc.SetPrincipals(ctx, d.Id(), 0, principalIds, roles.WithAutomaticVersioning(true))
		if err != nil {
			diags = append(diags, diag.Diagnostic{Severity: diag.Error, Summary: "error setting principals", Detail: err.Error()})
		} else {
			if err := d.Set(rolePrincipalIdsKey, principalIds); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange(roleGrantScopeIdsKey) {
		var grantScopeIds []string
		if grantScopeIdsVal, ok := d.GetOk(roleGrantScopeIdsKey); ok {
			grantScopes := grantScopeIdsVal.(*schema.Set).List()
			for _, grantScope := range grantScopes {
				grantScopeIds = append(grantScopeIds, grantScope.(string))
			}
		}
		_, err := rc.SetGrantScopes(ctx, d.Id(), 0, grantScopeIds, roles.WithAutomaticVersioning(true))
		if err != nil {
			diags = append(diags, diag.Diagnostic{Severity: diag.Error, Summary: "error setting grant scopes", Detail: err.Error()})
		} else {
			if err := d.Set(roleGrantScopeIdsKey, grantScopeIds); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return diags
}

func resourceRoleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	rc := roles.NewClient(md.client)

	_, err := rc.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting role: %s", err.Error())
	}

	return nil
}
