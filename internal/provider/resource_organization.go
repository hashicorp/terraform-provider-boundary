package provider

import (
	"fmt"

	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	organizationDescriptionKey = "description"
	organizationNameKey        = "name"
)

func resourceOrganization() *schema.Resource {
	return &schema.Resource{
		Create: resourceOrganizationCreate,
		Read:   resourceOrganizationRead,
		Update: resourceOrganizationUpdate,
		Delete: resourceOrganizationDelete,

		Schema: map[string]*schema.Schema{
			organizationNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			organizationDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

// convertOrganizationToResourceData populates the provided ResourceData with the appropriate values from the provided Project.
// The organization passed into thie function should be one read from the boundary API with all fields populated.
func convertOrganizationToResourceData(o *scopes.Scope, d *schema.ResourceData) error {
	if o.Name != "" {
		if err := d.Set(organizationNameKey, o.Name); err != nil {
			return err
		}
	}

	if o.Description != "" {
		if err := d.Set(organizationDescriptionKey, o.Description); err != nil {
			return err
		}
	}

	d.SetId(o.Id)
	return nil
}

// convertResourceDataToOrganization returns a localy built Project using the values provided in the ResourceData.
func convertResourceDataToOrganization(d *schema.ResourceData) (*scopes.Scope, error) {
	o := &scopes.Scope{Scope: &scopes.ScopeInfo{}}

	if descVal, ok := d.GetOk(organizationDescriptionKey); ok {
		o.Description = descVal.(string)
	}

	if nameVal, ok := d.GetOk(organizationNameKey); ok {
		o.Name = nameVal.(string)
	}

	if d.Id() != "" {
		o.Id = d.Id()
	}

	return o, nil
}

func resourceOrganizationCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	scp := scopes.NewScopesClient(client)
	o, err := convertResourceDataToOrganization(d)
	if err != nil {
		return err
	}

	o, _, err = scp.Create(
		ctx,
		client.ScopeId(),
		scopes.WithName(o.Name),
		scopes.WithDescription(o.Description))
	if err != nil {
		return err
	}
	d.SetId(o.Id)

	return nil
}

func resourceOrganizationRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	scp := scopes.NewScopesClient(client)

	o, _, err := scp.Read(ctx, d.Id())
	if err != nil {
		return err
	}
	return convertOrganizationToResourceData(o, d)
}

func resourceOrganizationUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	projClient := client.Clone()
	projClient.SetScopeId(d.Id())
	scp := scopes.NewScopesClient(projClient)
	o, err := convertResourceDataToOrganization(d)
	if err != nil {
		return err
	}

	if d.HasChange(organizationDescriptionKey) {
		desc := d.Get(organizationDescriptionKey).(string)
		o.Description = desc
	}

	if d.HasChange(organizationNameKey) {
		name := d.Get(organizationNameKey).(string)
		o.Name = name
	}

	o, _, err = scp.Update(
		ctx,
		d.Id(),
		0,
		scopes.WithAutomaticVersioning(),
		scopes.WithDescription(o.Description),
		scopes.WithName(o.Name),
	)
	if err != nil {
		return err
	}

	return convertOrganizationToResourceData(o, d)
}

func resourceOrganizationDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	projClient := client.Clone()
	projClient.SetScopeId(d.Id())
	scp := scopes.NewScopesClient(projClient)
	o, err := convertResourceDataToOrganization(d)
	if err != nil {
		return err
	}

	_, _, err = scp.Delete(ctx, o.Id)
	if err != nil {
		return fmt.Errorf("failed deleting organization: %w", err)
	}
	return nil
}
