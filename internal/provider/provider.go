package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/hashicorp/watchtower/api"
	"github.com/hashicorp/watchtower/api/scopes"
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
			"auth_method_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The auth method ID. Example am_1234567890",
			},
			"auth_method_username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The auth method username",
			},
			"auth_method_password": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The auth method password",
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

func providerAuthenticate(d *schema.ResourceData, client *api.Client) error {
	authMethodID, ok := d.GetOk("auth_method_id")
	if !ok {
		return errors.New("auth method ID not set, please set auth_method_id on the provider")
	}

	authMethodUser, ok := d.GetOk("auth_method_username")
	if !ok {
		return errors.New("auth method username not set, please set auth_method_username on the provider")
	}

	authMethodPass, ok := d.GetOk("auth_method_password")
	if !ok {
		return errors.New("auth method password not set, please set the auth_method_password on the provider")
	}

	org := &scopes.Org{
		Client: client,
	}
	ctx := context.Background()

	// note: Authenticate() calls SetToken() under the hood to set the
	// auth bearer on the client so we do not need to do anything with the
	// returned token after this call, so we ignore it
	_, apiErr, err := org.Authenticate(ctx, authMethodID.(string), authMethodUser.(string), authMethodPass.(string))
	if apiErr != nil {
		return errors.New(apiErr.Message)
	}
	if err != nil {
		return err
	}

	return nil
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

		if err := providerAuthenticate(d, client); err != nil {
			return nil, err
		}

		return &metaData{client: client, ctx: p.StopContext()}, nil
	}
}
