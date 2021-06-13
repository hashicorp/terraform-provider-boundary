package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/managedgroups"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	managedGroupMemberIdsKey = "member_ids"
	managedGroupFilterKey    = "filter"
)

func resourceManagedGroup() *schema.Resource {
	return &schema.Resource{
		Description: "The managed group resource allows you to configure a Boundary group.",

		CreateContext: resourceManagedGroupCreate,
		ReadContext:   resourceManagedGroupRead,
		UpdateContext: resourceManagedGroupUpdate,
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
			ScopeIdKey: {
				Description: "The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			managedGroupMemberIdsKey: {
				Description: "Resource IDs for managed group members, these are most likely boundary users.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			managedGroupFilterKey: {
				Description: "Filters...",
				Type:        schema.TypeString,
				Optional:    true,
			},
		},
	}
}

func setFromGroupResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw["scope_id"]); err != nil {
		return err
	}
	if err := d.Set(managedGroupMemberIdsKey, raw[managedGroupMemberIdsKey]); err != nil {
		return err
	}
	if err := d.Set(managedGroupFilterKey, raw[managedGroupFilterKey]); err != nil {
		return err
	}
	d.SetId(raw["id"].(string))
	return nil
}

func resourceManagedGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []managedgroups.Option{}

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

	v, ok := d.GetOk(managedGroupFilterKey)
	if ok {
		str := v.(string)
		opts = append(opts, managedgroups.WithOidcManagedGroupFilter(str))
	}

	grps := managedgroups.NewClient(md.client)

	gcr, err := grps.Create(ctx, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating managed group: %v", err)
	}
	if gcr == nil {
		return diag.Errorf("managed group nil after create")
	}
	raw := gcr.GetResponse().Map

	if val, ok := d.GetOk(managedGroupMemberIdsKey); ok {
		list := val.(*schema.Set).List()
		memberIds := make([]string, 0, len(list))
		for _, i := range list {
			memberIds = append(memberIds, i.(string))
		}
		gcsmr, err := grps.SetMembers(ctx, gcr.Item.Id, gcr.Item.Version, memberIds)
		if err != nil {
			return diag.Errorf("error setting principals on role: %v", err)
		}
		if gcsmr == nil {
			return diag.Errorf("managed group nil after setting members")
		}
		raw = gcsmr.GetResponse().Map
	}

	if err := setFromGroupResponseMap(d, raw); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceManagedGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	grps := managedgroups.NewClient(md.client)

	g, err := grps.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading managed group: %v", err)
	}
	if g == nil {
		return diag.Errorf("managed group nil after read")
	}

	if err := setFromGroupResponseMap(d, g.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceManagedGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	grps := managedgroups.NewClient(md.client)

	opts := []managedgroups.Option{}

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

	var filter *string
	if d.HasChange(managedGroupFilterKey) {
		if f, ok := d.GetOk(managedGroupFilterKey); ok {
			filterStr := f.(string)
			filter = &filterStr
			opts = append(opts, managedgroups.WithOidcManagedGroupFilter(filterStr))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, managedgroups.WithAutomaticVersioning(true))
		_, err := grps.Update(ctx, d.Id(), 0, opts...)
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
	if d.HasChange(managedGroupFilterKey) {
		if err := d.Set(managedGroupFilterKey, filter); err != nil {
			return diag.FromErr(err)
		}
	}

	// The above call may not actually happen, so we use d.Id() and automatic
	// versioning here
	if d.HasChange(managedGroupMemberIdsKey) {
		var memberIds []string
		if membersVal, ok := d.GetOk(managedGroupMemberIdsKey); ok {
			members := membersVal.(*schema.Set).List()
			for _, member := range members {
				memberIds = append(memberIds, member.(string))
			}

		}
		_, err := grps.SetMembers(ctx, d.Id(), 0, memberIds, managedgroups.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating members in managed group: %v", err)
		}
		if err := d.Set(managedGroupMemberIdsKey, memberIds); err != nil {
			return diag.FromErr(err)
		}

	}

	return nil
}

func resourceManagedGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	grps := managedgroups.NewClient(md.client)

	_, err := grps.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete managed group: %s", err.Error())
	}

	return nil
}
