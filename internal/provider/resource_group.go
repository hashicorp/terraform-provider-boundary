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

// convertResourceDataToGroup returns a localy built Group using the values provided in the ResourceData.
func convertResourceDataToGroup(d *schema.ResourceData, meta *metaData) *groups.Group {
	g := new(groups.Group)

	if descVal, ok := d.GetOk(groupDescriptionKey); ok {
		g.Description = descVal.(string)
	}

	if nameVal, ok := d.GetOk(groupNameKey); ok {
		g.Name = nameVal.(string)
	}

	if scopeIdVal, ok := d.GetOk(groupScopeIdKey); ok {
		g.ScopeId = scopeIdVal.(string)
	}

	if val, ok := d.GetOk(groupMemberIdsKey); ok {
		memberIds := val.(*schema.Set).List()
		for _, i := range memberIds {
			g.MemberIds = append(g.MemberIds, i.(string))
		}
	}

	if d.Id() != "" {
		g.Id = d.Id()
	}

	return g
}

func resourceGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	g := convertResourceDataToGroup(d, md)
	grps := groups.NewClient(client)

	memIds := g.MemberIds

	g, apiErr, err := grps.Create(
		ctx,
		g.ScopeId,
		groups.WithName(g.Name),
		groups.WithDescription(g.Description))
	if err != nil {
		return diag.Errorf("error creating group: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating group: %s", apiErr.Message)
	}

	if len(memIds) > 0 {
		g, apiErr, err = grps.SetMembers(
			ctx,
			g.Id,
			g.Version,
			memIds)
		if apiErr != nil {
			return diag.Errorf("error setting principals on role: %s", apiErr.Message)
		}
		if err != nil {
			return diag.Errorf("error setting principals on role: %v", err)
		}
	}

	return convertGroupToResourceData(g, d)
}

func resourceGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	g := convertResourceDataToGroup(d, md)
	grps := groups.NewClient(client)

	g, apiErr, err := grps.Read(ctx, g.Id)
	if err != nil {
		return diag.Errorf("error reading group: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading group: %s", apiErr.Message)
	}

	return convertGroupToResourceData(g, d)
}

func resourceGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	g := convertResourceDataToGroup(d, md)
	grps := groups.NewClient(client)

	var updateGroup bool

	switch {
	case d.HasChange(groupNameKey),
		d.HasChange(groupDescriptionKey):
		updateGroup = true
	}

	if updateGroup {
		_, apiErr, err := grps.Update(
			ctx,
			g.Id,
			0,
			groups.WithAutomaticVersioning(true),
			groups.WithName(g.Name),
			groups.WithDescription(g.Description))
		if err != nil {
			return diag.Errorf("error updating group: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating group: %s", apiErr.Message)
		}
	}

	if d.HasChange(groupMemberIdsKey) {
		memberIds := []string{}
		members := d.Get(groupMemberIdsKey).(*schema.Set).List()
		for _, member := range members {
			memberIds = append(memberIds, member.(string))
		}

		_, apiErr, err := grps.SetMembers(
			ctx,
			g.Id,
			0,
			memberIds,
			groups.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating members on group: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating members on group: %s", apiErr.Message)
		}
	}

	return convertGroupToResourceData(g, d)
}

func resourceGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	g := convertResourceDataToGroup(d, md)
	grps := groups.NewClient(client)

	_, apiErr, err := grps.Delete(ctx, g.Id)
	if err != nil {
		return diag.Errorf("error deleting group: %s", err.Error())
	}
	if apiErr != nil {
		return diag.Errorf("error deleting group: %s", apiErr.Message)
	}

	return nil
}
