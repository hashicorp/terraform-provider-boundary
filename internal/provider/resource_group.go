package provider

import (
	"fmt"

	"github.com/hashicorp/boundary/api/groups"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	groupNameKey        = "name"
	groupDescriptionKey = "description"
	groupScopeIdKey     = "scope_id"
	groupMemberIdsKey   = "member_ids"
)

func resourceGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceGroupCreate,
		Read:   resourceGroupRead,
		Update: resourceGroupUpdate,
		Delete: resourceGroupDelete,
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
func convertGroupToResourceData(g *groups.Group, d *schema.ResourceData) error {
	if g.Name != "" {
		if err := d.Set(groupNameKey, g.Name); err != nil {
			return err
		}
	}

	if g.Description != "" {
		if err := d.Set(groupDescriptionKey, g.Description); err != nil {
			return err
		}
	}

	if g.ScopeId != "" {
		if err := d.Set(groupScopeIdKey, g.ScopeId); err != nil {
			return err
		}
	}

	if g.MemberIds != nil {
		if err := d.Set(groupMemberIdsKey, g.MemberIds); err != nil {
			return err
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

func resourceGroupCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	g := convertResourceDataToGroup(d, md)
	grps := groups.NewClient(client)

	memIds := g.MemberIds

	g, apiErr, err := grps.Create(
		ctx,
		g.ScopeId,
		groups.WithName(g.Name),
		groups.WithDescription(g.Description))
	if err != nil {
		return fmt.Errorf("error creating group: %w", err)
	}
	if apiErr != nil {
		return fmt.Errorf("error creating group: %s", apiErr.Message)
	}

	if len(memIds) > 0 {
		g, apiErr, err = grps.SetMembers(
			ctx,
			g.Id,
			g.Version,
			memIds)
		if apiErr != nil {
			return fmt.Errorf("error setting principals on role: %s", apiErr.Message)
		}
		if err != nil {
			return fmt.Errorf("error setting principals on role: %w", err)
		}
	}

	return convertGroupToResourceData(g, d)
}

func resourceGroupRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	g := convertResourceDataToGroup(d, md)
	grps := groups.NewClient(client)

	g, apiErr, err := grps.Read(ctx, g.Id)
	if err != nil {
		return fmt.Errorf("error reading group: %w", err)
	}
	if apiErr != nil {
		return fmt.Errorf("error reading group: %s", apiErr.Message)
	}

	return convertGroupToResourceData(g, d)
}

func resourceGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

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
			return fmt.Errorf("error updating group: %w", err)
		}
		if apiErr != nil {
			return fmt.Errorf("error updating group: %s", apiErr.Message)
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
			return fmt.Errorf("error updating members on group: %w", err)
		}
		if apiErr != nil {
			return fmt.Errorf("error updating members on group: %s", apiErr.Message)
		}
	}

	return convertGroupToResourceData(g, d)
}

func resourceGroupDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	g := convertResourceDataToGroup(d, md)
	grps := groups.NewClient(client)

	_, apiErr, err := grps.Delete(ctx, g.Id)
	if err != nil {
		return fmt.Errorf("error deleting group: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error deleting group: %s", apiErr.Message)
	}

	return nil
}
