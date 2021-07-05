// Code generated by scripts/generate_datasource.go. DO NOT EDIT.
//go:generate go run ../../scripts/generate_datasource.go -name Roles -path roles

// This file was generated based on Boundary v0.4.0

package provider

import (
	"context"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var dataSourceRolesSchema = map[string]*schema.Schema{
	"filter": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"items": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"authorized_actions": {
					Type:        schema.TypeList,
					Description: "Output only. The available actions on this resource for this user.",
					Computed:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"created_time": {
					Type:        schema.TypeString,
					Description: "Output only. The time this resource was created.",
					Computed:    true,
				},
				"description": {
					Type:        schema.TypeString,
					Description: "Optional user-set description for identification purposes.",
					Computed:    true,
				},
				"grant_scope_id": {
					Type:        schema.TypeString,
					Description: "The Scope the grants will apply to. If the Role is at the global scope, this can be an org or project. If the Role is at an org scope, this can be a project within the org. It is invalid for this to be anything other than the Role's scope when the Role's scope is a project.",
					Computed:    true,
				},
				"grant_strings": {
					Type:        schema.TypeList,
					Description: "Output only. The grants that this role provides for its principals.",
					Computed:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"grants": {
					Type:        schema.TypeList,
					Description: "Output only. The parsed grant information.",
					Computed:    true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"canonical": {
								Type:        schema.TypeString,
								Description: "Output only. The canonically-formatted string.",
								Computed:    true,
							},
							"json": {
								Type:     schema.TypeList,
								Computed: true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"actions": {
											Type:        schema.TypeList,
											Description: "Output only. The actions.",
											Computed:    true,
											Elem: &schema.Schema{
												Type: schema.TypeString,
											},
										},
										"id": {
											Type:        schema.TypeString,
											Description: "Output only. The ID, if set.",
											Computed:    true,
										},
										"type": {
											Type:        schema.TypeString,
											Description: "Output only. The type, if set.",
											Computed:    true,
										},
									},
								},
							},
							"raw": {
								Type:        schema.TypeString,
								Description: "Output only. The original user-supplied string.",
								Computed:    true,
							},
						},
					},
				},
				"id": {
					Type:        schema.TypeString,
					Description: "Output only. The ID of the Role.",
					Computed:    true,
				},
				"name": {
					Type:        schema.TypeString,
					Description: "Optional name for identification purposes.",
					Computed:    true,
				},
				"principal_ids": {
					Type:        schema.TypeList,
					Description: "Output only. The IDs (only) of principals that are assigned to this role.",
					Computed:    true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"principals": {
					Type:        schema.TypeList,
					Description: "Output only. The principals that are assigned to this role.",
					Computed:    true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"id": {
								Type:        schema.TypeString,
								Description: "Output only. The ID of the principal.",
								Computed:    true,
							},
							"scope_id": {
								Type:        schema.TypeString,
								Description: "Output only. The Scope of the principal.",
								Computed:    true,
							},
							"type": {
								Type:        schema.TypeString,
								Description: "Output only. The type of the principal.",
								Computed:    true,
							},
						},
					},
				},
				"scope": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"description": {
								Type:        schema.TypeString,
								Description: "Output only. The description of the Scope, if any.",
								Computed:    true,
							},
							"id": {
								Type:        schema.TypeString,
								Description: "Output only. The ID of the Scope.",
								Computed:    true,
							},
							"name": {
								Type:        schema.TypeString,
								Description: "Output only. The name of the Scope, if any.",
								Computed:    true,
							},
							"parent_scope_id": {
								Type:        schema.TypeString,
								Description: "Output only. The ID of the parent Scope, if any. This field will be empty if this is the \"global\" scope.",
								Computed:    true,
							},
							"type": {
								Type:        schema.TypeString,
								Description: "Output only. The type of the Scope.",
								Computed:    true,
							},
						},
					},
				},
				"scope_id": {
					Type:        schema.TypeString,
					Description: "The ID of the Scope containing this Role.",
					Computed:    true,
				},
				"updated_time": {
					Type:        schema.TypeString,
					Description: "Output only. The time this resource was last updated.",
					Computed:    true,
				},
				"version": {
					Type:        schema.TypeInt,
					Description: "Version is used in mutation requests, after the initial creation, to ensure this resource has not changed.\nThe mutation will fail if the version does not match the latest known good version.",
					Computed:    true,
				},
			},
		},
	},
	"recursive": {
		Type:     schema.TypeBool,
		Optional: true,
	},
	"scope_id": {
		Type:     schema.TypeString,
		Optional: true,
	},
}

func dataSourceRoles() *schema.Resource {
	return &schema.Resource{
		Description: "Lists all Roles.",
		Schema:      dataSourceRolesSchema,
		ReadContext: dataSourceRolesRead,
	}
}

func dataSourceRolesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*metaData).client

	req, err := client.NewRequest(ctx, "GET", "roles", nil)
	if err != nil {
		return diag.FromErr(err)
	}

	q := url.Values{}
	q.Add("filter", d.Get("filter").(string))
	recursive := d.Get("recursive").(bool)
	if recursive {
		q.Add("recursive", strconv.FormatBool(recursive))
	}
	q.Add("scope_id", d.Get("scope_id").(string))
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
	err = set(dataSourceRolesSchema, d, resp.Map)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("boundary-roles")

	return nil
}