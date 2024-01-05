// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/groups"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceGroup() *schema.Resource {
	return &schema.Resource{
		Description: "The boundary_group data source allows you to find a Boundary group.",
		ReadContext: dataSourceGroupRead,

		Schema: map[string]*schema.Schema{
			NameKey: {
				Description:  "The name of the group to retrieve.",
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
				Description: "The ID of the retrieved group.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			DescriptionKey: {
				Description: "The description of the retrieved group.",
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
			GroupMemberIdsKey: {
				Description: "Resource IDs for group members, these are most likely boundary users.",
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
			},
		},
	}
}

func dataSourceGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	name := d.Get(NameKey).(string)
	scopeId := d.Get(ScopeIdKey).(string)

	gcl := groups.NewClient(md.client)
	groupsList, err := gcl.List(ctx, scopeId,
		groups.WithFilter(FilterWithItemNameMatches(name)),
	)
	if err != nil {
		return diag.Errorf("error calling list group: %v", err)
	}
	groups := groupsList.GetItems()
	if groups == nil {
		return diag.Errorf("no groups found")
	}
	if len(groups) == 0 {
		return diag.Errorf("no matching group found")
	}
	if len(groups) > 1 {
		return diag.Errorf("error found more than 1 group")
	}

	grr, err := gcl.Read(ctx, groups[0].Id)
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read group: %v", err)
	}
	if grr == nil {
		return diag.Errorf("group nil after read")
	}

	if err := setFromGroupRead(d, *grr.Item); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func setFromGroupRead(d *schema.ResourceData, group groups.Group) error {
	if err := d.Set(NameKey, group.Name); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, group.Description); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, group.ScopeId); err != nil {
		return err
	}
	if err := d.Set(GroupMemberIdsKey, group.MemberIds); err != nil {
		return err
	}

	d.Set(ScopeKey, flattenScopeInfo(group.Scope))
	d.SetId(group.Id)
	return nil
}
