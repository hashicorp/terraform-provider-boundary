package provider

/*
import (
	"context"
	"fmt"

	"github.com/hashicorp/boundary/api/roles"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	roleGrantScopeIdKey = "grant_scope_id"
	rolePrincipalIdsKey = "principal_ids"
	roleGrantStringsKey = "grant_strings"
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
		},
	}
}

func resourceRoleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []roles.Option{}

	var name *string
	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		name = &nameStr
		opts = append(opts, roles.WithName(nameStr))
	}

	var desc *string
	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		desc = &descStr
		opts = append(opts, roles.WithDescription(descStr))
	}

	var grantScopeId *string
	grantScopeIdVal, ok := d.GetOk(roleGrantScopeIdKey)
	if ok {
		grantScopeIdStr := grantScopeIdVal.(string)
		grantScopeId = &grantScopeIdStr
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

	t, apiErr, err := rc.Create(
		ctx,
		scopeId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling create role: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating role: %s", apiErr.Message)
	}

	if principalIds != nil {
		t, apiErr, err = rc.SetPrincipals(
			ctx,
			t.Id,
			t.Version,
			principalIds)
		if apiErr != nil {
			return diag.Errorf("error setting principal IDs on role: %s", apiErr.Message)
		}
		if err != nil {
			return diag.Errorf("error setting principal IDs on role: %v", err)
		}
		d.Set(rolePrincipalIdsKey, principalIds)
	}

	if grantStrings != nil {
		t, apiErr, err = rc.SetGrants(
			ctx,
			t.Id,
			t.Version,
			grantStrings)
		if apiErr != nil {
			return diag.Errorf("error setting grant strings on role: %s", apiErr.Message)
		}
		if err != nil {
			return diag.Errorf("error setting grant strings on role: %v", err)
		}
		d.Set(roleGrantStringsKey, grantStrings)
	}

	d.Set(NameKey, name)
	d.Set(DescriptionKey, desc)
	d.Set(ScopeIdKey, scopeId)
	d.Set(roleGrantScopeIdKey, grantScopeId)
	d.SetId(t.Id)

	return nil
}

func resourceRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	rc := roles.NewClient(md.client)

	t, apiErr, err := rc.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error calling read role: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading role: %s", apiErr.Message)
	}
	if t == nil {
		return diag.Errorf("role nil after read")
	}

	raw := t.LastResponseMap()
	if raw == nil {
		return []diag.Diagnostic{
			{
				Severity: diag.Warning,
				Summary:  "response map empty after read",
			},
		}
	}

	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(ScopeIdKey, raw["scope_id"])
	d.Set(roleGrantScopeIdKey, raw["grant_scope_id"])
	d.Set(rolePrincipalIdsKey, raw["principal_ids"])
	d.Set(roleGrantStringsKey, raw["grant_strings"])

	return nil
}

func resourceRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	r := convertResourceDataToRole(d)
	rolesClient := roles.NewClient(client)

	if d.HasChange(roleNameKey) {
		r.Name = d.Get(roleNameKey).(string)
	}

	if d.HasChange(roleDescriptionKey) {
		r.Description = d.Get(roleDescriptionKey).(string)
	}

	r, apiErr, err := rolesClient.Update(
		ctx,
		r.Id,
		0,
		roles.WithAutomaticVersioning(),
		roles.WithName(r.Name),
		roles.WithDescription(r.Description),
		roles.WithScopeId(r.Scope.Id))
	if apiErr != nil || err != nil {
		return fmt.Errorf("error updating role:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
	}

	if d.HasChange(roleGrantsKey) {
		grants := []string{}
		grantSet := d.Get(roleGrantsKey).(*schema.Set).List()

		for _, grant := range grantSet {
			grants = append(grants, grant.(string))
		}

		r, apiErr, err = rolesClient.SetGrants(
			ctx,
			r.Id,
			r.Version,
			grants,
			roles.WithScopeId(r.Scope.Id))
		if apiErr != nil || err != nil {
			return fmt.Errorf("error setting grants on role:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
		}
	}

	if d.HasChange(rolePrincipalsKey) {
		principalIds := []string{}
		principals := d.Get(rolePrincipalsKey).(*schema.Set).List()
		for _, principal := range principals {
			principalIds = append(principalIds, principal.(string))
		}

		r, apiErr, err = rolesClient.SetPrincipals(
			ctx,
			r.Id,
			r.Version,
			principalIds,
			roles.WithScopeId(r.Scope.Id))
		if apiErr != nil || err != nil {
			return fmt.Errorf("error updating principal on role:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
		}
	}

	return convertRoleToResourceData(r, d)
}

func resourceRoleDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	r := convertResourceDataToRole(d)
	rolesClient := roles.NewClient(client)

	_, apiErr, err := rolesClient.Delete(ctx, r.Id, roles.WithScopeId(r.Scope.Id))
	if apiErr != nil || err != nil {
		return fmt.Errorf("error deleting role:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
	}

	return nil
}
*/
