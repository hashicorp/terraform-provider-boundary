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
	roleUsersKey       = "users"
	roleGroupsKey      = "groups"
	roleGrantsKey      = "grants"
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
			roleUsersKey: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			roleGroupsKey: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			roleGrantsKey: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
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

	if u.UserIds != nil {
		if err := d.Set(roleUsersKey, u.UserIds); err != nil {
			return err
		}
	}

	if u.GroupIds != nil {
		if err := d.Set(roleGroupsKey, u.GroupIds); err != nil {
			return err
		}
	}

	if u.Grants != nil {
		if err := d.Set(roleGrantsKey, u.Grants); err != nil {
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
	if val, ok := d.GetOk(roleGroupsKey); ok {
		groupIds := val.(*schema.Set).List()
		for _, i := range groupIds {
			u.GroupIds = append(u.GroupIds, i.(string))
		}
	}
	if val, ok := d.GetOk(roleUsersKey); ok {
		userIds := val.(*schema.Set).List()
		for _, i := range userIds {
			u.UserIds = append(u.UserIds, i.(string))
		}
	}
	if val, ok := d.GetOk(roleGrantsKey); ok {
		grants := val.(*schema.Set).List()
		for _, i := range grants {
			u.Grants = append(u.Grants, i.(string))
		}
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

	o := &scopes.Org{
		Client: client,
	}

	r := convertResourceDataToRole(d)
	users := r.UserIds
	groups := r.GroupIds
	grants := r.Grants

	newRole, apiErr, err := o.CreateRole(ctx, r)
	if apiErr != nil {
		return fmt.Errorf("error creating role: %s\n", apiErr.Message)
	}
	if err != nil {
		return fmt.Errorf("error creating role: %s\n", err)
	}

	// on first create CreateRole() returns without err but upon
	// running AddGrants it claims the role is not found. This
	// doesn't occur in the test case but only on a live cluster.
	if len(grants) > 0 {
		r, apiErr, err = newRole.AddGrants(ctx, grants)
		if apiErr != nil {
			return fmt.Errorf("error setting grants on role:: %s\n", apiErr.Message)
		}
		if err != nil {
			return fmt.Errorf("error setting grants on role: %s\n", err)
		}
	}

	if len(users) > 0 || len(groups) > 0 {
		r, apiErr, err = r.SetPrincipals(ctx, groups, users)
		if apiErr != nil {
			return fmt.Errorf("error setting principle on role: %s\n", apiErr.Message)
		}
		if err != nil {
			return fmt.Errorf("error setting principle on role: %s\n", err)
		}
	}

	d.SetId(r.Id)

	return nil
}

func resourceRoleRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Org{
		Client: client,
	}

	r := convertResourceDataToRole(d)

	r, apiErr, err := o.ReadRole(ctx, r)
	if err != nil {
		return fmt.Errorf("error reading role: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error reading role: %s", apiErr.Message)
	}

	return convertRoleToResourceData(r, d)
}

func resourceRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Org{
		Client: client,
	}

	r := convertResourceDataToRole(d)

	if d.HasChange(roleNameKey) {
		n := d.Get(roleNameKey).(string)
		r.Name = &n
	}

	if d.HasChange(roleDescriptionKey) {
		d := d.Get(roleDescriptionKey).(string)
		r.Description = &d
	}

	r.GroupIds = nil
	r.UserIds = nil
	r.Grants = nil

	r, apiErr, err := o.UpdateRole(ctx, r)
	if apiErr != nil || err != nil {
		return fmt.Errorf("error updating role:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
	}

	grants := []string{}
	if d.HasChange(roleGrantsKey) {
		grantSet := d.Get(roleGrantsKey).(*schema.Set).List()
		for _, grant := range grantSet {
			grants = append(grants, grant.(string))
		}
	}

	if d.HasChange(roleGrantsKey) {
		_, apiErr, err := r.SetGrants(ctx, grants)
		if apiErr != nil || err != nil {
			return fmt.Errorf("error setting grants on role:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
		}
	}

	userIDs := []string{}
	if d.HasChange(roleUsersKey) {
		users := d.Get(roleUsersKey).(*schema.Set).List()
		for _, user := range users {
			userIDs = append(userIDs, user.(string))
		}
	}

	groupIDs := []string{}
	if d.HasChange(roleGroupsKey) {
		groups := d.Get(roleGroupsKey).(*schema.Set).List()
		for _, group := range groups {
			groupIDs = append(groupIDs, group.(string))
		}
	}

	if d.HasChange(roleGroupsKey) || d.HasChange(roleUsersKey) {
		r, apiErr, err = r.SetPrincipals(ctx, groupIDs, userIDs)
		if apiErr != nil || err != nil {
			return fmt.Errorf("error updating principle on role:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
		}
	}

	return convertRoleToResourceData(r, d)
}

func resourceRoleDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Org{
		Client: client,
	}

	r := convertResourceDataToRole(d)

	_, apiErr, err := o.DeleteRole(ctx, r)
	if apiErr != nil || err != nil {
		return fmt.Errorf("error deleting role:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
	}

	return nil
}
