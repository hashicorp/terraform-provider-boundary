package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/boundary/api/authtokens"
	"github.com/hashicorp/boundary/sdk/wrapper"
	wrapping "github.com/hashicorp/go-kms-wrapping"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func New() terraform.ResourceProvider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"base_url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: `The base url of the Boundary API, e.g. "http://127.0.0.1"`,
			},
			"recovery_kms_hcl": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "If set, will be parsed as HCL and used with the recovery KMS mechanism. While this is set, it will override other authentication information; the KMS mechanism will always be used.",
			},
			"auth_method_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The auth method ID e.g. ampw_1234567890",
			},
			"password_auth_method_login_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The auth method login name for password-style auth methods",
			},
			"password_auth_method_password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The auth method password for password-style auth methods",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"boundary_scope": resourceScope(),
			"boundary_user":  resourceUser(),
			/*
				"boundary_group": resourceGroup(),
					"boundary_host":         resourceHost(),
					"boundary_host_catalog": resourceHostCatalog(),
					"boundary_host_set":     resourceHostset(),
					"boundary_role":         resourceRole(),
					"boundary_target":       resourceTarget(),
			*/
		},
	}

	p.ConfigureFunc = providerConfigure(p)

	return p
}

type metaData struct {
	client             *api.Client
	authToken          *authtokens.AuthToken
	recoveryKmsWrapper wrapping.Wrapper
	ctx                context.Context
}

func providerAuthenticate(d *schema.ResourceData, md *metaData) error {
	var credentials map[string]interface{}

	authMethodId, authMethodIdOk := d.GetOk("auth_method_id")
	recoveryKmsHcl, recoveryKmsHclOk := d.GetOk("recovery_kms_hcl")

	switch {
	case recoveryKmsHclOk:
		wrapper, err := wrapper.GetWrapperFromHcl(recoveryKmsHcl.(string), "recovery")
		if err != nil {
			return fmt.Errorf(`error reading wrappers from "recovery_kms_hcl": %w`, err)
		}
		if wrapper == nil {
			return errors.New(`No "kms" block with purpose "recovery" found in "recovery_kms_hcl"`)
		}
		if err := wrapper.Init(md.ctx); err != nil {
			return fmt.Errorf("error initializing recovery kms: %w", err)
		}

		md.recoveryKmsWrapper = wrapper
		md.client.SetRecoveryKmsWrapper(wrapper)
		return nil

	case authMethodIdOk:
		switch {
		case strings.HasPrefix(authMethodId.(string), "ampw"):
			// Password-style
			authMethodLoginName, ok := d.GetOk("password_auth_method_login_name")
			if !ok {
				return errors.New("password-style auth method login name not set, please set password_auth_method_login_name on the provider")
			}
			authMethodPassword, ok := d.GetOk("password_auth_method_password")
			if !ok {
				return errors.New("password-style auth method password not set, please set password_auth_method_password on the provider")
			}
			credentials = map[string]interface{}{
				"login_name": authMethodLoginName,
				"password":   authMethodPassword,
			}

		default:
			return errors.New("no suitable typed auth method information found")
		}

	default:
		return errors.New("no suitable auth method information found")
	}

	am := authmethods.NewClient(md.client)

	at, apiErr, err := am.Authenticate(md.ctx, authMethodId.(string), credentials)
	if apiErr != nil {
		return errors.New(apiErr.Message)
	}
	if err != nil {
		return err
	}
	md.client.SetToken(at.Token)

	md.authToken = at
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

		client.SetLimiter(5, 5)

		md := &metaData{
			client: client,
			ctx:    p.StopContext(),
		}

		if err := providerAuthenticate(d, md); err != nil {
			return nil, err
		}

		return md, nil
	}
}
