// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/accounts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceAccount() *schema.Resource {
	return &schema.Resource{
		Description: "The boundary_account data source allows you to find a Boundary account.",
		ReadContext: dataSourceAccountRead,

		Schema: map[string]*schema.Schema{
			NameKey: {
				Description:  "The name of the account to retrieve.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			AuthMethodIdKey: {
				Description:  "The auth method ID that will be queried for the account.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			IDKey: {
				Description: "The ID of the retrieved account.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			DescriptionKey: {
				Description: "The description of the retrieved account.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			TypeKey: {
				Description: "The type of the account",
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
		},
	}
}

func dataSourceAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	name := d.Get(NameKey).(string)
	authMethodId := d.Get(AuthMethodIdKey).(string)

	acl := accounts.NewClient(md.client)
	accountsList, err := acl.List(ctx, authMethodId,
		accounts.WithFilter(FilterWithItemNameMatches(name)),
	)
	if err != nil {
		return diag.Errorf("error calling list account: %v", err)
	}
	accounts := accountsList.GetItems()
	if accounts == nil {
		return diag.Errorf("no accounts found")
	}
	if len(accounts) == 0 {
		return diag.Errorf("no matching account found")
	}
	if len(accounts) > 1 {
		return diag.Errorf("error found more than 1 account")
	}

	arr, err := acl.Read(ctx, accounts[0].Id)
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read account: %v", err)
	}
	if arr == nil {
		return diag.Errorf("account nil after read")
	}

	if err := setFromAccountRead(d, *arr.Item); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func setFromAccountRead(d *schema.ResourceData, account accounts.Account) error {
	if err := d.Set(NameKey, account.Name); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, account.Description); err != nil {
		return err
	}
	if err := d.Set(TypeKey, account.Type); err != nil {
		return err
	}

	d.Set(ScopeKey, flattenScopeInfo(account.Scope))
	d.SetId(account.Id)
	return nil
}
