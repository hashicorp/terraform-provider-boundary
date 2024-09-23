// Code generated by "make datasources"; DO NOT EDIT.
// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var dataSourceScopesSchema = map[string]*schema.Schema{
	"est_item_count": {
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "An estimate at the total items available. This may change during pagination.",
	},
	"filter": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "You can specify that the filter should only return items that match.\nRefer to [filter expressions](https://developer.hashicorp.com/boundary/docs/concepts/filtering) for more information.",
	},
	"items": {
		Type:        schema.TypeList,
		Computed:    true,
		Description: "The items returned in this page.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"authorized_actions": {
					Type:        schema.TypeList,
					Computed:    true,
					Description: "The available actions on this resource for this user.",
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"created_time": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The time this resource was created.",
				},
				"description": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Optional user-set descripton for identification purposes.",
				},
				"id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the scope.",
				},
				"name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Optional name for identification purposes.",
				},
				"primary_auth_method_id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"scope": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"description": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "The description of the scope, if any.",
							},
							"id": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "The ID of the scope.",
							},
							"name": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "The name of the scope, if any.",
							},
							"parent_scope_id": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "The ID of the parent scope, if any. This field is empty if it is the \"global\" scope.",
							},
							"type": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "The type of the scope.",
							},
						},
					},
				},
				"scope_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the scope this resource is in. If this is the \"global\" scope this field will be empty.",
				},
				"storage_policy_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The attached storage policy id.",
				},
				"type": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The type of the resource.",
				},
				"updated_time": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The time this resource was last updated.",
				},
				"version": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "Version is used in mutation requests, after the initial creation, to ensure this resource has not changed.\nThe mutation will fail if the version does not match the latest known good version.",
				},
			},
		},
	},
	"list_token": {
		Type:        schema.TypeString,
		Optional:    true,
		Computed:    true,
		Description: "An opaque token used to continue an existing iteration or\nrequest updated items. If paginating, use this token in the\nnext list request.",
	},
	"page_size": {
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The maximum size of a page in this iteration.\nIf you do not set a page size, Boundary uses the configured default page size.\nIf the page_size is greater than the default page size configured,\nBoundary truncates the page size to this number.",
	},
	"recursive": {
		Type:     schema.TypeBool,
		Optional: true,
	},
	"removed_ids": {
		Type:        schema.TypeList,
		Computed:    true,
		Description: "A list of item IDs that have been removed since they were returned\nas part of an pagination. They should be dropped from any client cache.\nThis may contain items that are not known to the cache, if they were\ncreated and deleted between listings.",
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	},
	"response_type": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The type of response, either \"delta\" or \"complete\".\nDelta signifies that this is part of a paginated result\nor an update to a previously completed pagination.\nComplete signifies that it is the last page.",
	},
	"scope_id": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"sort_by": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The name of the field which the items are sorted by.",
	},
	"sort_dir": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The direction of the sort, either \"asc\" or \"desc\".",
	},
}

func dataSourceScopes() *schema.Resource {
	return &schema.Resource{
		Description: "Lists scopes",
		ReadContext: dataSourceScopesRead,
		Schema:      dataSourceScopesSchema,
	}
}

func dataSourceScopesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*metaData).client

	req, err := client.NewRequest(ctx, "GET", "scopes", nil)
	if err != nil {
		return diag.FromErr(err)
	}

	q := url.Values{}
	q.Add("filter", d.Get("filter").(string))
	q.Add("list_token", d.Get("list_token").(string))
	if d.Get("scope_id") != 0 {
		q.Add("page_size", strconv.Itoa(d.Get("page_size").(int)))
	}
	recursive := d.Get("recursive").(bool)
	if recursive {
		q.Add("recursive", strconv.FormatBool(recursive))
	}
	if d.Get("scope_id") != "" {
		q.Add("scope_id", d.Get("scope_id").(string))
	}
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		diag.FromErr(err)
	}
	apiError, err := resp.Decode(nil)
	if err != nil {
		return diag.FromErr(err)
	}
	if apiError != nil {
		return apiErr(apiError)
	}
	err = set(dataSourceScopesSchema, d, resp.Map)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("boundary-scopes")

	return nil
}
