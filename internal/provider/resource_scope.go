package provider

import (
	"context"

	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	scopeGlobalScopeKey = "global_scope"
	scopeAutoCreateRole = "auto_create_role"
)

func resourceScope() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScopeCreate,
		ReadContext:   resourceScopeRead,
		UpdateContext: resourceScopeUpdate,
		DeleteContext: resourceScopeDelete,

		Schema: map[string]*schema.Schema{
			NameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			DescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			ScopeIdKey: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			scopeGlobalScopeKey: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Indicates that the scope containing this value is the global scope, which triggers some specialized behavior to allow it to be imported and managed.",
			},
			scopeAutoCreateRole: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If set, when a new scope is created, the provider will not disable the functionality that automatically creates a role in the new scope and gives permissions to manage the scope to the provider's user. Marking this true makes for simpler HCL but results in role resources that are unmanaged by Terraform.",
			},
		},
	}
}

func setFromScopeResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	if d.Id() == "global" {
		d.Set(ScopeIdKey, "global")
	} else {
		d.Set(ScopeIdKey, raw["scope_id"])
	}
	d.SetId(raw["id"].(string))
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
	if !d.Get(scopeAutoCreateRole).(bool) {
		opts = append(opts, scopes.WithSkipRoleCreation(true))
	}

	scp := scopes.NewClient(md.client)

	scr, apiErr, err := scp.Create(
		ctx,
		scopeId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling create scope: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating scope: %s", apiErr.Message)
	}
	if scr == nil {
		return diag.Errorf("scope nil after create")
	}

	setFromScopeResponseMap(d, scr.GetResponseMap())

	return nil
}

func resourceScopeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)

	srr, apiErr, err := scp.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling read scope: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading scope: %s", apiErr.Message)
	}
	if srr == nil {
		return diag.Errorf("scope nil after read")
	}

	setFromScopeResponseMap(d, srr.GetResponseMap())

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
		_, apiErr, err := scp.Update(
			ctx,
			d.Id(),
			0,
			opts...)
		if err != nil {
			return diag.Errorf("error calling update scope: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating scope: %s", apiErr.Message)
		}
	}

	if d.HasChange(NameKey) {
		d.Set(NameKey, name)
	}
	if d.HasChange(DescriptionKey) {
		d.Set(DescriptionKey, desc)
	}

	return nil
}

func resourceScopeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.Get(scopeGlobalScopeKey).(bool) {
		return nil
	}

	md := meta.(*metaData)
	scp := scopes.NewClient(md.client)

	_, apiErr, err := scp.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete scope: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error deleting scope: %s", apiErr.Message)
	}

	return nil
}
