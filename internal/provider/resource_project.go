package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/boundary/api/scopes"
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
func convertProjectToResourceData(p *scopes.Project, d *schema.ResourceData) error {
	if p.Name != nil {
		if err := d.Set(projectNameKey, p.Name); err != nil {
			return err
		}
	}
	if p.Description != nil {
		if err := d.Set(projectDescriptionKey, p.Description); err != nil {
			return err
		}
	}
	d.SetId(p.Id)
	return nil
}

// convertResourceDataToProject returns a localy built Project using the values provided in the ResourceData.
func convertResourceDataToProject(d *schema.ResourceData) *scopes.Project {
	p := &scopes.Project{}
	if descVal, ok := d.GetOk(projectDescriptionKey); ok {
		desc := descVal.(string)
		p.Description = &desc
	}
	if nameVal, ok := d.GetOk(projectNameKey); ok {
		name := nameVal.(string)
		p.Name = &name
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

	// The org id is declared in the client, so no need to specify that here.
	o := &scopes.Org{
		Client: client,
	}
	p := convertResourceDataToProject(d)
	p, _, err := o.CreateProject(ctx, p)
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

	o := &scopes.Org{
		Client: client,
	}
	p := &scopes.Project{Id: d.Id()}
	p, _, err := o.ReadProject(ctx, p)
	if err != nil {
		return err
	}
	return convertProjectToResourceData(p, d)
}

func resourceProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Org{
		Client: client,
	}
	p := &scopes.Project{
		Id: d.Id(),
	}

	if d.HasChange(projectDescriptionKey) {
		desc := d.Get(projectDescriptionKey).(string)
		if desc == "" {
			p.SetDefault(projectDescriptionKey)
		} else {
			p.Description = &desc
		}
	}

	if d.HasChange(projectNameKey) {
		name := d.Get(projectNameKey).(string)
		if name == "" {
			p.SetDefault(projectNameKey)
		} else {
			p.Name = &name
		}
	}

	p, _, err := o.UpdateProject(ctx, p)
	if err != nil {
		return err
	}

	return convertProjectToResourceData(p, d)
}

func resourceProjectDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Org{
		Client: client,
	}
	p := convertResourceDataToProject(d)
	_, _, err := o.DeleteProject(ctx, p)
	if err != nil {
		return fmt.Errorf("failed deleting project: %w", err)
	}
	return nil
}
