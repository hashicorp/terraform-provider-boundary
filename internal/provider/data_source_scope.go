package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	ParentScopeIdKey = "parent_scope_id"
)

func dataSourceScope() *schema.Resource {
	return &schema.Resource{
		Description: "The scope data source allows you to discover an existing Boundary scope by name.",
		ReadContext: dataSourceScopeRead,

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the scope.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The name of the scope to retrieve.",
				Type:        schema.TypeString,
				Required:    true,
			},
			DescriptionKey: {
				Description: "The scope description.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			ParentScopeIdKey: {
				Description: "The parent scope ID that will be queried for the scope.",
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}

func dataSourceScopeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	opts := []scopes.Option{}

	var name string
	if v, ok := d.GetOk(NameKey); ok {
		name = v.(string)
	} else {
		return diag.Errorf("no name provided")
	}

	var parentScopeId string
	if parentScopeIdVal, ok := d.GetOk(ParentScopeIdKey); ok {
		parentScopeId = parentScopeIdVal.(string)
	} else {
		return diag.Errorf("no parent scope ID provided")
	}

	scp := scopes.NewClient(md.client)

	scpls, err := scp.List(ctx, parentScopeId, opts...)
	if err != nil {
		return diag.Errorf("error calling read scope: %v", err)
	}
	if scpls == nil {
		return diag.Errorf("no scopes found")
	}

	var scopeIdRead string
	for _, scopeItem := range scpls.GetItems() {
		if scopeItem.Name == name {
			scopeIdRead = scopeItem.Id
			break
		}
	}

	if scopeIdRead == "" {
		return diag.Errorf("scope name %v not found in scope list", err)
	}

	srr, err := scp.Read(ctx, scopeIdRead)
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read scope: %v", err)
	}
	if srr == nil {
		return diag.Errorf("scope nil after read")
	}

	if err := setFromScopeReadResponseMap(d, srr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// Adding this function since we are inctroducing parent_scope_id to align with the API. Once this is introduced to the
// resource_scope it would be optimal to refactor this so both resources use the same set function.
func setFromScopeReadResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}

	d.SetId(raw["id"].(string))
	return nil
}
