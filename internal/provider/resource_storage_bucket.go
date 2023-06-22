// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/storagebuckets"
	"github.com/hashicorp/go-secure-stdlib/parseutil"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	storageBucketNameKey   = "bucket_name"
	storageBucketPrefixKey = "bucket_prefix"
)

func resourceStorageBucket() *schema.Resource {
	return &schema.Resource{
		Description:   "The storage bucket resource allows you to configure a Boundary storage bucket. A storage bucket can only belong to the Global scope or an Org scope.",
		CreateContext: resourceStorageBucketCreate,
		ReadContext:   resourceStorageBucketRead,
		UpdateContext: resourceStorageBucketUpdate,
		DeleteContext: resourceStorageBucketDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the storage bucket.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The storage bucket name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The storage bucket description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			PluginIdKey: {
				Description:   "The ID of the plugin that should back the resource. This or " + PluginNameKey + " must be defined.",
				Type:          schema.TypeString,
				ConflictsWith: []string{PluginNameKey},
				ExactlyOneOf:  []string{PluginIdKey, PluginNameKey},
				Optional:      true,
				ForceNew:      true,
				Computed:      true, // If name is provided this will be computed
			},
			PluginNameKey: {
				Description:   "The name of the plugin that should back the resource. This or " + PluginIdKey + " must be defined.",
				Type:          schema.TypeString,
				ConflictsWith: []string{PluginIdKey},
				ExactlyOneOf:  []string{PluginIdKey, PluginNameKey},
				Optional:      true,
				ForceNew:      true,
			},
			ScopeIdKey: {
				Description: "The scope for this storage bucket.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			SecretsJsonKey: {
				Description: `The secrets for the storage bucket. Either values encoded with the "jsonencode" function, pre-escaped JSON string, ` +
					`or a file:// or env:// path. Set to a string "null" to clear any existing values. NOTE: Unlike "attributes_json", removing ` +
					`this block will NOT clear secrets from the storage bucket; this allows injecting secrets for one call, then removing them for storage.`,
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			SecretsHmacKey: {
				Description: "The HMAC'd secrets value returned from the server.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			storageBucketNameKey: {
				Description: "The name of the bucket within the external object store service.",
				Type:        schema.TypeString,
				Required:    true,
			},
			storageBucketPrefixKey: {
				Description: "The prefix used to organize the data held within the external object store.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			internalSecretsConfigHmacKey: {
				Description: "Internal only. HMAC of (serverSecretsHmac + config secrets). Used for proper secrets handling.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			internalHmacUsedForSecretsConfigHmacKey: {
				Description: "Internal only. The Boundary-provided HMAC used to calculate the current value of the HMAC'd config. Used for drift detection.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			AttributesJsonKey: {
				Description: `The attributes for the storage bucket. Either values encoded with the "jsonencode" function, pre-escaped JSON string, ` +
					`or a file:// or env:// path. Set to a string "null" or remove the block to clear all attributes in the storage bucket.`,
				Type:     schema.TypeString,
				Optional: true,
				// If set to null in config and nothing comes from API, consider
				// it the same. Same if config changes from empty to null.
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					sanitizedNew, err := sanitizeJson(new)
					if err != nil {
						return false
					}
					new = string(sanitizedNew)
					switch {
					case old == new:
						return true
					case old == "null" && new == "":
						return true
					case old == "" && new == "null":
						return true
					default:
						return false
					}
				},
			},
			WorkerFilterKey: {
				Description: "Filters to the worker(s) that can handle requests for this storage bucket.",
				Type:        schema.TypeString,
				Required:    true,
			},
			internalForceUpdateKey: {
				Description: "Internal only. Used to force update so that we can always check the value of secrets.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},

		// We want to always force an update (which itself may not actually do
		// anything) so that we can properly check secrets state.
		CustomizeDiff: func(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
			return d.SetNewComputed(internalForceUpdateKey)
		},
	}
}

func setFromStorageBucketResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw[NameKey]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw[DescriptionKey]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw[ScopeIdKey]); err != nil {
		return err
	}

	// Plugin stuff
	{
		if err := d.Set(PluginIdKey, raw[PluginIdKey]); err != nil {
			return err
		}
		pluginRaw, ok := raw["plugin"]
		if !ok {
			return fmt.Errorf("plugin field not found in response")
		}
		pluginInfo, ok := pluginRaw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("plugin field in response has wrong type")
		}
		pluginNameRaw, ok := pluginInfo["name"]
		if !ok {
			return fmt.Errorf("plugin name field not found in response")
		}
		pluginName, ok := pluginNameRaw.(string)
		if !ok {
			return fmt.Errorf("plugin name field in response has wrong type")
		}
		if err := d.Set(PluginNameKey, pluginName); err != nil {
			return err
		}
	}

	// Attributes stuff
	{
		attrRaw, ok := raw["attributes"]
		switch ok {
		case true:
			encodedAttributes, err := json.Marshal(attrRaw)
			if err != nil {
				return err
			}
			if err := d.Set(AttributesJsonKey, string(encodedAttributes)); err != nil {
				return err
			}
		default:
			d.Set(AttributesJsonKey, nil)
		}
	}

	// Secrets stuff
	{
		// We do not save secrets into the state file, and they're not returned in
		// the response
		secretsHmacRaw, ok := raw[SecretsHmacKey]
		switch ok {
		case true:
			if err := d.Set(SecretsHmacKey, secretsHmacRaw); err != nil {
				return err
			}
		default:
			d.Set(SecretsHmacKey, nil)
		}
	}

	d.SetId(raw[IDKey].(string))

	if err := d.Set(internalForceUpdateKey, strconv.FormatInt(rand.Int63(), 10)); err != nil {
		return err
	}

	return nil
}

func resourceStorageBucketCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	sbClient := storagebuckets.NewClient(md.client)
	opts := []storagebuckets.Option{}

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	if pluginIdVal, ok := d.GetOk(PluginIdKey); ok {
		opts = append(opts, storagebuckets.WithPluginId(pluginIdVal.(string)))
	}

	if pluginNameVal, ok := d.GetOk(PluginNameKey); ok {
		opts = append(opts, storagebuckets.WithPluginName(pluginNameVal.(string)))
	}

	if nameVal, ok := d.GetOk(NameKey); ok {
		opts = append(opts, storagebuckets.WithName(nameVal.(string)))
	}

	if descVal, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, storagebuckets.WithDescription(descVal.(string)))
	}

	if storageBucketNameVal, ok := d.GetOk(storageBucketNameKey); ok {
		opts = append(opts, storagebuckets.WithBucketName(storageBucketNameVal.(string)))
	}

	if storageBucketPrefixVal, ok := d.GetOk(storageBucketPrefixKey); ok {
		opts = append(opts, storagebuckets.WithBucketPrefix(storageBucketPrefixVal.(string)))
	}

	if workerFilterVal, ok := d.GetOk(WorkerFilterKey); ok {
		opts = append(opts, storagebuckets.WithWorkerFilter(workerFilterVal.(string)))
	}

	attrsVal, ok := d.GetOk(AttributesJsonKey)
	if ok {
		attrsStr, err := parseutil.ParsePath(attrsVal.(string))
		if err != nil && !errors.Is(err, parseutil.ErrNotAUrl) {
			return diag.Errorf("error parsing path with attributes: %v", err)
		}
		switch attrsStr {
		case "null":
			opts = append(opts, storagebuckets.DefaultAttributes())
		default:
			// What comes in is json-encoded but we want to set a
			// map[string]interface{} so we unmarshal it and set that
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(attrsStr), &m); err != nil {
				return diag.Errorf("error unmarshaling attributes: %v", err)
			}
			opts = append(opts, storagebuckets.WithAttributes(m))
		}
	}

	secretsVal, ok := d.GetOk(SecretsJsonKey)
	var secretsJson string
	if ok {
		var err error
		secretsJson, err = parseutil.ParsePath(secretsVal.(string))
		if err != nil && !errors.Is(err, parseutil.ErrNotAUrl) {
			return diag.Errorf("error parsing path with secrets: %v", err)
		}
		switch secretsJson {
		case "null":
			opts = append(opts, storagebuckets.DefaultSecrets())
		default:
			// What comes in is json-encoded but we want to set a
			// map[string]interface{} so we unmarshal it and set that
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(secretsJson), &m); err != nil {
				return diag.Errorf("error unmarshaling secrets: %v", err)
			}
			opts = append(opts, storagebuckets.WithSecrets(m))
		}
	}

	sbr, err := sbClient.Create(ctx, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating storage bucket: %v", err)
	}
	if sbr == nil {
		return diag.Errorf("nil storage bucket after create")
	}

	if err := setFromStorageBucketResponseMap(d, sbr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	if serverHmac := d.Get(SecretsHmacKey).(string); serverHmac != "" {
		configHmac, err := calculateCurrentConfigHmac(serverHmac, secretsJson)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(internalSecretsConfigHmacKey, configHmac); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(internalHmacUsedForSecretsConfigHmacKey, serverHmac); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceStorageBucketRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	sbClient := storagebuckets.NewClient(md.client)

	sbrr, err := sbClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading storage bucket: %v", err)
	}
	if sbrr == nil {
		return diag.Errorf("storage bucket nil after read")
	}

	if err := setFromStorageBucketResponseMap(d, sbrr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceStorageBucketUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	sbClient := storagebuckets.NewClient(md.client)

	opts := []storagebuckets.Option{}

	// We need to refresh the current server hmac value to figure out what to do
	// next around secrets handling
	var clearStateSecrets, sendSecretsToBoundary bool
	var secretsJson string
	var currentDiagnostics diag.Diagnostics
	{
		sbrr, err := sbClient.Read(ctx, d.Id())
		if err != nil {
			if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
				d.SetId("")
				return nil
			}
			return diag.Errorf("error reading storage bucket in update: %v", err)
		}
		if sbrr == nil {
			return diag.Errorf("storage bucket nil after read in update")
		}
		var serverSecretsHmac string
		if secretsHmacRaw, ok := sbrr.GetResponse().Map[SecretsHmacKey]; ok {
			serverSecretsHmac = secretsHmacRaw.(string)
		}
		// Get current secrets_json value
		secretsJson, err = parseutil.ParsePath(d.Get(SecretsJsonKey).(string))
		if err != nil && !errors.Is(err, parseutil.ErrNotAUrl) {
			return diag.Errorf("error parsing path with secrets: %v", err)
		}
		// Now that we have the value from the server, see if anything needs to be
		// done
		var diagWarning *diag.Diagnostic
		clearStateSecrets, sendSecretsToBoundary, diagWarning, err = calculateConfigHmacPlan(serverSecretsHmac, secretsJson, d)
		if err != nil {
			return diag.FromErr(err)
		}
		if diagWarning != nil {
			currentDiagnostics = append(currentDiagnostics, *diagWarning)
		}
	}

	if d.HasChange(NameKey) {
		opts = append(opts, storagebuckets.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			opts = append(opts, storagebuckets.WithName(nameStr))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, storagebuckets.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			opts = append(opts, storagebuckets.WithDescription(descStr))
		}
	}

	if d.HasChange(storageBucketNameKey) {
		opts = append(opts, storagebuckets.DefaultBucketName())
		storageBucketNameVal, ok := d.GetOk(storageBucketNameKey)
		if ok {
			storageBucketNameStr := storageBucketNameVal.(string)
			opts = append(opts, storagebuckets.WithBucketName(storageBucketNameStr))
		}
	}

	if d.HasChange(storageBucketPrefixKey) {
		opts = append(opts, storagebuckets.DefaultBucketPrefix())
		storageBucketPrefixVal, ok := d.GetOk(storageBucketPrefixKey)
		if ok {
			storageBucketPrefixStr := storageBucketPrefixVal.(string)
			opts = append(opts, storagebuckets.WithBucketPrefix(storageBucketPrefixStr))
		}
	}

	if d.HasChange(WorkerFilterKey) {
		opts = append(opts, storagebuckets.DefaultBucketName())
		workerFilterVal, ok := d.GetOk(WorkerFilterKey)
		if ok {
			workerFilterStr := workerFilterVal.(string)
			opts = append(opts, storagebuckets.WithWorkerFilter(workerFilterStr))
		}
	}

	if d.HasChange(AttributesJsonKey) {
		attrsVal, ok := d.GetOk(AttributesJsonKey)
		if ok {
			attrsStr, err := parseutil.ParsePath(attrsVal.(string))
			if err != nil && !errors.Is(err, parseutil.ErrNotAUrl) {
				return append(currentDiagnostics, diag.Errorf("error parsing path with attributes: %v", err)...)
			}
			switch attrsStr {
			case "null", "":
				opts = append(opts, storagebuckets.DefaultAttributes())
			default:
				// What comes in is json-encoded but we want to set a
				// map[string]interface{} so we unmarshal it and set that
				var m map[string]interface{}
				if err := json.Unmarshal([]byte(attrsStr), &m); err != nil {
					return append(currentDiagnostics, diag.Errorf("error unmarshaling attributes: %v", err)...)
				}
				opts = append(opts, storagebuckets.WithAttributes(m))
			}
		} else {
			opts = append(opts, storagebuckets.DefaultAttributes())
		}
	}

	if sendSecretsToBoundary {
		switch secretsJson {
		case "null":
			opts = append(opts, storagebuckets.DefaultSecrets())
		default:
			// What comes in is json-encoded but we want to set a
			// map[string]interface{} so we unmarshal it and set that
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(secretsJson), &m); err != nil {
				return append(currentDiagnostics, diag.Errorf("error unmarshaling secrets: %v", err)...)
			}
			opts = append(opts, storagebuckets.WithSecrets(m))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, storagebuckets.WithAutomaticVersioning(true))
		sbur, err := sbClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return append(currentDiagnostics, diag.Errorf("error updating storage bucket: %v", err)...)
		}
		if sbur == nil {
			return append(currentDiagnostics, diag.Errorf("storage bucket nil after update")...)
		}

		if err := setFromStorageBucketResponseMap(d, sbur.GetResponse().Map); err != nil {
			return append(currentDiagnostics, diag.FromErr(err)...)
		}
	}

	// Save any updated secrets information if needed
	switch {
	case clearStateSecrets:
		if err := d.Set(internalSecretsConfigHmacKey, nil); err != nil {
			return append(currentDiagnostics, diag.FromErr(err)...)
		}
		if err := d.Set(internalHmacUsedForSecretsConfigHmacKey, nil); err != nil {
			return append(currentDiagnostics, diag.FromErr(err)...)
		}

	case sendSecretsToBoundary:
		if serverHmac := d.Get(SecretsHmacKey).(string); serverHmac != "" {
			configHmac, err := calculateCurrentConfigHmac(serverHmac, secretsJson)
			if err != nil {
				return append(currentDiagnostics, diag.FromErr(err)...)
			}
			if err := d.Set(internalSecretsConfigHmacKey, configHmac); err != nil {
				return append(currentDiagnostics, diag.FromErr(err)...)
			}
			if err := d.Set(internalHmacUsedForSecretsConfigHmacKey, serverHmac); err != nil {
				return append(currentDiagnostics, diag.FromErr(err)...)
			}
		}
	}

	return nil
}

func resourceStorageBucketDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	sbClient := storagebuckets.NewClient(md.client)

	_, err := sbClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting storage bucket: %v", err)
	}

	return nil
}
