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
	scopeGlobalScopeKey        = "global_scope"
	scopeAutoCreateAdminRole   = "auto_create_admin_role"
	scopeAutoCreateDefaultRole = "auto_create_default_role"
)

func resourceScope() *schema.Resource {
	return &schema.Resource{
		Description: "The scope resource allows you to configure a Boundary scope.",

		CreateContext: resourceScopeCreate,
		ReadContext:   resourceScopeRead,
		UpdateContext: resourceScopeUpdate,
		DeleteContext: resourceScopeDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the scope.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The scope name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The scope description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID containing the sub scope resource.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			scopeGlobalScopeKey: {
				Description: "Indicates that the scope containing this value is the global scope, which triggers some specialized behavior to allow it to be imported and managed.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			scopeAutoCreateAdminRole: {
				Description: "If set, when a new scope is created, the provider will not disable the functionality that automatically creates a role in the new scope and gives permissions to manage the scope to the provider's user. Marking this true makes for simpler HCL but results in role resources that are unmanaged by Terraform.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			scopeAutoCreateDefaultRole: {
				Description: "Only relevant when creating an org scope. If set, when a new scope is created, the provider will not disable the functionality that automatically creates a role in the new scope and gives listing of scopes and auth methods and the ability to authenticate to the anonymous user. Marking this true makes for simpler HCL but results in role resources that are unmanaged by Terraform.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
		},
	}
}

func setFromScopeResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}
	if d.Id() == "global" {
		if err := d.Set(ScopeIdKey, "global"); err != nil {
			return err
		}
	} else {
		if err := d.Set(ScopeIdKey, raw["scope_id"]); err != nil {
			return err
		}
	}

	d.SetId(raw["id"].(string))
	return nil
}

func resourceScopeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.Get(scopeGlobalScopeKey).(bool) {
		d.SetId("global")
		return resourceScopeRead(ctx, d, meta)
	}

	md := meta.(*metaData)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []scopes.Option{}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, scopes.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, scopes.WithDescription(descStr))
	}

	// Always skip unless overridden, because if you're using TF to manage this
	// creates a resource outside of TF's control. So the normal TF paradigm
	// would be to create a role in the current scope giving permissions in the
	// new scope, once you have the new scope ID, and TF can figure out the
	// ordering.
	//
	// TODO: (?) Put authentication information, if available, into a data
	// source from the current token, so that the user can be introspected when
	// defining these roles instead of having to be explicitly defined in
	// config.
	if !d.Get(scopeAutoCreateAdminRole).(bool) {
		opts = append(opts, scopes.WithSkipAdminRoleCreation(true))
	}
	if !d.Get(scopeAutoCreateDefaultRole).(bool) {
		opts = append(opts, scopes.WithSkipDefaultRoleCreation(true))
	}

	scp := scopes.NewClient(md.client)

	scr, err := scp.Create(ctx, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating scope: %v", err)
	}
	if scr == nil {
		return diag.Errorf("scope nil after create")
	}

	if err := setFromScopeResponseMap(d, scr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceScopeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)

	srr, err := scp.Read(ctx, d.Id())
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

	if err := setFromScopeResponseMap(d, srr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceScopeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)

	opts := []scopes.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, scopes.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, scopes.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, scopes.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, scopes.WithDescription(descStr))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, scopes.WithAutomaticVersioning(true))
		_, err := scp.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating scope: %v", err)
		}
	}

	if d.HasChange(NameKey) {
		if err := d.Set(NameKey, name); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange(DescriptionKey) {
		if err := d.Set(DescriptionKey, desc); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceScopeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.Get(scopeGlobalScopeKey).(bool) {
		return nil
	}

	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)

	_, err := scp.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting scope: %v", err)
	}

	return nil
}
