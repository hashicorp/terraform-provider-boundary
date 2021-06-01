package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentiallibraries"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const credentialStoreIdKey = "credential_store_id"

func resourceCredentialLibrary() *schema.Resource {
	return &schema.Resource{
		Description: "The credential library resource allows you to configure a Boundary credential library.",

		CreateContext: resourceCredentialLibraryCreate,
		ReadContext:   resourceCredentialLibraryRead,
		UpdateContext: resourceCredentialLibraryUpdate,
		DeleteContext: resourceCredentialLibraryDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the credential library.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The credential library name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The credential library description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID for the credential library.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			TypeKey: {
				Description: "The resource type.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			//TODO (malnick) Does this need to be ForceNew?
			credentialStoreIdKey: {
				Description: "The ID of the credential store that this library belongs to.",
				Type:        schema.TypeString,
				Optional:    true,
			},
		},
	}
}

func setFromCredentialLibraryResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw[NameKey]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw[DescriptionKey]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw[ScopeIdKey]); err != nil {
		return err
	}
	if err := d.Set(TypeKey, raw[TypeKey]); err != nil {
		return err
	}
	if err := d.Set(credentialStoreIdKey, raw[credentialStoreIdKey]); err != nil {
		return err
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceCredentialLibraryCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	opts := []credentiallibraries.Option{}

	if v, ok := d.GetOk(NameKey); ok {
		opts = append(opts, credentiallibraries.WithName(v.(string)))
	}

	if v, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, credentiallibraries.WithDescription(v.(string)))
	}

	if v, ok := d.GetOk(ScopeIdKey); ok {
		opts = append(opts, credentiallibraries.WithScope(v.(string)))
	}

	var credentialstoreid string
	cid, ok := d.GetOk(credentialStoreIdKey)
	if ok {
		credentialstoreid = cid.(string)
	} else {
		return diag.Errorf("no credential store ID is set")
	}

	client := credentiallibraries.NewClient(md.client)

	cr, err := client.Create(ctx, credentialstoreid, opts...)
	if err != nil {
		return diag.Errorf("error creating credential library: %v", err)
	}
	if cr == nil {
		return diag.Errorf("nil credential library after create")
	}

	if err := setFromCredentialLibraryResponseMap(d, cr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceCredentialLibraryRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentiallibraries.NewClient(md.client)

	cr, err := client.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading credential library: %v", err)
	}
	if cr == nil {
		return diag.Errorf("credential library nil after read")
	}

	if err := setFromCredentialLibraryResponseMap(d, cr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceCredentialLibraryUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentiallibraries.NewClient(md.client)

	opts := []credentiallibraries.Option{}

	if d.HasChange(NameKey) {
		opts = append(opts, credentiallibraries.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			opts = append(opts, credentiallibraries.WithName(nameVal.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, credentiallibraries.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			opts = append(opts, credentiallibraries.WithDescription(descVal.(string)))
		}
	}

	// TODO (malnick) If credential store ID does not force new, add update logic here...

	if len(opts) > 0 {
		opts = append(opts, credentiallibraries.WithAutomaticVersioning(true))
		aur, err := client.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating credential library: %v", err)
		}

		setFromCredentialLibraryResponseMap(d, aur.GetResponse().Map)
	}

	return nil
}

func resourceCredentialLibraryDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentiallibraries.NewClient(md.client)

	_, err := client.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting credential library: %v", err)
	}

	return nil
}
