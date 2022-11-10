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
	version                            = "version"
	address                            = "address"
	canonicalTags                      = "canonical_tags"
	configTags                         = "config_tags"
	workerGeneratedAuthToken           = "worker_generated_auth_token"
	controllerGeneratedActivationToken = "controller_generated_activation_token"
	apiTags                            = "api_tags"
	releaseVersion                     = "release_version"
	authorizedActions                  = "authorized_actions"
)

func resourceWorker() *schema.Resource {
	return &schema.Resource{
		Description: "The resource allows you to create a self-managed worker object.",

		CreateContext: resourceWorkerCreate,
		ReadContext:   resourceWorkerRead,
		UpdateContext: resourceWorkerUpdate,
		DeleteContext: resourceWorkerDelete,
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
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The description for the worker.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			address: {
				Description: "The accessible address of the self managed worker.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			canonicalTags: {
				Description: "The aggregated view of worker tags and API tags.",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				Computed: true,
			},
			configTags: {
<<<<<<< HEAD:internal/provider/worker.go
				Description: "Tags as configured in the worker's HCL file.",
=======
				Description: "",
>>>>>>> ef6f54e859b702c5a853f3f4e90737190ddf3dcc:internal/provider/resource_self_managed_worker.go
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				Computed: true,
			},
			workerGeneratedAuthToken: {
				Description: "The worker authentication token required to register the worker for the worker-led authentication flow. Leaving this blank will result in a controller generated token.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			controllerGeneratedActivationToken: {
				Description: "A single use token generated by the controller to be passed to the self-managed worker.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			apiTags: {
				Description: "API tags applied to the worker.",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				Optional: true,
			},
			releaseVersion: {
				Description: "The version of the Boundary binary running on the self managed worker.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			authorizedActions: {
				Description: "A list of actions that the worker is entitled to perform.",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
		},
	}
}

func setFromWorkerResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	d.SetId(raw["id"].(string))
	d.Set(ScopeIdKey, raw["scope_id"])
	d.Set(NameKey, raw["name"])
	d.Set(DescriptionKey, raw["description"])
	d.Set(address, raw["address"])
	d.Set(canonicalTags, raw["canonical_tags"])
	d.Set(configTags, raw["config_tags"])
	d.Set(workerGeneratedAuthToken, raw["worker_generated_auth_token"])
	d.Set(controllerGeneratedActivationToken, raw["controller_generated_activation_token"])
	d.Set(releaseVersion, raw["release_version"])
	d.Set(authorizedActions, raw["authorized_actions"])
	d.Set(apiTags, raw["api_tags"])

	return nil
}

func resourceWorkerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	if err := setFromWorkerResponseMap(d, wrr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceWorkerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		if err := setFromWorkerResponseMap(d, wkrc.GetResponse().Map); err != nil {
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
		if err := setFromWorkerResponseMap(d, wkrc.GetResponse().Map); err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

func resourceWorkerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	return nil
}

func resourceWorkerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	wClient := workers.NewClient(md.client)

	_, err := wClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting worker: %v", err)
	}

	return nil
}
