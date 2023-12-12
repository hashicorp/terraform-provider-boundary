// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceAuthMethod() *schema.Resource {
	return &schema.Resource{
		Description: "The boundary_auth_method data source allows you to find a Boundary auth method.",
		ReadContext: dataSourceAuthMethodRead,

		Schema: map[string]*schema.Schema{
			NameKey: {
				Description:  "The name of the auth method to retrieve.",
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
				Description: "The ID of the retrieved auth method.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			DescriptionKey: {
				Description: "The description of the retrieved auth method.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			TypeKey: {
				Description: "The type of the auth method",
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

func dataSourceAuthMethodRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	name := d.Get(NameKey).(string)
	scopeId := d.Get(ScopeIdKey).(string)

	amcl := authmethods.NewClient(md.client)
	authMethodsList, err := amcl.List(ctx, scopeId,
		authmethods.WithFilter(FilterWithItemNameMatches(name)),
	)
	if err != nil {
		return diag.Errorf("error calling list auth method: %v", err)
	}
	authMethods := authMethodsList.GetItems()
	if authMethods == nil {
		return diag.Errorf("no auth methods found")
	}
	if len(authMethods) == 0 {
		return diag.Errorf("no matching auth method found")
	}
	if len(authMethods) > 1 {
		return diag.Errorf("error found more than 1 auth method")
	}

	amrr, err := amcl.Read(ctx, authMethods[0].Id)
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read auth method: %v", err)
	}
	if amrr == nil {
		return diag.Errorf("auth method nil after read")
	}

	if err := setFromAuthMethodRead(d, *amrr.Item); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func setFromAuthMethodRead(d *schema.ResourceData, authMethod authmethods.AuthMethod) error {
	if err := d.Set(NameKey, authMethod.Name); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, authMethod.Description); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, authMethod.ScopeId); err != nil {
		return err
	}
	if err := d.Set(TypeKey, authMethod.Type); err != nil {
		return err
	}
	d.Set(ScopeKey, flattenScopeInfo(authMethod.Scope))
	d.SetId(authMethod.Id)
	return nil
}
