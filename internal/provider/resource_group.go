package provider

import (
	"fmt"

	"github.com/hashicorp/boundary/api/groups"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	groupNameKey        = "name"
	groupDescriptionKey = "description"
	groupScopeIDKey     = "scope_id"
	groupMemberIDsKey   = "member_ids"
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
			groupScopeIDKey: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			groupMemberIDsKey: {
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

	// TODO when calling SetMembers() on a group the API returns
	// a nil ScopeInfo so we are checking here to ensure
	// we catch it. This work around can possibly ignore
	// an updated scope ID when updating members and scope
	// of a group simultaniously.
	if g.Scope != nil && g.Scope.Id != "" {
		if err := d.Set(groupScopeIDKey, g.Scope.Id); err != nil {
			return err
		}
	}

	if g.MemberIds != nil {
		if err := d.Set(groupMemberIDsKey, g.MemberIds); err != nil {
			return err
		}
	}

	d.SetId(g.Id)

	return nil
}

// convertResourceDataToGroup returns a localy built Group using the values provided in the ResourceData.
func convertResourceDataToGroup(d *schema.ResourceData, meta *metaData) *groups.Group {
	g := &groups.Group{Scope: &scopes.ScopeInfo{}}

	if descVal, ok := d.GetOk(groupDescriptionKey); ok {
		g.Description = descVal.(string)
	}

	if nameVal, ok := d.GetOk(groupNameKey); ok {
		g.Name = nameVal.(string)
	}

	if scopeIDVal, ok := d.GetOk(groupScopeIDKey); ok {
		g.Scope.Id = scopeIDVal.(string)
	}

	if val, ok := d.GetOk(groupMemberIDsKey); ok {
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
	grps := groups.NewGroupsClient(client)

	memIDs := g.MemberIds

	g, apiErr, err := grps.Create(
		ctx,
		groups.WithScopeId(g.Scope.Id),
		groups.WithName(g.Name),
		groups.WithDescription(g.Description))
	if err != nil {
		return fmt.Errorf("error creating group: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error creating group: %s", apiErr.Message)
	}

	if len(memIDs) > 0 {
		g, apiErr, err = grps.SetMembers(
			ctx,
			g.Id,
			g.Version,
			memIDs,
			groups.WithScopeId(g.Scope.Id))
		if apiErr != nil {
			return fmt.Errorf("error setting principals on role: %s\n", apiErr.Message)
		}
		if err != nil {
			return fmt.Errorf("error setting principals on role: %s\n", err)
		}
	}

	return convertGroupToResourceData(g, d)
}

func resourceGroupRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	g := convertResourceDataToGroup(d, md)
	grps := groups.NewGroupsClient(client)

	g, apiErr, err := grps.Read(ctx, g.Id, groups.WithScopeId(g.Scope.Id))
	if err != nil {
		return fmt.Errorf("error reading group: %s", err.Error())
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
	grps := groups.NewGroupsClient(client)

	if d.HasChange(groupNameKey) {
		g.Name = d.Get(groupNameKey).(string)
	}

	if d.HasChange(groupDescriptionKey) {
		g.Description = d.Get(groupDescriptionKey).(string)
	}

	g.Scope.Id = d.Get(groupScopeIDKey).(string)

	g, apiErr, err := grps.Update(
		ctx,
		g.Id,
		0,
		groups.WithScopeId(g.Scope.Id),
		groups.WithAutomaticVersioning(),
		groups.WithName(g.Name),
		groups.WithDescription(g.Description))
	if err != nil {
		return err
	}
	if apiErr != nil {
		return fmt.Errorf("%+v\n", apiErr.Message)
	}

	if d.HasChange(groupMemberIDsKey) {
		memberIds := []string{}
		members := d.Get(groupMemberIDsKey).(*schema.Set).List()
		for _, member := range members {
			memberIds = append(memberIds, member.(string))
		}

		g, apiErr, err = grps.SetMembers(
			ctx,
			g.Id,
			g.Version,
			memberIds,
			groups.WithScopeId(g.Scope.Id))
		if apiErr != nil || err != nil {
			return fmt.Errorf("error updating members on group:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
		}
	}

	return convertGroupToResourceData(g, d)
}

func resourceGroupDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	g := convertResourceDataToGroup(d, md)
	grps := groups.NewGroupsClient(client)

	_, apiErr, err := grps.Delete(ctx, g.Id, groups.WithScopeId(g.Scope.Id))
	if err != nil {
		return fmt.Errorf("error deleting group: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error deleting group: %s", apiErr.Message)
	}

	return nil
}
