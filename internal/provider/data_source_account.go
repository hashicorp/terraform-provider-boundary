// Copyright (c) HashiCorp, Inc.
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
			IDKey: {
				Description: "The ID of the retrieved account.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description:  "The name of the account to retrieve.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			DescriptionKey: {
				Description: "The description of the retrieved account.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			AuthMethodIdKey: {
				Description:  "The auth method ID that will be queried for the account.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}

func dataSourceAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	opts := []accounts.Option{}

	var name string
	if v, ok := d.GetOk(NameKey); ok {
		name = v.(string)
	} else {
		return diag.Errorf("no name provided")
	}

	var authMethodId string
	if authMethodIdVal, ok := d.GetOk(AuthMethodIdKey); ok {
		authMethodId = authMethodIdVal.(string)
	} else {
		return diag.Errorf("no auth method ID provided")
	}

	acl := accounts.NewClient(md.client)

	als, err := acl.List(ctx, authMethodId, opts...)
	if err != nil {
		return diag.Errorf("error calling list account: %v", err)
	}
	if als == nil {
		return diag.Errorf("no accounts found")
	}

	var accountIdRead string
	for _, accountItem := range als.GetItems() {
		if accountItem.Name == name {
			accountIdRead = accountItem.Id
			break
		}
	}

	if accountIdRead == "" {
		return diag.Errorf("account name %v not found in account list", err)
	}

	arr, err := acl.Read(ctx, accountIdRead)
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

	if err := setFromAccountReadResponseMap(d, arr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func setFromAccountReadResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}

	d.SetId(raw["id"].(string))
	return nil
}
