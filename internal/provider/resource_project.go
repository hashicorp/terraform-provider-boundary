package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/watchtower/api/scopes"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectCreate,
		Read:   resourceProjectRead,
		Delete: resourceProjectDelete,

		// TODO: Add the ability to define a parent org instead of using one defined in the provider.
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

// projectToResourceData populates the provided ResourceData with the appropriate values from the provided Project.
// The project passed into thie function should be one read from the watchtower API with all fields populated.
func projectToResourceData(p *scopes.Project, d *schema.ResourceData) error {
	if p.Name != nil {
		if err := d.Set("name", p.Name); err != nil {
			return err
		}
	}
	if p.Description != nil {
		if err := d.Set("description", p.Description); err != nil {
			return err
		}
	}
	d.SetId(p.Id)
	return nil
}

// resourceDataToProject returns a localy built Project using the values provided in the ResourceData.
func resourceDataToProject(d *schema.ResourceData) *scopes.Project {
	p := &scopes.Project{}
	if descVal, ok := d.GetOk("description"); ok {
		desc := descVal.(string)
		p.Description = &desc
	}
	if nameVal, ok := d.GetOk("name"); ok {
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
	o := &scopes.Organization{
		Client: client,
	}
	p := resourceDataToProject(d)
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

	o := &scopes.Organization{
		Client: client,
	}
	p := &scopes.Project{Id: d.Id()}
	p, _, err := o.ReadProject(ctx, p)
	if err != nil {
		return err
	}
	return projectToResourceData(p, d)
}

func resourceProjectDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO: Implement
	return nil
}
