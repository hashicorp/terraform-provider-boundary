// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/managedgroups"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const managedGroupLdapGroupNamesKey = "group_names"

func resourceManagedGroupLdap() *schema.Resource {
	return &schema.Resource{
		Description: "The managed group resource allows you to configure a Boundary group.",

		CreateContext: resourceManagedGroupLdapCreate,
		ReadContext:   resourceManagedGroupLdapRead,
		UpdateContext: resourceManagedGroupLdapUpdate,
		DeleteContext: resourceManagedGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the group.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The managed group name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The managed group description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			AuthMethodIdKey: {
				Description: "The resource ID for the auth method.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			managedGroupLdapGroupNamesKey: {
				Description: "The list of groups that make up the managed group.",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required: true,
			},
		},
	}
}

func setFromManagedGroupLdapResponseMap(d *schema.ResourceData, raw map[string]any) error {
	if err := d.Set(NameKey, raw[NameKey]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw[DescriptionKey]); err != nil {
		return err
	}
	if err := d.Set(AuthMethodIdKey, raw[AuthMethodIdKey]); err != nil {
		return err
	}

	if attrsVal, ok := raw["attributes"]; ok {
		attrs := attrsVal.(map[string]interface{})
		if err := d.Set(managedGroupLdapGroupNamesKey, attrs[managedGroupLdapGroupNamesKey].([]interface{})); err != nil {
			return err
		}
	}

	d.SetId(raw[IDKey].(string))

	return nil
}

func resourceManagedGroupLdapCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	grpClient := managedgroups.NewClient(md.client)

	var authMethodId string
	authMethodVal, ok := d.GetOk(AuthMethodIdKey)
	if !ok {
		return diag.Errorf("no auth method ID provided")
	}
	authMethodId = authMethodVal.(string)

	var opts []managedgroups.Option
	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, managedgroups.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, managedgroups.WithDescription(descStr))
	}

	if v, ok := d.GetOk(managedGroupLdapGroupNamesKey); ok {
		names := make([]string, 0, len(v.([]any)))
		for _, n := range v.([]any) {
			names = append(names, n.(string))
		}
		opts = append(opts, managedgroups.WithLdapManagedGroupGroupNames(names))
	}

	grp, err := grpClient.Create(ctx, authMethodId, opts...)
	if err != nil {
		return diag.Errorf("error creating ldap managed group: %v", err)
	}
	if grp == nil {
		return diag.Errorf("managed group nil after create")
	}
	raw := grp.GetResponse().Map

	if err := setFromManagedGroupLdapResponseMap(d, raw); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceManagedGroupLdapRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	grpClient := managedgroups.NewClient(md.client)

	grp, err := grpClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading managed group: %v", err)
	}
	if grp == nil {
		return diag.Errorf("managed group nil after read")
	}

	if err := setFromManagedGroupLdapResponseMap(d, grp.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceManagedGroupLdapUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	grpClient := managedgroups.NewClient(md.client)

	var opts []managedgroups.Option
	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, managedgroups.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, managedgroups.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, managedgroups.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, managedgroups.WithDescription(descStr))
		}
	}

	var names []string
	if d.HasChange(managedGroupLdapGroupNamesKey) {
		if v, ok := d.GetOk(managedGroupLdapGroupNamesKey); ok {
			names = make([]string, 0, len(v.([]any)))
			for _, n := range v.([]any) {
				names = append(names, n.(string))
			}
			opts = append(opts, managedgroups.WithLdapManagedGroupGroupNames(names))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, managedgroups.WithAutomaticVersioning(true))
		_, err := grpClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating managed group: %v", err)
		}
	}

	if d.HasChange(NameKey) {
		if err := d.Set(NameKey, name); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(DescriptionKey) {
		if err := d.Set(DescriptionKey, desc); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(managedGroupLdapGroupNamesKey) {
		if err := d.Set(managedGroupLdapGroupNamesKey, names); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
