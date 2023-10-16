// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentialstores"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCredentialStoreStatic() *schema.Resource {
	return &schema.Resource{
		Description: "The static credential store data source allows you to discover an existing Boundary static credential store by name",
		ReadContext: dataSourceCredentialStoreStaticRead,

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the retrieved static credential store",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The name of the static credential store to retrieve",
				Type:        schema.TypeString,
				Required:    true,
			},
			DescriptionKey: {
				Description: "The description of the retrieved scope",
				Type:        schema.TypeString,
				Computed:    true,
			},
			ScopeIdKey: {
				Description: "The scope for this credential store",
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}
func dataSourceCredentialStoreStaticRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	opts := []credentialstores.Option{}

	var name string
	if v, ok := d.GetOk(NameKey); ok {
		name = v.(string)
	} else {
		return diag.Errorf("no name provided")
	}

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope is set")
	}

	client := credentialstores.NewClient(md.client)

	csl, err := client.List(ctx, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error calling read static credential store: %v", err)
	}
	if csl == nil {
		return diag.Errorf("no static credential store found")
	}

	var credentialstorestaticIdRead string
	for _, scopeItem := range csl.GetItems() {
		if scopeItem.Name == name {
			credentialstorestaticIdRead = scopeItem.Id
			break
		}
	}
	if credentialstorestaticIdRead == "" {
		return diag.Errorf("static credential store %v not found", err)
	}

	srr, err := client.Read(ctx, credentialstorestaticIdRead)
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read static credential store: %v", err)
	}
	if srr == nil {
		return diag.Errorf("static credential store nil after read")
	}

	if err := setFromStaticCredentialStoreResponseMap(d, srr.GetResponse().Map, false); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
func setFromStaticCredentialStoreReadResponseMap(d *schema.ResourceData, raw map[string]interface{}, fromRead bool) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}

	d.SetId(raw["id"].(string))
	return nil
}
