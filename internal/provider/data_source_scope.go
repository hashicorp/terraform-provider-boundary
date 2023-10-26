// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceScope() *schema.Resource {
	return &schema.Resource{
		Description: "The scope data source allows you to discover an existing Boundary scope by name.",
		ReadContext: dataSourceScopeRead,

		Schema: map[string]*schema.Schema{
			NameKey: {
				Description:  "The name of the scope to retrieve.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			ScopeIdKey: {
				Description:  "The parent scope ID that will be queried for the scope.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			IDKey: {
				Description: "The ID of the retrieved scope.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			DescriptionKey: {
				Description: "The description of the retrieved scope.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceScopeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	name := d.Get(NameKey).(string)
	scopeId := d.Get(ScopeIdKey).(string)

	scl := scopes.NewClient(md.client)
	scopesList, err := scl.List(ctx, scopeId,
		scopes.WithFilter(FilterWithItemNameMatches(name)),
	)
	if err != nil {
		return diag.Errorf("error calling list scope: %v", err)
	}
	scopes := scopesList.GetItems()
	if scopes == nil {
		return diag.Errorf("no scopes found")
	}
	if len(scopes) == 0 {
		return diag.Errorf("no matching scope found")
	}
	if len(scopes) > 1 {
		return diag.Errorf("error found more than 1 scope")
	}

	srr, err := scl.Read(ctx, scopes[0].Id)
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read scope: %v", err)
	}
	if srr == nil {
		return diag.Errorf("scope nil after read")
	}

	if err := setFromScopeRead(d, *srr.Item); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func setFromScopeRead(d *schema.ResourceData, scope scopes.Scope) error {
	if err := d.Set(NameKey, scope.Name); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, scope.Description); err != nil {
		return err
	}

	d.SetId(scope.Id)
	return nil
}
