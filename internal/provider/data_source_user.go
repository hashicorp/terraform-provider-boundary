// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/boundary/api/users"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// const (
// 	loginNameKey     = "login_name"
// 	primaryAccountID = "primary_account_id"
// )

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
				Description: "The username.",
				Type:        schema.TypeString,
				Required:    true,
			},
			DescriptionKey: {
				Description: "The user description.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "global",
			},
			userAccountIDsKey: {
				Description: "Account ID's to associate with this user resource.",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			// TODO: add after basic functionality works

			// authorizedActions: {
			// 	Description: "A list of actions that the worker is entitled to perform.",
			// 	Type:        schema.TypeList,
			// 	Elem: &schema.Schema{
			// 		Type: schema.TypeString,
			// 	},
			// 	Computed: true,
			// },
			// loginNameKey: {
			// 	Description: "Login name for user.",
			// 	Type:        schema.TypeString,
			// 	Computed:    true,
			// },
			// primaryAccountID: {
			// 	Description: "Primary account ID.",
			// 	Type:        schema.TypeString,
			// 	Computed:    true,
			// },
		},
	}
}

func dataSourceUserRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	usrs := users.NewClient(md.client)

	opts := []users.Option{}

	// Get user ID using name
	name := d.Get("name").(string)
	scopeID := d.Get("scope_id").(string)

	opts = append(opts, users.WithName(name))

	usersList, err := usrs.List(ctx, scopeID, opts...)

	if err != nil {
		return diag.Errorf("error calling list user: %v", err)
	}
	users := usersList.GetItems()

	// check length, 0 means no user, > 1 means too many
	if len(users) == 0 || users[0] == nil {
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
	d.SetId(user.Id)
	return nil
}
