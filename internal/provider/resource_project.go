package provider

import (
	"fmt"

	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	projectDescriptionKey = "description"
	projectNameKey        = "name"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectCreate,
		Read:   resourceProjectRead,
		Update: resourceProjectUpdate,
		Delete: resourceProjectDelete,

		// TODO: Add the ability to define a parent org instead of using one defined in the provider.
		Schema: map[string]*schema.Schema{
			projectNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			projectDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
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
	d.SetId(p.Id)
	return nil
}

// convertResourceDataToProject returns a localy built Project using the values provided in the ResourceData.
func convertResourceDataToProject(d *schema.ResourceData) *scopes.Scope {
	p := &scopes.Scope{}
	if descVal, ok := d.GetOk(projectDescriptionKey); ok {
		p.Description = descVal.(string)
	}
	if nameVal, ok := d.GetOk(projectNameKey); ok {
		p.Name = nameVal.(string)
	}
	if d.Id() != "" {
		p.Id = d.Id()
	}
	return p
}

func resourceProjectCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	scp := scopes.NewScopesClient(client)
	p := convertResourceDataToProject(d)
	p, _, err := scp.Create(
		ctx,
		client.ScopeId(),
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

	projClient := client.Clone()
	projClient.SetScopeId(d.Id())
	scp := scopes.NewScopesClient(projClient)
	p := convertResourceDataToProject(d)

	if d.HasChange(projectDescriptionKey) {
		desc := d.Get(projectDescriptionKey).(string)
		p.Description = desc
	}

	if d.HasChange(projectNameKey) {
		name := d.Get(projectNameKey).(string)
		p.Name = name
	}

	p, _, err := scp.Update(
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

	projClient := client.Clone()
	projClient.SetScopeId(d.Id())
	scp := scopes.NewScopesClient(projClient)
	p := convertResourceDataToProject(d)

	_, _, err := scp.Delete(ctx, p.Id)
	if err != nil {
		return fmt.Errorf("failed deleting project: %w", err)
	}
	return nil
}
