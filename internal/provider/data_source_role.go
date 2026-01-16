// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/roles"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceRole() *schema.Resource {
	return &schema.Resource{
		Description: "The boundary_role data source allows you to find a Boundary role.",
		ReadContext: dataSourceRoleRead,

		Schema: map[string]*schema.Schema{
			NameKey: {
				Description:  "The name of the role to retrieve.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			ScopeIdKey: {
				Description:  "The scope ID in which the resource is created. Defaults `global` if unset.",
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "global",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			IDKey: {
				Description: "The ID of the retrieved role.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			DescriptionKey: {
				Description: "The description of the retrieved role.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			ScopeKey: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						IDKey: {
							Type:     schema.TypeString,
							Computed: true,
						},
						NameKey: {
							Type:     schema.TypeString,
							Computed: true,
						},
						TypeKey: {
							Type:     schema.TypeString,
							Computed: true,
						},
						DescriptionKey: {
							Type:     schema.TypeString,
							Computed: true,
						},
						ParentScopeIdKey: {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			rolePrincipalIdsKey: {
				Description: "The list of principal (user or group) IDs added as principals on the role.",
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
			},
			roleGrantStringsKey: {
				Description: "The list of stringified grants for the role.",
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
			},
			roleGrantScopeIdsKey: {
				Description: "The list of scope IDs for the role's grants.",
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
			},
		},
	}
}

func dataSourceRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	name := d.Get(NameKey).(string)
	scopeId := d.Get(ScopeIdKey).(string)

	rcl := roles.NewClient(md.client)
	rolesList, err := rcl.List(ctx, scopeId,
		roles.WithFilter(FilterWithItemNameMatches(name)),
	)
	if err != nil {
		return diag.Errorf("error calling list role: %v", err)
	}
	roles := rolesList.GetItems()
	if roles == nil {
		return diag.Errorf("no roles found")
	}
	if len(roles) == 0 {
		return diag.Errorf("no matching role found")
	}
	if len(roles) > 1 {
		return diag.Errorf("error found more than 1 role")
	}

	rrr, err := rcl.Read(ctx, roles[0].Id)
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read role: %v", err)
	}
	if rrr == nil {
		return diag.Errorf("role nil after read")
	}

	if err := setFromRoleRead(d, *rrr.Item); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func setFromRoleRead(d *schema.ResourceData, role roles.Role) error {
	if err := d.Set(NameKey, role.Name); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, role.Description); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, role.ScopeId); err != nil {
		return err
	}
	if err := d.Set(roleGrantStringsKey, role.GrantStrings); err != nil {
		return err
	}
	if err := d.Set(roleGrantScopeIdsKey, role.GrantScopeIds); err != nil {
		return err
	}
	if err := d.Set(rolePrincipalIdsKey, role.PrincipalIds); err != nil {
		return err
	}

	d.Set(ScopeKey, flattenScopeInfo(role.Scope))
	d.SetId(role.Id)
	return nil
}
