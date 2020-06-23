package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/watchtower/api/roles"
	"github.com/hashicorp/watchtower/api/scopes"
)

const (
	roleNameKey        = "name"
	roleDescriptionKey = "description"
)

func resourceRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceRoleCreate,
		Read:   resourceRoleRead,
		Update: resourceRoleUpdate,
		Delete: resourceRoleDelete,
		Schema: map[string]*schema.Schema{
			roleNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			roleDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}

}

// convertRoleToResourceData creates a ResourceData type from a Role
func convertRoleToResourceData(u *roles.Role, d *schema.ResourceData) error {
	if u.Name != nil {
		if err := d.Set(roleNameKey, u.Name); err != nil {
			return err
		}
	}

	if u.Description != nil {
		if err := d.Set(roleDescriptionKey, u.Description); err != nil {
			return err
		}
	}

	d.SetId(u.Id)

	return nil
}

// convertResourceDataToRole returns a localy built Role using the values provided in the ResourceData.
func convertResourceDataToRole(d *schema.ResourceData) *roles.Role {
	u := &roles.Role{}
	if descVal, ok := d.GetOk(roleDescriptionKey); ok {
		desc := descVal.(string)
		u.Description = &desc
	}
	if nameVal, ok := d.GetOk(roleNameKey); ok {
		name := nameVal.(string)
		u.Name = &name
	}

	if d.Id() != "" {
		u.Id = d.Id()
	}

	return u
}

func resourceRoleCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Organization{
		Client: client,
	}

	u := convertResourceDataToRole(d)

	u, apiErr, err := o.CreateRole(ctx, u)
	if err != nil {
		return fmt.Errorf("error calling new role: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error creating role: %s", *apiErr.Message)
	}

	d.SetId(u.Id)

	return nil
}

func resourceRoleRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Organization{
		Client: client,
	}

	u := convertResourceDataToRole(d)

	u, apiErr, err := o.ReadRole(ctx, u)
	if err != nil {
		return fmt.Errorf("error reading role: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error reading role: %s", *apiErr.Message)
	}

	return convertRoleToResourceData(u, d)
}

func resourceRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Organization{
		Client: client,
	}

	u := convertResourceDataToRole(d)

	if d.HasChange(roleNameKey) {
		n := d.Get(roleNameKey).(string)
		u.Name = &n
	}

	if d.HasChange(roleDescriptionKey) {
		d := d.Get(roleDescriptionKey).(string)
		u.Description = &d
	}

	u, apiErr, err := o.UpdateRole(ctx, u)
	if err != nil {
		return err
	}
	if apiErr != nil {
		return fmt.Errorf("error updating role: %s\n   Invalid request fields: %v\n", *apiErr.Message, apiErr.Details.RequestFields)
	}

	return convertRoleToResourceData(u, d)
}

func resourceRoleDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Organization{
		Client: client,
	}

	u := convertResourceDataToRole(d)

	_, apiErr, err := o.DeleteRole(ctx, u)
	if err != nil {
		return fmt.Errorf("error deleting role: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error deleting role: %s", *apiErr.Message)
	}

	return nil
}
