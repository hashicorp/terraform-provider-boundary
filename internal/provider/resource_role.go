package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/roles"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	roleGrantScopeIdKey = "grant_scope_id"
	rolePrincipalIdsKey = "principal_ids"
	roleGrantStringsKey = "grant_strings"
	roleDefaultRoleKey  = "default_role"
)

func resourceRole() *schema.Resource {
	return &schema.Resource{
		Description: "The role resource allows you to configure a Boundary role.",

		CreateContext: resourceRoleCreate,
		ReadContext:   resourceRoleRead,
		UpdateContext: resourceRoleUpdate,
		DeleteContext: resourceRoleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the role.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The role name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The role description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			rolePrincipalIdsKey: {
				Description: "A list of principal (user or group) IDs to add as principals on the role.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			roleGrantStringsKey: {
				Description: " A list of stringified grants for the role.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			roleGrantScopeIdKey: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			roleDefaultRoleKey: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Indicates that the role containing this value is the default role (that is, has the id 'r_default'), which triggers some specialized behavior to allow it to be imported and managed.",
			},
		},
	}
}

func setFromRoleResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw["name"]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw["description"]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw["scope_id"]); err != nil {
		return err
	}
	if err := d.Set(rolePrincipalIdsKey, raw["principal_ids"]); err != nil {
		return err
	}
	if err := d.Set(roleGrantStringsKey, raw["grant_strings"]); err != nil {
		return err
	}
	if err := d.Set(roleGrantScopeIdKey, raw["grant_scope_id"]); err != nil {
		return err
	}
	d.SetId(raw["id"].(string))
	return nil
}

func resourceRoleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.Get(roleDefaultRoleKey).(bool) {
		d.SetId("r_default")
		return resourceRoleRead(ctx, d, meta)
	}

	md := meta.(*metaData)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []roles.Option{}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, roles.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, roles.WithDescription(descStr))
	}

	grantScopeIdVal, ok := d.GetOk(roleGrantScopeIdKey)
	if ok {
		grantScopeIdStr := grantScopeIdVal.(string)
		opts = append(opts, roles.WithGrantScopeId(grantScopeIdStr))
	}

	var principalIds []string
	if principalIdsVal, ok := d.GetOk(rolePrincipalIdsKey); ok {
		list := principalIdsVal.(*schema.Set).List()
		principalIds = make([]string, 0, len(list))
		for _, i := range list {
			principalIds = append(principalIds, i.(string))
		}
	}

	var grantStrings []string
	if grantStringsVal, ok := d.GetOk(roleGrantStringsKey); ok {
		list := grantStringsVal.(*schema.Set).List()
		grantStrings = make([]string, 0, len(list))
		for _, i := range list {
			grantStrings = append(grantStrings, i.(string))
		}
	}

	rc := roles.NewClient(md.client)

	tcr, err := rc.Create(ctx, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error calling create role: %v", err)
	}
	if tcr == nil {
		return diag.Errorf("nil role after create")
	}
	raw := tcr.GetResponse().Map

	if principalIds != nil {
		tspr, err := rc.SetPrincipals(ctx, tcr.Item.Id, 0, principalIds, roles.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error setting principal IDs on role: %v", err)
		}
		if tspr == nil {
			return diag.Errorf("nil role after setting principal IDs")
		}
		raw = tspr.GetResponse().Map
	}

	if grantStrings != nil {
		tsgr, err := rc.SetGrants(ctx, tcr.Item.Id, 0, grantStrings, roles.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error setting grant strings on role: %v", err)
		}
		if tsgr == nil {
			return diag.Errorf("nil role after setting grant strings")
		}
		raw = tsgr.GetResponse().Map
	}

	if err := setFromRoleResponseMap(d, raw); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	rc := roles.NewClient(md.client)

	trr, err := rc.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read role: %v", err)
	}
	if trr == nil {
		return diag.Errorf("role nil after read")
	}

	if err := setFromRoleResponseMap(d, trr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceRoleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	rc := roles.NewClient(md.client)

	opts := []roles.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, roles.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, roles.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, roles.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, roles.WithDescription(descStr))
		}
	}

	var grantScopeId *string
	if d.HasChange(roleGrantScopeIdKey) {
		opts = append(opts, roles.DefaultGrantScopeId())
		grantScopeIdVal, ok := d.GetOk(roleGrantScopeIdKey)
		if ok {
			grantScopeIdStr := grantScopeIdVal.(string)
			grantScopeId = &grantScopeIdStr
			opts = append(opts, roles.WithGrantScopeId(grantScopeIdStr))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, roles.WithAutomaticVersioning(true))
		_, err := rc.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating target: %v", err)
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
	if d.HasChange(roleGrantScopeIdKey) {
		if err := d.Set(roleGrantScopeIdKey, grantScopeId); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(roleGrantStringsKey) {
		var grantStrings []string
		if grantStringsVal, ok := d.GetOk(roleGrantStringsKey); ok {
			grants := grantStringsVal.(*schema.Set).List()
			for _, grant := range grants {
				grantStrings = append(grantStrings, grant.(string))
			}
		}
		_, err := rc.SetGrants(ctx, d.Id(), 0, grantStrings, roles.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating grant strings on role: %v", err)
		}
		if err := d.Set(roleGrantStringsKey, grantStrings); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(rolePrincipalIdsKey) {
		var principalIds []string
		if principalIdsVal, ok := d.GetOk(rolePrincipalIdsKey); ok {
			principals := principalIdsVal.(*schema.Set).List()
			for _, principal := range principals {
				principalIds = append(principalIds, principal.(string))
			}
		}
		_, err := rc.SetPrincipals(ctx, d.Id(), 0, principalIds, roles.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating grant strings on role: %v", err)
		}
		if err := d.Set(rolePrincipalIdsKey, principalIds); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceRoleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.Get(roleDefaultRoleKey).(bool) {
		return nil
	}

	md := meta.(*metaData)
	rc := roles.NewClient(md.client)

	_, err := rc.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting role: %s", err.Error())
	}

	return nil
}
