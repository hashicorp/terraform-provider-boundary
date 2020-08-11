package provider

import (
	"fmt"

	"github.com/hashicorp/boundary/api/groups"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	groupNameKey        = "name"
	groupDescriptionKey = "description"
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

	d.SetId(g.Id)

	return nil
}

// convertResourceDataToGroup returns a localy built Group using the values provided in the ResourceData.
func convertResourceDataToGroup(d *schema.ResourceData) *groups.Group {
	g := &groups.Group{}
	if descVal, ok := d.GetOk(groupDescriptionKey); ok {
		g.Description = descVal.(string)
	}
	if nameVal, ok := d.GetOk(groupNameKey); ok {
		g.Name = nameVal.(string)
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

	grps := groups.NewGroupsClient(client)

	g := convertResourceDataToGroup(d)

	g, apiErr, err := grps.Create(ctx, groups.WithName(g.Name), groups.WithDescription(g.Description))
	if err != nil {
		return fmt.Errorf("error calling new group: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error creating group: %s", apiErr.Message)
	}

	d.SetId(g.Id)

	return nil
}

func resourceGroupRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	grps := groups.NewGroupsClient(client)

	g := convertResourceDataToGroup(d)

	g, apiErr, err := grps.Read(ctx, g.Id)
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

	grps := groups.NewGroupsClient(client)

	g := convertResourceDataToGroup(d)

	if d.HasChange(groupNameKey) {
		g.Name = d.Get(groupNameKey).(string)
	}

	if d.HasChange(groupDescriptionKey) {
		g.Description = d.Get(groupDescriptionKey).(string)
	}

	g, apiErr, err := grps.Update(
		ctx,
		g.Id,
		0,
		groups.WithAutomaticVersioning(),
		groups.WithName(g.Name),
		groups.WithDescription(g.Description))
	if err != nil {
		return err
	}
	if apiErr != nil {
		return fmt.Errorf("error updating group: %s\n   Invalid request fields: %v\n", apiErr.Message, apiErr.Details.RequestFields)
	}

	return convertGroupToResourceData(g, d)
}

func resourceGroupDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	grps := groups.NewGroupsClient(client)

	g := convertResourceDataToGroup(d)

	_, apiErr, err := grps.Delete(ctx, g.Id)
	if err != nil {
		return fmt.Errorf("error deleting group: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error deleting group: %s", apiErr.Message)
	}

	return nil
}
