package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	projectDescriptionKey = "description"
	projectNameKey        = "name"
	projectScopeIDKey     = "scope_id"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectCreate,
		Read:   resourceProjectRead,
		Update: resourceProjectUpdate,
		Delete: resourceProjectDelete,

		Schema: map[string]*schema.Schema{
			projectNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			projectDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			projectScopeIDKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

// convertProjectToResourceData populates the provided ResourceData with the appropriate values from the provided Project.
// The project passed into thie function should be one read from the boundary API with all fields populated.
func convertProjectToResourceData(p *scopes.Scope, d *schema.ResourceData) error {
	if p.Name != "" {
		if err := d.Set(projectNameKey, p.Name); err != nil {
			return err
		}
	}

	if p.Description != "" {
		if err := d.Set(projectDescriptionKey, p.Description); err != nil {
			return err
		}
	}

	if p.Scope != nil && p.Scope.Id != "" {
		if err := d.Set(groupScopeIDKey, p.Scope.Id); err != nil {
			return err
		}
	}

	d.SetId(p.Id)
	return nil
}

// convertResourceDataToProject returns a localy built Project using the values provided in the ResourceData.
func convertResourceDataToProject(d *schema.ResourceData) (*scopes.Scope, error) {
	p := &scopes.Scope{Scope: &scopes.ScopeInfo{}}

	if descVal, ok := d.GetOk(projectDescriptionKey); ok {
		p.Description = descVal.(string)
	}

	if nameVal, ok := d.GetOk(projectNameKey); ok {
		p.Name = nameVal.(string)
	}

	if scopeIDVal, ok := d.GetOk(projectScopeIDKey); ok {
		// boundary only knows about scope_id, and here we want to ensure
		// we manage a project within an organization
		if !strings.HasPrefix(scopeIDVal.(string), "o_") {
			return p, fmt.Errorf("can not use scope_id '%s' for project management", scopeIDVal.(string))
		}
		p.Scope.Id = scopeIDVal.(string)
	}

	if d.Id() != "" {
		p.Id = d.Id()
	}

	return p, nil
}

func resourceProjectCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	scp := scopes.NewScopesClient(client)
	p, err := convertResourceDataToProject(d)
	if err != nil {
		return err
	}

	p, _, err = scp.Create(
		ctx,
		p.Scope.Id,
		scopes.WithName(p.Name),
		scopes.WithDescription(p.Description))
	if err != nil {
		return err
	}
	d.SetId(p.Id)

	return nil
}

func resourceProjectRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	scp := scopes.NewScopesClient(client)

	p, _, err := scp.Read(ctx, d.Id())
	if err != nil {
		return err
	}
	return convertProjectToResourceData(p, d)
}

func resourceProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	//projClient := client.Clone()
	//projClient.SetScopeId(d.Id())
	scp := scopes.NewScopesClient(client) //projClient)
	p, err := convertResourceDataToProject(d)
	if err != nil {
		return err
	}

	if d.HasChange(projectDescriptionKey) {
		desc := d.Get(projectDescriptionKey).(string)
		p.Description = desc
	}

	if d.HasChange(projectNameKey) {
		name := d.Get(projectNameKey).(string)
		p.Name = name
	}

	p, _, err = scp.Update(
		ctx,
		d.Id(),
		0,
		scopes.WithAutomaticVersioning(),
		scopes.WithDescription(p.Description),
		scopes.WithName(p.Name),
	)
	if err != nil {
		return err
	}

	return convertProjectToResourceData(p, d)
}

func resourceProjectDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	//projClient := client.Clone()
	//projClient.SetScopeId(d.Id())
	scp := scopes.NewScopesClient(client) //projClient)
	p, err := convertResourceDataToProject(d)
	if err != nil {
		return err
	}

	_, _, err = scp.Delete(ctx, p.Id)
	if err != nil {
		return fmt.Errorf("failed deleting project: %w", err)
	}
	return nil
}
