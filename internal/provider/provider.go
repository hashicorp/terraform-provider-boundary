package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/hashicorp/watchtower/api"
)

func New() terraform.ResourceProvider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"default_organization": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("WATCHTOWER_DEFAULT_ORG", ""),
				Description: "The Watchtower organization scope to operate all actions in if not provided in the individual resources.",
			},
			"base_url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The base url of the Watchtower API.  For example 'http://127.0.0.1/'",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"watchtower_group":        resourceGroup(),
			"watchtower_host_catalog": resourceHostCatalog(),
			"watchtower_project":      resourceProject(),
			"watchtower_role":         resourceRole(),
			"watchtower_user":         resourceUser(),
		},
	}

	p.ConfigureFunc = providerConfigure(p)

	return p
}

type metaData struct {
	client *api.Client
	ctx    context.Context
}

func providerConfigure(p *schema.Provider) schema.ConfigureFunc {
	return func(d *schema.ResourceData) (interface{}, error) {
		client, err := api.NewClient(nil)
		if err != nil {
			return nil, err
		}
		if err := client.SetAddr(d.Get("base_url").(string)); err != nil {
			return nil, err
		}
		client.SetOrg(d.Get("default_organization").(string))

		// TODO: Pass these in through the config, add token, etc...
		client.SetLimiter(5, 5)

		return &metaData{client: client, ctx: p.StopContext()}, nil
	}
}
