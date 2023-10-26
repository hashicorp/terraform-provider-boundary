// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/users"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		Description: "The user data source allows you to find a Boundary user.",
		ReadContext: dataSourceUserRead,

		Schema: map[string]*schema.Schema{
			NameKey: {
				Description:  "The username to search for.",
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
				Description: "The ID of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			DescriptionKey: {
				Description: "The user description.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			UserAccountIdsKey: {
				Description: "Account ID's to associate with this user resource.",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
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
			authorizedActions: {
				Description: "A list of actions that the worker is entitled to perform.",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			LoginNameKey: {
				Description: "Login name for user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			PrimaryAccountIdKey: {
				Description: "Primary account ID.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceUserRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	name := d.Get(NameKey).(string)
	scopeID := d.Get(ScopeIdKey).(string)

	ucl := users.NewClient(md.client)
	usersList, err := ucl.List(ctx, scopeID,
		users.WithFilter(FilterWithItemNameMatches(name)),
	)
	if err != nil {
		return diag.Errorf("error calling list user: %v", err)
	}
	users := usersList.GetItems()
	if users == nil {
		return diag.Errorf("no users found")
	}
	if len(users) == 0 {
		return diag.Errorf("no matching user found")
	}
	if len(users) > 1 {
		return diag.Errorf("error found more than 1 user")
	}

	urr, err := ucl.Read(ctx, users[0].Id)
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read scope: %v", err)
	}
	if urr == nil {
		return diag.Errorf("scope nil after read")
	}

	if err := setFromUserRead(d, *urr.Item); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func setFromUserRead(d *schema.ResourceData, user users.User) error {
	if err := d.Set(NameKey, user.Name); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, user.Description); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, user.ScopeId); err != nil {
		return err
	}
	if err := d.Set(UserAccountIdsKey, user.AccountIds); err != nil {
		return err
	}
	if err := d.Set(authorizedActions, user.AuthorizedActions); err != nil {
		return err
	}
	if err := d.Set(LoginNameKey, user.LoginName); err != nil {
		return err
	}
	if err := d.Set(PrimaryAccountIdKey, user.PrimaryAccountId); err != nil {
		return err
	}

	d.Set(ScopeKey, flattenScopeInfo(user.Scope))

	d.SetId(user.Id)
	return nil
}
