package provider

import (
	"fmt"

	"github.com/hashicorp/boundary/api/roles"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	roleNameKey        = "name"
	roleDescriptionKey = "description"
	rolePrincipalsKey  = "principals"
	roleGrantsKey      = "grants"
	roleScopeIDKey     = "scope_id"
)

func resourceRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRoleCreate,
		ReadContext:   resourceRoleRead,
		UpdateContext: resourceRoleUpdate,
		DeleteContext: resourceRoleDelete,
		Schema: map[string]*schema.Schema{
			roleNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			roleDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			rolePrincipalsKey: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			roleGrantsKey: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			roleScopeIDKey: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

// convertRoleToResourceData creates a ResourceData type from a Role
func convertRoleToResourceData(r *roles.Role, d *schema.ResourceData) error {
	if r.Name != "" {
		if err := d.Set(roleNameKey, r.Name); err != nil {
			return err
		}
	}

	if r.Description != "" {
		if err := d.Set(roleDescriptionKey, r.Description); err != nil {
			return err
		}
	}

	if r.PrincipalIds != nil {
		if err := d.Set(rolePrincipalsKey, r.PrincipalIds); err != nil {
			return err
		}
	}

	if r.Grants != nil {
		grants := []string{}
		for _, grant := range r.Grants {
			grants = append(grants, grant.Raw)
		}
		if err := d.Set(roleGrantsKey, grants); err != nil {
			return err
		}
	}

	if r.Scope.Id != "" {
		if err := d.Set(roleScopeIDKey, r.Scope.Id); err != nil {
			return err
		}
	}

	d.SetId(r.Id)

	return nil
}

// convertResourceDataToRole returns a localy built Role using the values provided in the ResourceData.
func convertResourceDataToRole(d *schema.ResourceData) *roles.Role {
	r := &roles.Role{Scope: &scopes.ScopeInfo{}}

	if projIDVal, ok := d.GetOk(roleScopeIDKey); ok {
		r.Scope.Id = projIDVal.(string)
	}

	if descVal, ok := d.GetOk(roleDescriptionKey); ok {
		r.Description = descVal.(string)
	}

	if nameVal, ok := d.GetOk(roleNameKey); ok {
		r.Name = nameVal.(string)
	}

	if val, ok := d.GetOk(rolePrincipalsKey); ok {
		principalIds := val.(*schema.Set).List()
		for _, i := range principalIds {
			r.PrincipalIds = append(r.PrincipalIds, i.(string))
		}
	}

	if val, ok := d.GetOk(roleGrantsKey); ok {
		grants := val.(*schema.Set).List()
		for _, i := range grants {
			g := &roles.Grant{Raw: i.(string)}
			r.Grants = append(r.Grants, g)
		}
	}

	if d.Id() != "" {
		r.Id = d.Id()
	}

	return r
}

func resourceRoleCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	r := convertResourceDataToRole(d)
	rolesClient := roles.NewClient(client)

	principals := r.PrincipalIds
	grants := []string{}
	for _, g := range r.Grants {
		grants = append(grants, g.Raw)
	}

	r, apiErr, err := rolesClient.Create(
		ctx,
		roles.WithName(r.Name),
		roles.WithDescription(r.Description),
		roles.WithScopeId(r.Scope.Id))
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
		r, apiErr, err = rolesClient.AddGrants(
			ctx,
			r.Id,
			r.Version,
			grants,
			roles.WithScopeId(r.Scope.Id))
		if apiErr != nil {
			return fmt.Errorf("error setting grants on role:: %s\n", apiErr.Message)
		}
		if err != nil {
			return fmt.Errorf("error setting grants on role: %s\n", err)
		}
	}

	if len(principals) > 0 {
		r, apiErr, err = rolesClient.SetPrincipals(
			ctx,
			r.Id,
			r.Version,
			principals,
			roles.WithScopeId(r.Scope.Id))
		if apiErr != nil {
			return fmt.Errorf("error setting principals on role: %s\n", apiErr.Message)
		}
		if err != nil {
			return fmt.Errorf("error setting principals on role: %s\n", err)
		}
	}

	return convertRoleToResourceData(r, d)
}

func resourceRoleRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	r := convertResourceDataToRole(d)
	rolesClient := roles.NewClient(client)

	r, apiErr, err := rolesClient.Read(ctx, r.Id, roles.WithScopeId(r.Scope.Id))
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

	r := convertResourceDataToRole(d)
	rolesClient := roles.NewClient(client)

	if d.HasChange(roleNameKey) {
		r.Name = d.Get(roleNameKey).(string)
	}

	if d.HasChange(roleDescriptionKey) {
		r.Description = d.Get(roleDescriptionKey).(string)
	}

	r, apiErr, err := rolesClient.Update(
		ctx,
		r.Id,
		0,
		roles.WithAutomaticVersioning(),
		roles.WithName(r.Name),
		roles.WithDescription(r.Description),
		roles.WithScopeId(r.Scope.Id))
	if apiErr != nil || err != nil {
		return fmt.Errorf("error updating role:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
	}

	if d.HasChange(roleGrantsKey) {
		grants := []string{}
		grantSet := d.Get(roleGrantsKey).(*schema.Set).List()

		for _, grant := range grantSet {
			grants = append(grants, grant.(string))
		}

		r, apiErr, err = rolesClient.SetGrants(
			ctx,
			r.Id,
			r.Version,
			grants,
			roles.WithScopeId(r.Scope.Id))
		if apiErr != nil || err != nil {
			return fmt.Errorf("error setting grants on role:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
		}
	}

	if d.HasChange(rolePrincipalsKey) {
		principalIds := []string{}
		principals := d.Get(rolePrincipalsKey).(*schema.Set).List()
		for _, principal := range principals {
			principalIds = append(principalIds, principal.(string))
		}

		r, apiErr, err = rolesClient.SetPrincipals(
			ctx,
			r.Id,
			r.Version,
			principalIds,
			roles.WithScopeId(r.Scope.Id))
		if apiErr != nil || err != nil {
			return fmt.Errorf("error updating principal on role:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
		}
	}

	return convertRoleToResourceData(r, d)
}

func resourceRoleDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	r := convertResourceDataToRole(d)
	rolesClient := roles.NewClient(client)

	_, apiErr, err := rolesClient.Delete(ctx, r.Id, roles.WithScopeId(r.Scope.Id))
	if apiErr != nil || err != nil {
		return fmt.Errorf("error deleting role:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
	}

	return nil
}
