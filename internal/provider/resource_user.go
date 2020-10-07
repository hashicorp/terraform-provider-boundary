package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/users"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const userAccountIDsKey = "account_ids"

func resourceUser() *schema.Resource {
	return &schema.Resource{
		Description: "The user resource allows you to configure a Boundary user.",

		CreateContext: resourceUserCreate,
		ReadContext:   resourceUserRead,
		UpdateContext: resourceUserUpdate,
		DeleteContext: resourceUserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the user.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The username. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The user description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID in which the resource is created. Defaults to the provider's `default_scope` if unset.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			userAccountIDsKey: {
				Description: "Account ID's to associate with this user resource.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func setFromUserResponseMap(d *schema.ResourceData, raw map[string]interface{}) {
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(ScopeIdKey, raw["scope_id"])
	d.Set(userAccountIDsKey, raw["account_ids"])
	d.SetId(raw["id"].(string))
}

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []users.Option{}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, users.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, users.WithDescription(descStr))
	}

	usrs := users.NewClient(md.client)

	ucr, err := usrs.Create(ctx, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating user: %v", err)
	}
	if ucr == nil {
		return diag.Errorf("user nil after create")
	}
	raw := ucr.GetResponseMap()

	if val, ok := d.GetOk(userAccountIDsKey); ok {
		list := val.(*schema.Set).List()
		acctIds := make([]string, 0, len(list))
		for _, i := range list {
			acctIds = append(acctIds, i.(string))
		}
		usrac, err := usrs.SetAccounts(ctx, ucr.Item.Id, ucr.Item.Version, acctIds)
		if err != nil {
			return diag.Errorf("error setting accounts on user: %v", err)
		}
		if usrac == nil {
			return diag.Errorf("user nil after setting accounts")
		}
		raw = usrac.GetResponseMap()
	}

	setFromUserResponseMap(d, raw)

	return nil
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	usrs := users.NewClient(md.client)

	urr, err := usrs.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Status == int32(http.StatusNotFound) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read user: %v", err)
	}
	if urr == nil {
		return diag.Errorf("user nil after read")
	}

	setFromUserResponseMap(d, urr.GetResponseMap())

	return nil
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	usrs := users.NewClient(md.client)

	opts := []users.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, users.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, users.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, users.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, users.WithDescription(descStr))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, users.WithAutomaticVersioning(true))
		_, err := usrs.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating user: %v", err)
		}
	}

	if d.HasChange(NameKey) {
		d.Set(NameKey, name)
	}
	if d.HasChange(DescriptionKey) {
		d.Set(DescriptionKey, desc)
	}

	if d.HasChange(userAccountIDsKey) {
		var accountIds []string
		if accountsVal, ok := d.GetOk(userAccountIDsKey); ok {
			accounts := accountsVal.(*schema.Set).List()
			for _, account := range accounts {
				accountIds = append(accountIds, account.(string))
			}

		}
		_, err := usrs.SetAccounts(ctx, d.Id(), 0, accountIds, users.WithAutomaticVersioning(true))
		if err != nil {
			return diag.Errorf("error updating accounts on user: %v", err)
		}
		d.Set(userAccountIDsKey, accountIds)
	}

	return nil
}

func resourceUserDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	usrs := users.NewClient(md.client)

	_, err := usrs.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting user: %v", err)
	}

	return nil
}
