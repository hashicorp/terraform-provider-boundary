// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/boundary/api/scopes"
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
			IDKey: {
				Description: "The ID of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description:  "The username to search for.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			DescriptionKey: {
				Description: "The user description.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			ScopeIdKey: {
				Description:  "The scope ID in which the resource is created. Defaults `global` if unset.",
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "global",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			userAccountIDsKey: {
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
	usrs := users.NewClient(md.client)

	opts := []users.Option{}

	// Get user ID using name
	name := d.Get(NameKey).(string)
	scopeID := d.Get(ScopeIdKey).(string)

	opts = append(opts, users.WithFilter(FilterWithItemNameMatches(name)))

	usersList, err := usrs.List(ctx, scopeID, opts...)
	if err != nil {
		return diag.Errorf("error calling list user: %v", err)
	}
	users := usersList.GetItems()

	// check length, 0 means no user, > 1 means too many
	if len(users) == 0 {
		return diag.Errorf("no matching user found: %v", err)
	}

	if len(users) > 1 {
		return diag.Errorf("error found more than 1 user: %v", err)
	}

	if err := setFromUserItem(d, *users[0]); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func setFromUserItem(d *schema.ResourceData, user users.User) error {
	if err := d.Set(NameKey, user.Name); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, user.Description); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, user.ScopeId); err != nil {
		return err
	}
	if err := d.Set(userAccountIDsKey, user.AccountIds); err != nil {
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

func flattenScopeInfo(scope *scopes.ScopeInfo) []interface{} {
	if scope == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{})

	if v := scope.Id; v != "" {
		m[IDKey] = v
	}
	if v := scope.Type; v != "" {
		m[TypeKey] = v
	}
	if v := scope.Description; v != "" {
		m[DescriptionKey] = v
	}
	if v := scope.ParentScopeId; v != "" {
		m[ParentScopeIdKey] = v
	}
	if v := scope.Name; v != "" {
		m[NameKey] = v
	}

	return []interface{}{m}
}
