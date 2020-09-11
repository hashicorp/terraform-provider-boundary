package provider

import (
	"context"

	"github.com/hashicorp/boundary/api/groups"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	groupNameKey        = "name"
	groupDescriptionKey = "description"
	groupScopeIdKey     = "scope_id"
	groupMemberIdsKey   = "member_ids"
)

func resourceGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGroupCreate,
		ReadContext:   resourceGroupRead,
		UpdateContext: resourceGroupUpdate,
		DeleteContext: resourceGroupDelete,
		Schema: map[string]*schema.Schema{
			groupNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			groupDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			groupScopeIdKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			groupMemberIdsKey: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

// convertGroupToResourceData creates a ResourceData type from a Group
func convertGroupToResourceData(g *groups.Group, d *schema.ResourceData) diag.Diagnostics {
	if g.Name != "" {
		if err := d.Set(groupNameKey, g.Name); err != nil {
			return diag.FromErr(err)
		}
	}

	if g.Description != "" {
		if err := d.Set(groupDescriptionKey, g.Description); err != nil {
			return diag.FromErr(err)
		}
	}

	if g.ScopeId != "" {
		if err := d.Set(groupScopeIdKey, g.ScopeId); err != nil {
			return diag.FromErr(err)
		}
	}

	if g.MemberIds != nil {
		if err := d.Set(groupMemberIdsKey, g.MemberIds); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(g.Id)

	return nil
}

// convertResourceDataToOpts returns a localy built set of Opts using the values provided in the ResourceData.
func convertResourceDataToOpts(d *schema.ResourceData, meta *metaData) []groups.Option {
	opts := []groups.Option{}

	if nameVal, ok := d.GetOk(groupNameKey); ok {
		opts = append(opts, groups.WithName(nameVal.(string)))
	}

	if descVal, ok := d.GetOk(groupDescriptionKey); ok {
		opts = append(opts, groups.WithDescription(descVal.(string)))
	}

	return opts
}

func resourceGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	var scopeId string
	if scopeIdVal, ok := d.GetOk(groupScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []groups.Option{}

	var name *string
	nameVal, ok := d.GetOk(groupNameKey)
	if ok {
		nameStr := nameVal.(string)
		name = &nameStr
		opts = append(opts, groups.WithName(nameStr))
	}

	var desc *string
	descVal, ok := d.GetOk(groupDescriptionKey)
	if ok {
		descStr := descVal.(string)
		desc = &descStr
		opts = append(opts, groups.WithDescription(descStr))
	}

	grps := groups.NewClient(client)

	g, apiErr, err := grps.Create(
		ctx,
		scopeId,
		opts...)
	if err != nil {
		return diag.Errorf("error creating group: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating group: %s", apiErr.Message)
	}

	if val, ok := d.GetOk(groupMemberIdsKey); ok {
		list := val.(*schema.Set).List()
		memberIds := make([]string, 0, len(list))
		for _, i := range list {
			memberIds = append(memberIds, i.(string))
		}
		g, apiErr, err = grps.SetMembers(
			ctx,
			g.Id,
			g.Version,
			memberIds)
		if apiErr != nil {
			return diag.Errorf("error setting principals on role: %s", apiErr.Message)
		}
		if err != nil {
			return diag.Errorf("error setting principals on role: %v", err)
		}

		if err := d.Set(groupMemberIdsKey, memberIds); err != nil {
			return diag.FromErr(err)
		}
	}

	if name != nil {
		if err := d.Set(groupNameKey, name); err != nil {
			return diag.FromErr(err)
		}
	}

	if desc != nil {
		if err := d.Set(groupDescriptionKey, *desc); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(g.Id)

	return nil
}

func resourceGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	grps := groups.NewClient(client)

	g, apiErr, err := grps.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error reading group: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading group: %s", apiErr.Message)
	}
	if g == nil {
		return diag.Errorf("group nil after read")
	}

	raw := g.LastResponseMap()
	if raw == nil {
		return []diag.Diagnostic{
			{
				Severity: diag.Warning,
				Summary:  "response map empty after read",
			},
		}
	}

	d.Set(groupNameKey, raw["name"])
	d.Set(groupDescriptionKey, raw["description"])
	d.Set(groupScopeIdKey, raw["scope_id"])

	return nil
}

func resourceGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	grps := groups.NewClient(client)

	opts := []groups.Option{}

	var name *string
	if d.HasChange(groupNameKey) {
		opts = append(opts, groups.DefaultName())
		nameVal, ok := d.GetOk(groupNameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, groups.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(groupDescriptionKey) {
		opts = append(opts, groups.DefaultDescription())
		descVal, ok := d.GetOk(groupDescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, groups.WithDescription(descStr))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, groups.WithAutomaticVersioning(true))
		_, apiErr, err := grps.Update(
			ctx,
			d.Id(),
			0,
			opts...)
		if err != nil {
			return diag.Errorf("error updating group: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating group: %s", apiErr.Message)
		}
	}

	if d.HasChange(groupNameKey) {
		d.Set(groupNameKey, name)
	}
	if d.HasChange(groupDescriptionKey) {
		d.Set(groupDescriptionKey, desc)
	}

	if d.HasChange(groupMemberIdsKey) {
		var memberIds []string
		if membersVal, ok := d.GetOk(groupMemberIdsKey); ok {
			members := membersVal.(*schema.Set).List()
			for _, member := range members {
				memberIds = append(memberIds, member.(string))
			}
		}
		_, apiErr, err := grps.SetMembers(
			ctx,
			d.Id(),
			0,
			memberIds,
			groups.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating members on group: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating members on group: %s", apiErr.Message)
		}
		d.Set(groupMemberIdsKey, memberIds)
	}

	return nil
}

func resourceGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	grps := groups.NewClient(client)

	_, apiErr, err := grps.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting group: %s", err.Error())
	}
	if apiErr != nil {
		return diag.Errorf("error deleting group: %s", apiErr.Message)
	}

	return nil
}
