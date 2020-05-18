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

func resourceProjectCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	o := &scopes.Organization{
		Client: client,
	}
	if parentValue, ok := d.GetOk("parent"); ok && parentValue.(string) != "" {
		o.Id = parentValue.(string)
	}
	p := &scopes.Project{}
	if descVal, ok := d.GetOk("description"); ok {
		desc := descVal.(string)
		p.Description = &desc
	}
	if nameVal, ok := d.GetOk("name"); ok {
		name := nameVal.(string)
		p.Name = &name
	}
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
	if parentValue, ok := d.GetOk("parent"); ok && parentValue.(string) != "" {
		o.Id = parentValue.(string)
	}
	p := &scopes.Project{Id: d.Id()}
	p, _, err := o.ReadProject(ctx, p)
	if err != nil {
		return err
	}

	d.Set("name", p.Name)
	d.Set("description", p.Description)
	return nil
}

func resourceProjectDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO: Implement
	return nil
}
