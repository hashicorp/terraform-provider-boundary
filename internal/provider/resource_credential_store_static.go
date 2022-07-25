package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentialstores"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const staticCredentialStoreType = "static"

func resourceStaticCredentialStore() *schema.Resource {
	return &schema.Resource{
		Description: "The static credential store resource allows you to configure a Boundary static credential store using Vault.",

		CreateContext: resourceStaticCredentialStoreCreate,
		ReadContext:   resourceStaticCredentialStoreRead,
		UpdateContext: resourceStaticCredentialStoreUpdate,
		DeleteContext: resourceStaticCredentialStoreDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the static credential store.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The static credential store name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The static credential store description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope for this credential store.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
		},
	}
}

func setFromStaticCredentialStoreResponseMap(d *schema.ResourceData, raw map[string]interface{}, fromRead bool) error {
	if err := d.Set(NameKey, raw[NameKey]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw[DescriptionKey]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw[ScopeIdKey]); err != nil {
		return err
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceStaticCredentialStoreCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var opts []credentialstores.Option
	if v, ok := d.GetOk(NameKey); ok {
		opts = append(opts, credentialstores.WithName(v.(string)))
	}
	if v, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, credentialstores.WithDescription(v.(string)))
	}

	var scope string
	gotScope, ok := d.GetOk(ScopeIdKey)
	if ok {
		scope = gotScope.(string)
	} else {
		return diag.Errorf("no scope is set")
	}

	client := credentialstores.NewClient(md.client)
	cr, err := client.Create(ctx, staticCredentialStoreType, scope, opts...)
	if err != nil {
		return diag.Errorf("error creating credential store: %v", err)
	}
	if cr == nil {
		return diag.Errorf("nil credential store after create")
	}

	if err := setFromStaticCredentialStoreResponseMap(d, cr.GetResponse().Map, false); err != nil {
		return diag.Errorf("error generating credential store from response map: %v", err)
	}

	return nil
}

func resourceStaticCredentialStoreRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentialstores.NewClient(md.client)

	cr, err := client.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading credential store: %v", err)
	}
	if cr == nil {
		return diag.Errorf("credential store nil after read")
	}

	if err := setFromStaticCredentialStoreResponseMap(d, cr.GetResponse().Map, true); err != nil {
		return diag.Errorf("error generating credential store from response map: %v", err)
	}

	return nil
}

func resourceStaticCredentialStoreUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentialstores.NewClient(md.client)

	var opts []credentialstores.Option
	if d.HasChange(NameKey) {
		opts = append(opts, credentialstores.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			opts = append(opts, credentialstores.WithName(nameVal.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, credentialstores.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			opts = append(opts, credentialstores.WithDescription(descVal.(string)))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, credentialstores.WithAutomaticVersioning(true))
		crUpdate, err := client.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating credential store: %v", err)
		}
		if crUpdate == nil {
			return diag.Errorf("credential store nil after update")
		}

		if err = setFromStaticCredentialStoreResponseMap(d, crUpdate.GetResponse().Map, false); err != nil {
			return diag.Errorf("error generating credential store from response map: %v", err)
		}
	}

	return nil
}

func resourceStaticCredentialStoreDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentialstores.NewClient(md.client)

	_, err := client.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting credential store: %v", err)
	}

	return nil
}
