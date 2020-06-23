package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/watchtower/api/groups"
	"github.com/hashicorp/watchtower/api/scopes"
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
func convertGroupToResourceData(u *groups.Group, d *schema.ResourceData) error {
	if u.Name != nil {
		if err := d.Set(groupNameKey, u.Name); err != nil {
			return err
		}
	}

	if u.Description != nil {
		if err := d.Set(groupDescriptionKey, u.Description); err != nil {
			return err
		}
	}

	d.SetId(u.Id)

	return nil
}

// convertResourceDataToGroup returns a localy built Group using the values provided in the ResourceData.
func convertResourceDataToGroup(d *schema.ResourceData) *groups.Group {
	u := &groups.Group{}
	if descVal, ok := d.GetOk(groupDescriptionKey); ok {
		desc := descVal.(string)
		u.Description = &desc
	}
	if nameVal, ok := d.GetOk(groupNameKey); ok {
		name := nameVal.(string)
		u.Name = &name
	}

	if d.Id() != "" {
		u.Id = d.Id()
	}

	return u
}

func resourceGroupCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Organization{
		Client: client,
	}

	u := convertResourceDataToGroup(d)

	u, apiErr, err := o.CreateGroup(ctx, u)
	if err != nil {
		return fmt.Errorf("error calling new group: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error creating group: %s", *apiErr.Message)
	}

	d.SetId(u.Id)

	return nil
}

func resourceGroupRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Organization{
		Client: client,
	}

	u := convertResourceDataToGroup(d)

	u, apiErr, err := o.ReadGroup(ctx, u)
	if err != nil {
		return fmt.Errorf("error reading group: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error reading group: %s", *apiErr.Message)
	}

	return convertGroupToResourceData(u, d)
}

func resourceGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Organization{
		Client: client,
	}

	u := convertResourceDataToGroup(d)

	if d.HasChange(groupNameKey) {
		n := d.Get(groupNameKey).(string)
		u.Name = &n
	}

	if d.HasChange(groupDescriptionKey) {
		d := d.Get(groupDescriptionKey).(string)
		u.Description = &d
	}

	u, apiErr, err := o.UpdateGroup(ctx, u)
	if err != nil {
		return err
	}
	if apiErr != nil {
		return fmt.Errorf("error updating group: %s\n   Invalid request fields: %v\n", *apiErr.Message, apiErr.Details.RequestFields)
	}

	return convertGroupToResourceData(u, d)
}

func resourceGroupDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Organization{
		Client: client,
	}

	u := convertResourceDataToGroup(d)

	_, apiErr, err := o.DeleteGroup(ctx, u)
	if err != nil {
		return fmt.Errorf("error deleting group: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error deleting group: %s", *apiErr.Message)
	}

	return nil
}
