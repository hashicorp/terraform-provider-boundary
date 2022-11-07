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
	scopeIdKey      = "scope_id"
	authMethodIdKey = "auth_method_id"
)

func resourceScopePrimaryAuthMethod() *schema.Resource {
	return &schema.Resource{
		Description: "Setting the scope's primary auth method.",

		CreateContext: resourceScopePrimaryAuthMethodCreate,
		ReadContext:   resourceScopePrimaryAuthMethodRead,
		UpdateContext: resourceScopePrimaryAuthMethodUpdate,
		DeleteContext: resourceScopePrimaryAuthMethodDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			scopeIdKey: {
				Description: "The ID of the scope.",
				Type:        schema.TypeString,
				Required:    true,
			},
			authMethodIdKey: {
				Description: "The ID of the auth method.",
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}

func setFromScopePrimaryAuthMethodResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(authMethodIdKey, raw["primary_auth_method_id"]); err != nil {
		return err
	}
	d.SetId(raw["id"].(string))
	return nil
}

func resourceScopePrimaryAuthMethodCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	scopeId := d.Get(scopeIdKey).(string)
	authMethodId := d.Get(authMethodIdKey).(string)

	opts := []scopes.Option{
		scopes.WithAutomaticVersioning(true),
		scopes.WithPrimaryAuthMethodId(authMethodId),
	}
	scpClient := scopes.NewClient(md.client)
	apiResponse, err := scpClient.Update(ctx, scopeId, 0, opts...)
	if err != nil {
		return diag.Errorf("error setting scope primary auth method: %v", err)
	}
	if apiResponse == nil {
		return diag.Errorf("scope nil after updating primary auth method")
	}

	if err := setFromScopePrimaryAuthMethodResponseMap(d, apiResponse.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceScopePrimaryAuthMethodRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scpClient := scopes.NewClient(md.client)

	apiResponse, err := scpClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read scope: %v", err)
	}
	if apiResponse == nil {
		return diag.Errorf("scope nil after read")
	}

	if err := setFromScopePrimaryAuthMethodResponseMap(d, apiResponse.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceScopePrimaryAuthMethodUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scpClient := scopes.NewClient(md.client)

	if d.HasChange(scopeIdKey) {
		oldScopeId, _ := d.GetChange(scopeIdKey)
		if oldScopeId != nil {
			_, err := scpClient.Update(ctx, oldScopeId.(string), 0,
				scopes.WithAutomaticVersioning(true),
				scopes.DefaultPrimaryAuthMethodId())
			if err != nil {
				if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() != http.StatusNotFound {
					return diag.Errorf("error removing primary auth method from old scope %v", err)
				}
			}
		}
	}

	if d.HasChange(authMethodIdKey) {
		scopeId := d.Get(scopeIdKey).(string)
		authMethodId := d.Get(authMethodIdKey).(string)
		apiResponse, err := scpClient.Update(ctx, scopeId, 0,
			scopes.WithAutomaticVersioning(true),
			scopes.WithPrimaryAuthMethodId(authMethodId))
		if err != nil {
			return diag.Errorf("error setting primary auth method on scope %v", err)
		}
		if err = setFromScopePrimaryAuthMethodResponseMap(d, apiResponse.GetResponse().Map); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceScopePrimaryAuthMethodDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scpClient := scopes.NewClient(md.client)

	_, err := scpClient.Update(ctx, d.Id(), 0,
		scopes.WithAutomaticVersioning(true),
		scopes.DefaultPrimaryAuthMethodId())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			return nil
		}
		return diag.Errorf("error removing primary auth method from scope: %v", err)
	}

	return nil
}
