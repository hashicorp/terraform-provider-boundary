package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/hostcatalogs"
	"github.com/hashicorp/go-secure-stdlib/parseutil"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/crypto/blake2b"
)

const (
	hostCatalogTypePlugin = "plugin"
)

func resourceHostCatalogPlugin() *schema.Resource {
	return &schema.Resource{
		Description: "The host catalog resource allows you to configure a Boundary plugin-type host catalog. Host " +
			"catalogs are always part of a project, so a project resource should be used inline or you " +
			"should have the project ID in hand to successfully configure a host catalog.",

		CreateContext: resourceHostCatalogPluginCreate,
		ReadContext:   resourceHostCatalogPluginRead,
		UpdateContext: resourceHostCatalogPluginUpdate,
		DeleteContext: resourceHostCatalogPluginDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the host catalog.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The host catalog name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The host catalog description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID in which the resource is created.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			PluginIdKey: {
				Description:   "The ID of the plugin that should back the resource. This or " + PluginNameKey + " must be defined.",
				Type:          schema.TypeString,
				ConflictsWith: []string{PluginNameKey},
				Optional:      true,
				ForceNew:      true,
				Computed:      true, // If name is provided this will be computed
			},
			PluginNameKey: {
				Description:   "The name of the plugin that should back the resource. This or " + PluginIdKey + " must be defined.",
				Type:          schema.TypeString,
				ConflictsWith: []string{PluginIdKey},
				Optional:      true,
				ForceNew:      true,
				Computed:      true,
			},
			AttributesJsonKey: {
				Description: `The attributes for the host catalog. Either values encoded with the "jsonencode" function, pre-escaped JSON string, ` +
					`or a file:// or env:// path. Set to a string "null" or remove the block to clear all attributes in the host catalog.`,
				Type:     schema.TypeString,
				Optional: true,
				// If set to null in config and nothing comes from API, consider
				// it the same. Same if config changes from empty to null.
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
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
			SecretsJsonKey: {
				Description: `The secrets for the host catalog. Either values encoded with the "jsonencode" function, pre-escaped JSON string, ` +
					`or a file:// or env:// path. Set to a string "null" to clear any existing values. NOTE: Unlike "attributes_json", removing ` +
					`this block will NOT clear secrets from the host catalog; this allows injecting secrets for one call, then removing them for storage.`,
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			SecretsHmacKey: {
				Description: "The HMAC'd secrets value returned from the server.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			internalSecretsConfigHmacKey: {
				Description: "Internal only. HMAC of (serverSecretsHmac + config secrets). Used for proper secrets handling.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			internalHmacUsedForSecretsConfigHmacKey: {
				Description: "Internal only. The Boundary-provided HMAC used to calculate the current value of the HMAC'd config. Used for drift detection.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			internalForceUpdateKey: {
				Description: "Internal only. Used to force update so that we can always check the value of secrets.",
				Type:        schema.TypeString,
				Optional:    true,
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

// calculateCurrentConfigHmac generates an HMAC'd config value using the
// server's calculated HMAC as an HMAC key. Prior to calculating this we parse
// and then re-marshal the JSON to take advantage of Go alphabetizing JSON
// output so that we can ensure we'll treat the same objects as equivalent
// regardless of initial input order.
func calculateCurrentConfigHmac(serverHmac, secretsStr string) (string, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(secretsStr), &v); err != nil {
		return "", err
	}
	// Remarshal so we sanitize the order
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	key := blake2b.Sum256([]byte(serverHmac))
	mac := hmac.New(sha256.New, key[:])
	_, err = mac.Write(jsonBytes)
	if err != nil {
		return "", err
	}
	hmac := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(hmac), nil
}

// calculateConfigHmacPlan, given the schema and the current server HMAC,
// returns what to set on the server (if anything) and any diagnostics. If
// clearExisting is set we should nil out existing values in state. If
// sendToBoundary is set then the read secrets_json should be sent in an API
// call.
func calculateConfigHmacPlan(serverHmac, secretsJson string, d *schema.ResourceData) (clearExisting, sendToBoundary bool, diagWarn *diag.Diagnostic, retErr error) {
	existingConfigHmac := d.Get(internalSecretsConfigHmacKey).(string)

	// Iterate through possible states and handle appropriately
	switch {
	case serverHmac == "" && secretsJson == "":
		// State 1: No configured secrets in either Boundary or TF. Clear HMAC'd
		// config if present.
		return true, false, nil, nil

	case serverHmac == "" && secretsJson != "":
		// State 2: Boundary has no secrets but TF does. Put secrets into
		// Boundary. HMAC'd config will be updated after. However, if the value
		// is "null" it's a no-op so just make sure HMAC'd config is cleared if
		// present.
		switch secretsJson {
		case "null":
			return true, false, nil, nil

		default:
			return false, true, nil, nil
		}

	case serverHmac != "" && secretsJson == "":
		// State 3: Boundary has configured secrets but nothing in TF config. In
		// this case we make a break from "normal" TF behavior; rather than
		// interpret this as needing to clear secrets out of Boundary, we do
		// nothing. That way the user is free to configure secrets directly in
		// the API (thus never having it transport through TF or any of its
		// parts or configs) or remove them from the TF config file once they're
		// set. Note that we wipe knowledge of the HMAC'd config in this case,
		// so if they re-add values (even the same ones) we'll hit state 4. This
		// is the escape hatch for state 6a below.
		return true, false, nil, nil

	case serverHmac != "" && secretsJson != "" && existingConfigHmac == "":
		// State 4: In this case Boundary has secrets and TF config has secrets,
		// but we have no HMAC value at all in state. This almost certainly
		// means that Boundary was configured before the Terraform configuration
		// file was written, or at least prior to the value being added to TF's
		// config, since running TF with config would generate an HMAC'd config
		// value. We can assume that the TF config is new and meant as an update.
		return false, true, nil, nil

	case serverHmac != "" && secretsJson != "":
		// At this point we need to calculated the current HMAC'd config value
		// so we can see if it matches.
		currentConfigHmac, err := calculateCurrentConfigHmac(serverHmac, secretsJson)
		if err != nil {
			return false, false, nil, err
		}

		if existingConfigHmac == currentConfigHmac {
			// State 5: Both Boundary and TF have secrets configured and they
			// are known to match, so do nothing.
			return false, false, nil, nil
		}

		// The next bit is a little complicated. We have been storing not just
		// the HMAC'd config but also the boundary HMAC value that was used to
		// generate it. If that value has not changed, but the HMAC'd config
		// has, it means that the Terraform configuration has changed (but
		// Boundary has not been given new values since the last time we
		// calculated from the TF config) and we can assume that it is new. If
		// that value has changed, it is unclear whether or not Boundary's new
		// values are correct or the existing config values are correct. Picking
		// wrong could lead to a denial of service (e.g. if creds provided
		// through TF were rotated and no longer valid) so instead we raise this
		// as a warning to the user.
		//
		// The next question, however, is how does the user escape from this
		// scenario? Simply changing TF config more won't work as we'll be right
		// back here. The answer is state 3. If they remove the secrets we will
		// wipe knowlege of the HMAC'd config from TF; when they add them back
		// they'll hit state 4 and we'll put the new values in.
		switch {
		case serverHmac != d.Get(internalHmacUsedForSecretsConfigHmacKey).(string):
			// State 6a: mismatch. Warn the user.
			return false, false, &diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Mismatch in secrets state between Boundary and Terraform.",
				Detail: `Boundary's secret state is out of sync with Terraform. Usually this is the result of secrets being provided ` +
					`directly via Boundary's API. To suppress this warning, either remove the secrets via Boundary's API and allow Terraform to ` +
					`repopulate them, or remove the secrets_json block from Terraform's configuration file (the next time you add secrets_json ` +
					`back to the file, those values will be used to overwrite the current Boundary values.)`,
			}, nil

		default:
			// State 6b: Boundary's HMAC hasn't changed from the one we used to
			// generate the HMAC'd config, so we know the config has changed.
			return false, true, nil, nil
		}

	default:
		return false, false, nil, fmt.Errorf(
			"unhandled secrets state; server hmac is found: %t; secrets detected in config: %t; existing hmac'd config: %t",
			serverHmac != "", secretsJson != "", existingConfigHmac != "")
	}
}

func setFromHostCatalogPluginResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
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

func resourceHostCatalogPluginCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	opts := []hostcatalogs.Option{}

	var foundPluginId bool
	var foundPluginName bool
	if pluginIdVal, ok := d.GetOk(PluginIdKey); ok {
		pluginId := pluginIdVal.(string)
		opts = append(opts, hostcatalogs.WithPluginId(pluginId))
		foundPluginId = true
	}
	if pluginNameVal, ok := d.GetOk(PluginNameKey); ok {
		pluginName := pluginNameVal.(string)
		opts = append(opts, hostcatalogs.WithPluginName(pluginName))
		foundPluginName = true
	}
	if !foundPluginId && !foundPluginName {
		return diag.Errorf("neither plugin ID nor plugin name provided")
	}

	nameVal, ok := d.GetOk(NameKey)
	if ok {
		nameStr := nameVal.(string)
		opts = append(opts, hostcatalogs.WithName(nameStr))
	}

	descVal, ok := d.GetOk(DescriptionKey)
	if ok {
		descStr := descVal.(string)
		opts = append(opts, hostcatalogs.WithDescription(descStr))
	}

	attrsVal, ok := d.GetOk(AttributesJsonKey)
	if ok {
		attrsStr, err := parseutil.ParsePath(attrsVal.(string))
		if err != nil && !errors.Is(err, parseutil.ErrNotAUrl) {
			return diag.Errorf("error parsing path with attributes: %v", err)
		}
		switch attrsStr {
		case "null":
			opts = append(opts, hostcatalogs.DefaultAttributes())
		default:
			// What comes in is json-encoded but we want to set a
			// map[string]interface{} so we unmarshal it and set that
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(attrsStr), &m); err != nil {
				return diag.Errorf("error unmarshaling attributes: %v", err)
			}
			opts = append(opts, hostcatalogs.WithAttributes(m))
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
			opts = append(opts, hostcatalogs.DefaultSecrets())
		default:
			// What comes in is json-encoded but we want to set a
			// map[string]interface{} so we unmarshal it and set that
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(secretsJson), &m); err != nil {
				return diag.Errorf("error unmarshaling secrets: %v", err)
			}
			opts = append(opts, hostcatalogs.WithSecrets(m))
		}
	}

	hcClient := hostcatalogs.NewClient(md.client)

	hccr, err := hcClient.Create(ctx, hostCatalogTypePlugin, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating host catalog: %v", err)
	}
	if hccr == nil {
		return diag.Errorf("host catalog nil after create")
	}

	if err := setFromHostCatalogPluginResponseMap(d, hccr.GetResponse().Map); err != nil {
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

func resourceHostCatalogPluginRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hcClient := hostcatalogs.NewClient(md.client)

	hcrr, err := hcClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading host catalog: %v", err)
	}
	if hcrr == nil {
		return diag.Errorf("host catalog nil after read")
	}

	if err := setFromHostCatalogPluginResponseMap(d, hcrr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceHostCatalogPluginUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hcClient := hostcatalogs.NewClient(md.client)

	// We need to refresh the current server hmac value to figure out what to do
	// next around secrets handling
	var clearStateSecrets, sendSecretsToBoundary bool
	var secretsJson string
	var currentDiagnostics diag.Diagnostics
	{
		hcrr, err := hcClient.Read(ctx, d.Id())
		if err != nil {
			if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
				d.SetId("")
				return nil
			}
			return diag.Errorf("error reading host catalog in update: %v", err)
		}
		if hcrr == nil {
			return diag.Errorf("host catalog nil after read in update")
		}
		var serverSecretsHmac string
		if secretsHmacRaw, ok := hcrr.GetResponse().Map[SecretsHmacKey]; ok {
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

	opts := []hostcatalogs.Option{}

	if d.HasChange(NameKey) {
		opts = append(opts, hostcatalogs.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			opts = append(opts, hostcatalogs.WithName(nameStr))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, hostcatalogs.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			opts = append(opts, hostcatalogs.WithDescription(descStr))
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
				opts = append(opts, hostcatalogs.DefaultAttributes())
			default:
				// What comes in is json-encoded but we want to set a
				// map[string]interface{} so we unmarshal it and set that
				var m map[string]interface{}
				if err := json.Unmarshal([]byte(attrsStr), &m); err != nil {
					return append(currentDiagnostics, diag.Errorf("error unmarshaling attributes: %v", err)...)
				}
				opts = append(opts, hostcatalogs.WithAttributes(m))
			}
		} else {
			opts = append(opts, hostcatalogs.DefaultAttributes())
		}
	}

	if sendSecretsToBoundary {
		switch secretsJson {
		case "null":
			opts = append(opts, hostcatalogs.DefaultSecrets())
		default:
			// What comes in is json-encoded but we want to set a
			// map[string]interface{} so we unmarshal it and set that
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(secretsJson), &m); err != nil {
				return append(currentDiagnostics, diag.Errorf("error unmarshaling secrets: %v", err)...)
			}
			opts = append(opts, hostcatalogs.WithSecrets(m))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, hostcatalogs.WithAutomaticVersioning(true))
		hcur, err := hcClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return append(currentDiagnostics, diag.Errorf("error updating host catalog: %v", err)...)
		}
		if hcur == nil {
			return append(currentDiagnostics, diag.Errorf("host catalog nil after update")...)
		}

		if err := setFromHostCatalogPluginResponseMap(d, hcur.GetResponse().Map); err != nil {
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

func resourceHostCatalogPluginDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	hcClient := hostcatalogs.NewClient(md.client)

	_, err := hcClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting host catalog: %v", err)
	}

	return nil
}
