package provider

import (
	"context"

	"github.com/hashicorp/boundary/api/groups"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	groupMemberIdsKey = "member_ids"
)

func resourceGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGroupCreate,
		ReadContext:   resourceGroupRead,
		UpdateContext: resourceGroupUpdate,
		DeleteContext: resourceGroupDelete,
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
			groupMemberIdsKey: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []groups.Option{}

	var name *string
	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		name = &nameStr
		opts = append(opts, groups.WithName(nameStr))
	}

	var desc *string
	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		desc = &descStr
		opts = append(opts, groups.WithDescription(descStr))
	}

	grps := groups.NewClient(client)

	g, apiErr, err := grps.Create(
		ctx,
		scopeId,
		opts...)
	if err != nil {
		return diag.Errorf("error calling new group: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error creating group: %s", apiErr.Message)
	}

	if val, ok := d.GetOk(groupMemberIdsKey); ok {
		list := val.(*schema.Set).List()
		memberIds := make([]string, 0, len(list))
		for _, i := range list {
			memberIds = append(memberIds, i.(string))
		}
		g, apiErr, err = grps.SetMembers(
			ctx,
			g.Id,
			g.Version,
			memberIds)
		if apiErr != nil {
			return diag.Errorf("error setting principals on role: %s", apiErr.Message)
		}
		if err != nil {
			return diag.Errorf("error setting principals on role: %v", err)
		}

		if err := d.Set(groupMemberIdsKey, memberIds); err != nil {
			return diag.FromErr(err)
		}
	}

	if name != nil {
		if err := d.Set(NameKey, name); err != nil {
			return diag.FromErr(err)
		}
	}

	if desc != nil {
		if err := d.Set(DescriptionKey, *desc); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(g.Id)

	return nil
}

func resourceGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	grps := groups.NewClient(client)

	g, apiErr, err := grps.Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error reading group: %v", err)
	}
	if apiErr != nil {
		return diag.Errorf("error reading group: %s", apiErr.Message)
	}
	if g == nil {
		return diag.Errorf("group nil after read")
	}

	raw := g.LastResponseMap()
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
	d.Set(groupMemberIdsKey, raw["member_ids"])

	return nil
}

func resourceGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	grps := groups.NewClient(client)

	opts := []groups.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, groups.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, groups.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, groups.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, groups.WithDescription(descStr))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, groups.WithAutomaticVersioning(true))
		_, apiErr, err := grps.Update(
			ctx,
			d.Id(),
			0,
			opts...)
		if err != nil {
			return diag.Errorf("error updating group: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating group: %s", apiErr.Message)
		}
	}

	if d.HasChange(NameKey) {
		d.Set(NameKey, name)
	}
	if d.HasChange(DescriptionKey) {
		d.Set(DescriptionKey, desc)
	}

	if d.HasChange(groupMemberIdsKey) {
		var memberIds []string
		if membersVal, ok := d.GetOk(groupMemberIdsKey); ok {
			members := membersVal.(*schema.Set).List()
			for _, member := range members {
				memberIds = append(memberIds, member.(string))
			}
		}
		_, apiErr, err := grps.SetMembers(
			ctx,
			d.Id(),
			0,
			memberIds,
			groups.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating members on group: %v", err)
		}
		if apiErr != nil {
			return diag.Errorf("error updating members on group: %s", apiErr.Message)
		}
		d.Set(groupMemberIdsKey, memberIds)
	}

	return nil
}

func resourceGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := md.client

	grps := groups.NewClient(client)

	_, apiErr, err := grps.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting group: %s", err.Error())
	}
	if apiErr != nil {
		return diag.Errorf("error deleting group: %s", apiErr.Message)
	}

	return nil
}
