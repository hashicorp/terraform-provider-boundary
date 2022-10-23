package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/workers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	scope                              = "scope"
	scopeId                            = "global"
	createdTime                        = "created_time"
	updatedTime                        = "updated_time"
	version                            = "version"
	address                            = "address"
	canonicalTags                      = "canonical_tags"
	configTags                         = "config_tags"
	lastStatusTime                     = "last_status_time"
	workerGeneratedAuthToken           = "worker_generated_auth_token"
	controllerGeneratedActivationToken = "controller_generated_activation_token"
	apiTags                            = "api_tags"
	releaseVersion                     = "release_version"
	authorizedActions                  = "authorized_actions"
)

func resourceSelfManagedWorker() *schema.Resource {
	return &schema.Resource{
		Description: "The resource allows you to create a self-managed worker object.",

		CreateContext: resourceSelfManagedWorkerCreate,
		ReadContext:   resourceSelfManagedWorkerRead,
		UpdateContext: resourceSelfManagedWorkerUpdate,
		DeleteContext: resourceSelfManagedWorkerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the worker.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			ScopeIdKey: {
				Description: "The scope for the worker.",
				Type:        schema.TypeString,
				Required:    true,
			},
			NameKey: {
				Description: "The name for the worker.",
				Type:        schema.TypeString,
				Computed:    false,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The description for the worker.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			createdTime: {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			updatedTime: {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			version: {
				Description: "",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			address: {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			canonicalTags: {
				Description: "",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
				},
				Computed: true,
			},
			configTags: {
				Description: "",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
				},
				Computed: true,
			},
			workerGeneratedAuthToken: {
				Description: "",
				Type:        schema.TypeString,
				Optional:    true,
			},
			controllerGeneratedActivationToken: {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			TypeKey: {
				Description: "",
				Type:        schema.TypeString,
				Computed:    true,
			},
			apiTags: {
				Description: "",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
				},
				Computed: true,
			},
			releaseVersion: {
				Description: "",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			authorizedActions: {
				Description: "",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
		},
	}
}

func setFromSelfManagedWorkerResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	d.SetId(raw["id"].(string))
	d.Set(ScopeIdKey, raw["scope_id"])
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(createdTime, raw["created_time"])
	d.Set(updatedTime, raw["updated_time"])
	d.Set(version, raw["version"])
	d.Set(address, raw["address"])
	d.Set(canonicalTags, raw["canonical_tags"])
	d.Set(configTags, raw["config_tags"])
	d.Set(workerGeneratedAuthToken, raw["worker_generated_auth_token"])
	d.Set(controllerGeneratedActivationToken, raw["controller_generated_activation_token"])
	d.Set(TypeKey, raw["type"])
	d.Set(apiTags, raw["api_tags"])
	d.Set(releaseVersion, raw["release_version"])
	d.Set(authorizedActions, raw["authorized_actions"])

	return nil
}

func resourceSelfManagedWorkerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	wkrs := workers.NewClient(md.client)

	wrr, err := wkrs.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error calling read worker: %v", err)
	}
	if wrr == nil {
		return diag.Errorf("worker nil after read")
	}

	if err := setFromSelfManagedWorkerResponseMap(d, wrr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceSelfManagedWorkerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	opts := []workers.Option{}

	if v, ok := d.GetOk(NameKey); ok {
		opts = append(opts, workers.WithName(v.(string)))
	}

	if v, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, workers.WithDescription(v.(string)))
	}

	var workerAuthToken string
	if v, ok := d.GetOk(workerGeneratedAuthToken); ok {
		workerAuthToken = v.(string)
	}

	wkr := workers.NewClient(md.client)

	if len(workerAuthToken) > 0 {
		wkrc, err := wkr.CreateWorkerLed(ctx, workerAuthToken, scopeId, opts...)
		if err != nil {
			return diag.Errorf("error creating worker: %v", err)
		}
		if wkrc == nil {
			return diag.Errorf("worker nil after create")
		}
		if err := setFromSelfManagedWorkerResponseMap(d, wkrc.GetResponse().Map); err != nil {
			return diag.FromErr(err)
		}
	} else {
		wkrc, err := wkr.CreateControllerLed(ctx, scopeId, opts...)
		if err != nil {
			return diag.Errorf("error creating worker: %v", err)
		}
		if wkrc == nil {
			return diag.Errorf("worker nil after create")
		}
		if err := setFromSelfManagedWorkerResponseMap(d, wkrc.GetResponse().Map); err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

func resourceSelfManagedWorkerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	wkr := workers.NewClient(md.client)

	opts := []workers.Option{}

	var name *string
	if d.HasChange(NameKey) {
		opts = append(opts, workers.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			nameStr := nameVal.(string)
			name = &nameStr
			opts = append(opts, workers.WithName(nameStr))
		}
	}

	var desc *string
	if d.HasChange(DescriptionKey) {
		opts = append(opts, workers.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			descStr := descVal.(string)
			desc = &descStr
			opts = append(opts, workers.WithDescription(descStr))
		}
	}

	var versionInt int
	if versionVal, ok := d.GetOk(version); ok {
		versionInt = versionVal.(int)
	}

	if len(opts) > 0 {
		opts = append(opts, workers.WithAutomaticVersioning(true))
		_, err := wkr.Update(ctx, d.Id(), uint32(versionInt), opts...)
		if err != nil {
			return diag.Errorf("error updating worker: %v", err)
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

	// if d.HasChange(userAccountIDsKey) {
	// 	var accountIds []string
	// 	if accountsVal, ok := d.GetOk(userAccountIDsKey); ok {
	// 		accounts := accountsVal.(*schema.Set).List()
	// 		for _, account := range accounts {
	// 			accountIds = append(accountIds, account.(string))
	// 		}

	// 	}
	// 	_, err := wkr.s(ctx, d.Id(), 0, accountIds, workers.WithAutomaticVersioning(true))
	// 	if err != nil {
	// 		return diag.Errorf("error updating accounts on user: %v", err)
	// 	}
	// 	if err := d.Set(userAccountIDsKey, accountIds); err != nil {
	// 		return diag.FromErr(err)
	// 	}
	// }

	return nil
}

// func resourceSelfManagedWorkerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
// 	md := meta.(*metaData)

// 	var id string
// 	if idVal, ok := d.GetOk(IDKey); ok {
// 		id = idVal.(string)
// 	} else {
// 		return diag.Errorf("no scope ID provided")
// 	}

// 	var versionInt int
// 	if versionVal, ok := d.GetOk(version); ok {
// 		versionInt = versionVal.(int)
// 	}

// 	opts := []workers.Option{}

// 	if v, ok := d.GetOk(NameKey); ok {
// 		opts = append(opts, workers.WithName(v.(string)))
// 	}

// 	if v, ok := d.GetOk(DescriptionKey); ok {
// 		opts = append(opts, workers.WithDescription(v.(string)))
// 	}

// 	wkrs := workers.NewClient(md.client)

// 	wup, err := wkrs.Update(ctx, id, uint32(versionInt), opts...)
// 	if err != nil {
// 		return diag.Errorf("error updating worker: %v", err)
// 	}
// 	if wup == nil {
// 		return diag.Errorf("worker nil after create")
// 	}
// 	if err := setFromSelfManagedWorkerResponseMap(d, wup.GetResponse().Map); err != nil {
// 		return diag.FromErr(err)
// 	}

// 	return nil
// }

func resourceSelfManagedWorkerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	wClient := workers.NewClient(md.client)

	_, err := wClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting worker: %v", err)
	}

	return nil
}
