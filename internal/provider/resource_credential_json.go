// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentials"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	credentialJsonCredentialType = "json"
	credentialJsonObjectKey      = "object"
	credentialJsonObjectHmacKey  = "object_hmac"
)

func resourceCredentialJson() *schema.Resource {
	return &schema.Resource{
		Description: "The json credential resource allows you to congiure a credential using a json object.",

		CreateContext: resourceCredentialJsonCreate,
		ReadContext:   resourceCredentialJsonRead,
		UpdateContext: resourceCredentialJsonUpdate,
		DeleteContext: resourceCredentialJsonDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of this json credential.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The name of this json credential. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The description of this json credential.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreIdKey: {
				Description: "The credential store in which to save this json credential.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			credentialJsonObjectKey: {
				Description: `The object for the this json credential. Either values encoded with the "jsonencode" function, pre-escaped JSON string, or a file`,
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			credentialJsonObjectHmacKey: {
				Description: "The object hmac.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func setFromCredentialJsonResponseMap(d *schema.ResourceData, raw map[string]interface{}, fromRead bool) error {
	if err := d.Set(NameKey, raw[NameKey]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw[DescriptionKey]); err != nil {
		return err
	}
	if err := d.Set(credentialStoreIdKey, raw[credentialStoreIdKey]); err != nil {
		return err
	}

	if attrsVal, ok := raw["attributes"]; ok {
		attrs := attrsVal.(map[string]interface{})
		stateObjectHmac := d.Get(credentialJsonObjectHmacKey)
		boundaryObjectHmac := attrs[credentialJsonObjectHmacKey].(string)
		if stateObjectHmac.(string) != boundaryObjectHmac && fromRead {
			if err := d.Set(credentialJsonObjectKey, "(changed in Boundary)"); err != nil {
				return err
			}
		}
		if err := d.Set(credentialJsonObjectHmacKey, boundaryObjectHmac); err != nil {
			return nil
		}
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceCredentialJsonCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var opts []credentials.Option
	if v, ok := d.GetOk(NameKey); ok {
		opts = append(opts, credentials.WithName(v.(string)))
	}
	if v, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, credentials.WithDescription(v.(string)))
	}

	if v, ok := d.GetOk(credentialJsonObjectKey); ok {
		var jsonObject map[string]interface{}
		if err := json.Unmarshal([]byte(v.(string)), &jsonObject); err != nil {
			return diag.Errorf("error unmarshaling json: %v", err)
		}
		opts = append(opts, credentials.WithJsonCredentialObject(jsonObject))
	}

	var credentialStoreId string
	retrievedStoreId, ok := d.GetOk(credentialStoreIdKey)
	if ok {
		credentialStoreId = retrievedStoreId.(string)
	} else {
		return diag.Errorf("credential store id is unset")
	}

	client := credentials.NewClient(md.client)
	cred, err := client.Create(ctx, credentialJsonCredentialType, credentialStoreId, opts...)
	if err != nil {
		return diag.Errorf("error creating credential: %v", err)
	}
	if cred == nil {
		return diag.Errorf("nil credential after create")
	}

	if err := setFromCredentialJsonResponseMap(d, cred.GetResponse().Map, false); err != nil {
		return diag.Errorf("error generating credential from response map: %v", err)
	}

	return nil
}

func resourceCredentialJsonRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentials.NewClient(md.client)

	cred, err := client.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading credential: %v", err)
	}
	if cred == nil {
		return diag.Errorf("credential nil after read")
	}

	if err := setFromCredentialJsonResponseMap(d, cred.GetResponse().Map, true); err != nil {
		return diag.Errorf("error generating credential from response map: %v", err)
	}

	return nil
}

func resourceCredentialJsonUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentials.NewClient(md.client)

	var opts []credentials.Option
	if d.HasChange(NameKey) {
		opts = append(opts, credentials.DefaultName())
		if v, ok := d.GetOk(NameKey); ok {
			opts = append(opts, credentials.WithName(v.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, credentials.DefaultDescription())
		if v, ok := d.GetOk(DescriptionKey); ok {
			opts = append(opts, credentials.WithDescription(v.(string)))
		}
	}

	if d.HasChange(credentialJsonObjectKey) {
		if v, ok := d.GetOk(credentialJsonObjectKey); ok {
			var jsonObject map[string]interface{}
			if err := json.Unmarshal([]byte(v.(string)), &jsonObject); err != nil {
				return diag.Errorf("error unmarshaling json: %v", err)
			}
			opts = append(opts, credentials.WithJsonCredentialObject(jsonObject))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, credentials.WithAutomaticVersioning(true))
		credUpdate, err := client.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating credential: %v", err)
		}
		if credUpdate == nil {
			return diag.Errorf("credential nil after update")
		}

		if err = setFromCredentialJsonResponseMap(d, credUpdate.GetResponse().Map, false); err != nil {
			return diag.Errorf("error generating credential from response map: %v", err)
		}
	}

	return nil
}

func resourceCredentialJsonDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentials.NewClient(md.client)

	_, err := client.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting credential: %v", err)
	}

	return nil
}
