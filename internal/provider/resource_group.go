package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/groups"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	groupMemberIdsKey = "member_ids"
)

func resourceGroup() *schema.Resource {
	return &schema.Resource{
		Description: "The group resource allows you to configure a Boundary group.",

		CreateContext: resourceGroupCreate,
		ReadContext:   resourceGroupRead,
		UpdateContext: resourceGroupUpdate,
		DeleteContext: resourceGroupDelete,
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
				Description: "The group name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The group description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			groupMemberIdsKey: {
				Description: "Resource IDs for group members, these are most likely boundary users.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func setFromGroupResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(ScopeIdKey, raw["scope_id"])
	d.Set(groupMemberIdsKey, raw["member_ids"])
	d.SetId(raw["id"].(string))
}

func resourceGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []groups.Option{}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, groups.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, groups.WithDescription(descStr))
	}

	grps := groups.NewClient(md.client)

	gcr, err := grps.Create(ctx, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating group: %v", err)
	}
	if gcr == nil {
		return diag.Errorf("group nil after create")
	}
	raw := gcr.GetResponseMap()

	if val, ok := d.GetOk(groupMemberIdsKey); ok {
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
			return diag.Errorf("group nil after setting members")
		}
		raw = gcsmr.GetResponseMap()
	}

	setFromGroupResponseMap(d, raw)

	return nil
}

func resourceGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	grps := groups.NewClient(md.client)

	g, err := grps.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr.Status == int32(http.StatusNotFound) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading group: %v", err)
	}
	if g == nil {
		return diag.Errorf("group nil after read")
	}

	setFromGroupResponseMap(d, g.GetResponseMap())

	return nil
}

func resourceGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	grps := groups.NewClient(md.client)

	opts := []groups.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, groups.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, groups.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, groups.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, groups.WithDescription(descStr))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, groups.WithAutomaticVersioning(true))
		_, err := grps.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating group: %v", err)
		}
	}

	if d.HasChange(NameKey) {
		d.Set(NameKey, name)
	}
	if d.HasChange(DescriptionKey) {
		d.Set(DescriptionKey, desc)
	}

	// The above call may not actually happen, so we use d.Id() and automatic
	// versioning here
	if d.HasChange(groupMemberIdsKey) {
		var memberIds []string
		if membersVal, ok := d.GetOk(groupMemberIdsKey); ok {
			members := membersVal.(*schema.Set).List()
			for _, member := range members {
				memberIds = append(memberIds, member.(string))
			}

		}
		_, err := grps.SetMembers(ctx, d.Id(), 0, memberIds, groups.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating members in group: %v", err)
		}
		d.Set(groupMemberIdsKey, memberIds)
	}

	return nil
}

func resourceGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	grps := groups.NewClient(md.client)

	_, err := grps.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete group: %s", err.Error())
	}

	return nil
}
