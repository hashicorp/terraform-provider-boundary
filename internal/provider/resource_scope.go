package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	scopeDescriptionKey = "description"
	scopeNameKey        = "name"
	scopeScopeIdKey     = "scope_id"
)

func resourceScope() *schema.Resource {
	return &schema.Resource{
		Create: resourceScopeCreate,
		Read:   resourceScopeRead,
		Update: resourceScopeUpdate,
		Delete: resourceScopeDelete,

		Schema: map[string]*schema.Schema{
			scopeNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			scopeDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			scopeScopeIdKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

// convertScopeToResourceData populates the provided ResourceData with the appropriate values from the provided Scope.
// The scope passed into thie function should be one read from the boundary API with all fields populated.
func convertScopeToResourceData(p *scopes.Scope, d *schema.ResourceData) error {
	if p.Name != "" {
		if err := d.Set(scopeNameKey, p.Name); err != nil {
			return err
		}
	}

	if p.Description != "" {
		if err := d.Set(scopeDescriptionKey, p.Description); err != nil {
			return err
		}
	}

	if p.ScopeId != "" {
		if err := d.Set(scopeScopeIdKey, p.ScopeId); err != nil {
			return err
		}
	}

	d.SetId(p.Id)
	return nil
}

// convertResourceDataToScope returns a localy built Scope using the values provided in the ResourceData.
func convertResourceDataToScope(d *schema.ResourceData) (*scopes.Scope, error) {
	p := new(scopes.Scope)

	if descVal, ok := d.GetOk(scopeDescriptionKey); ok {
		p.Description = descVal.(string)
	}

	if nameVal, ok := d.GetOk(scopeNameKey); ok {
		p.Name = nameVal.(string)
	}

	if scopeIdVal, ok := d.GetOk(scopeScopeIdKey); ok {
		switch {
		case strings.HasPrefix(d.Id(), "o_"), d.Id() == "global":
			if scopeIdVal.(string) != "global" {
				return p, fmt.Errorf("cannot use scope_id %q for scope management", "global")
			}
		case strings.HasPrefix(d.Id(), "p_"):
			if !strings.HasPrefix(scopeIdVal.(string), "o_") {
				return p, fmt.Errorf("cannot use scope_id %q for scope management", scopeIdVal.(string))
			}
		}
		p.ScopeId = scopeIdVal.(string)
	}

	if d.Id() != "" {
		p.Id = d.Id()
	}

	return p, nil
}

func resourceScopeCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	scp := scopes.NewClient(client)
	p, err := convertResourceDataToScope(d)
	if err != nil {
		return err
	}

	p, apiErr, err := scp.Create(
		ctx,
		p.ScopeId,
		scopes.WithName(p.Name),
		scopes.WithDescription(p.Description))
	if err != nil {
		return fmt.Errorf("error calling new scope: %w", err)
	}
	if apiErr != nil {
		return fmt.Errorf("error creating scope: %s", apiErr.Message)
	}
	d.SetId(p.Id)

	return nil
}

func resourceScopeRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	scp := scopes.NewClient(client)

	p, apiErr, err := scp.Read(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("error calling read scope: %w", err)
	}
	if apiErr != nil {
		return fmt.Errorf("error reading scope: %s", apiErr.Message)
	}
	return convertScopeToResourceData(p, d)
}

func resourceScopeUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	scp := scopes.NewClient(client)
	p, err := convertResourceDataToScope(d)
	if err != nil {
		return err
	}

	if d.HasChange(scopeDescriptionKey) {
		desc := d.Get(scopeDescriptionKey).(string)
		p.Description = desc
	}

	if d.HasChange(scopeNameKey) {
		name := d.Get(scopeNameKey).(string)
		p.Name = name
	}

	p, apiErr, err := scp.Update(
		ctx,
		d.Id(),
		0,
		scopes.WithAutomaticVersioning(true),
		scopes.WithDescription(p.Description),
		scopes.WithName(p.Name),
	)
	if err != nil {
		return fmt.Errorf("error calling update scope: %w", err)
	}
	if apiErr != nil {
		return fmt.Errorf("error updating scope: %s", apiErr.Message)
	}

	return convertScopeToResourceData(p, d)
}

func resourceScopeDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	scp := scopes.NewClient(client)
	p, err := convertResourceDataToScope(d)
	if err != nil {
		return err
	}

	_, apiErr, err := scp.Delete(ctx, p.Id)
	if err != nil {
		return fmt.Errorf("error calling delete scope: %w", err)
	}
	if apiErr != nil {
		return fmt.Errorf("error deleting scope: %s", apiErr.Message)
	}
	return nil
}
