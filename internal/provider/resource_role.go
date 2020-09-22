package provider

import (
	"context"
	"net/http"

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
		CreateContext: resourceRoleCreate,
		ReadContext:   resourceRoleRead,
		UpdateContext: resourceRoleUpdate,
		DeleteContext: resourceRoleDelete,
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
			rolePrincipalIdsKey: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			roleGrantStringsKey: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
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

func setFromRoleResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(ScopeIdKey, raw["scope_id"])
	d.Set(rolePrincipalIdsKey, raw["principal_ids"])
	d.Set(roleGrantStringsKey, raw["grant_strings"])
	d.Set(roleGrantScopeIdKey, raw["grant_scope_id"])
	d.SetId(raw["id"].(string))
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

	tcr, apiErr, err := rc.Create(
		ctx,
		scopeId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling create role: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating role: %s", apiErr.Message)
	}
	if tcr == nil {
		return diag.Errorf("nil role after create")
	}
	raw := tcr.GetResponseMap()

	if principalIds != nil {
		tspr, apiErr, err := rc.SetPrincipals(
			ctx,
			tcr.Item.Id,
			0,
			principalIds,
			roles.WithAutomaticVersioning(true))
		if apiErr != nil {
			return diag.Errorf("error setting principal IDs on role: %s", apiErr.Message)
		}
		if err != nil {
			return diag.Errorf("error setting principal IDs on role: %v", err)
		}
		if tspr == nil {
			return diag.Errorf("nil role after setting principal IDs")
		}
		raw = tspr.GetResponseMap()
	}

	if grantStrings != nil {
		tsgr, apiErr, err := rc.SetGrants(
			ctx,
			tcr.Item.Id,
			0,
			grantStrings,
			roles.WithAutomaticVersioning(true))
		if apiErr != nil {
			return diag.Errorf("error setting grant strings on role: %s", apiErr.Message)
		}
		if err != nil {
			return diag.Errorf("error setting grant strings on role: %v", err)
		}
		if tsgr == nil {
			return diag.Errorf("nil role after setting grant strings")
		}
		raw = tsgr.GetResponseMap()
	}

	setFromRoleResponseMap(d, raw)

	return nil
}

func resourceRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	rc := roles.NewClient(md.client)

	trr, apiErr, err := rc.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling read role: %v", err)
	}
	if apiErr != nil {
		if apiErr.Status == int32(http.StatusNotFound) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading role: %s", apiErr.Message)
	}
	if trr == nil {
		return diag.Errorf("role nil after read")
	}

	setFromRoleResponseMap(d, trr.GetResponseMap())

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
		_, apiErr, err := rc.Update(
			ctx,
			d.Id(),
			0,
			opts...)
		if err != nil {
			return diag.Errorf("error calling update target: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating target: %s", apiErr.Message)
		}
	}

	if d.HasChange(NameKey) {
		d.Set(NameKey, name)
	}
	if d.HasChange(DescriptionKey) {
		d.Set(DescriptionKey, desc)
	}
	if d.HasChange(roleGrantScopeIdKey) {
		d.Set(roleGrantScopeIdKey, grantScopeId)
	}

	if d.HasChange(roleGrantStringsKey) {
		var grantStrings []string
		if grantStringsVal, ok := d.GetOk(roleGrantStringsKey); ok {
			grants := grantStringsVal.(*schema.Set).List()
			for _, grant := range grants {
				grantStrings = append(grantStrings, grant.(string))
			}
		}
		_, apiErr, err := rc.SetGrants(
			ctx,
			d.Id(),
			0,
			grantStrings,
			roles.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating grant strings on role: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating grant strings on role: %s", apiErr.Message)
		}
		d.Set(roleGrantStringsKey, grantStrings)
	}

	if d.HasChange(rolePrincipalIdsKey) {
		var principalIds []string
		if principalIdsVal, ok := d.GetOk(rolePrincipalIdsKey); ok {
			principals := principalIdsVal.(*schema.Set).List()
			for _, principal := range principals {
				principalIds = append(principalIds, principal.(string))
			}
		}
		_, apiErr, err := rc.SetPrincipals(
			ctx,
			d.Id(),
			0,
			principalIds,
			roles.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating grant strings on role: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating grant strings on role: %s", apiErr.Message)
		}
		d.Set(rolePrincipalIdsKey, principalIds)
	}

	return nil
}

func resourceRoleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.Get(roleDefaultRoleKey).(bool) {
		return nil
	}

	md := meta.(*metaData)
	rc := roles.NewClient(md.client)

	_, apiErr, err := rc.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling delete role: %s", err.Error())
	}
	if apiErr != nil {
		return diag.Errorf("error deleting role: %s", apiErr.Message)
	}

	return nil
}
