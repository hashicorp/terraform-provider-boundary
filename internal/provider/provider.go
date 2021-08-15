package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/boundary/sdk/wrapper"
	wrapping "github.com/hashicorp/go-kms-wrapping"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func init() {
	// descriptions are written in markdown for docs
	schema.DescriptionKind = schema.StringMarkdown
}

func New() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"addr": {
				Type:        schema.TypeString,
				Required:    true,
				Description: `The base url of the Boundary API, e.g. "http://127.0.0.1:9200". If not set, it will be read from the "BOUNDARY_ADDR" env var.`,
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: `The Boundary token to use, as a string or path on disk containing just the string. If set, the token read here will be used in place of authenticating with the auth method specified in "auth_method_id", although the recovery KMS mechanism will still override this. Can also be set with the BOUNDARY_TOKEN environment variable.`,
			},
			"recovery_kms_hcl": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Can be a heredoc string or a path on disk. If set, the string/file will be parsed as HCL and used with the recovery KMS mechanism. While this is set, it will override any other authentication information; the KMS mechanism will always be used. See Boundary's KMS docs for examples: https://boundaryproject.io/docs/configuration/kms",
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
			"boundary_account":                  resourceAccount(),
			"boundary_account_password":         resourceAccountPassword(),
			"boundary_account_oidc":             resourceAccountOidc(),
			"boundary_auth_method":              resourceAuthMethod(),
			"boundary_auth_method_password":     resourceAuthMethodPassword(),
			"boundary_auth_method_oidc":         resourceAuthMethodOidc(),
			"boundary_credential_library_vault": resourceCredentialLibraryVault(),
			"boundary_credential_store_vault":   resourceCredentialStoreVault(),
			"boundary_managed_group":            resourceManagedGroup(),
			"boundary_group":                    resourceGroup(),
			"boundary_host":                     resourceHost(),
			"boundary_host_catalog":             resourceHostCatalog(),
			"boundary_host_set":                 resourceHostset(),
			"boundary_role":                     resourceRole(),
			"boundary_scope":                    resourceScope(),
			"boundary_target":                   resourceTarget(),
			"boundary_user":                     resourceUser(),
		},
	}

	p.ConfigureContextFunc = providerConfigure(p)

	return p
}

type metaData struct {
	client             *api.Client
	recoveryKmsWrapper wrapping.Wrapper
}

func providerAuthenticate(ctx context.Context, d *schema.ResourceData, md *metaData) error {
	var credentials map[string]interface{}

	authMethodId, authMethodIdOk := d.GetOk("auth_method_id")
	recoveryKmsHcl, recoveryKmsHclOk := d.GetOk("recovery_kms_hcl")
	if token, ok := d.GetOk("token"); ok {
		md.client.SetToken(token.(string))
	}

	switch {
	case recoveryKmsHclOk:
		recoveryHclStr, _, err := ReadPathOrContents(recoveryKmsHcl.(string))
		if err != nil {
			return fmt.Errorf(`error reading data from "recovery_kms_hcl": %v`, err)
		}
		wrapper, err := wrapper.GetWrapperFromHcl(recoveryHclStr, "recovery")
		if err != nil {
			return fmt.Errorf(`error reading wrappers from "recovery_kms_hcl": %v`, err)
		}
		if wrapper == nil {
			return errors.New(`No "kms" block with purpose "recovery" found in "recovery_kms_hcl"`)
		}
		if err := wrapper.Init(ctx); err != nil {
			return fmt.Errorf("error initializing recovery kms: %v", err)
		}

		md.recoveryKmsWrapper = wrapper
		md.client.SetRecoveryKmsWrapper(wrapper)
		return nil

	case md.client.Token() != "":
		// Use the token sourced from the conf file or env var

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

		am := authmethods.NewClient(md.client)

		at, err := am.Authenticate(ctx, authMethodId.(string), "login", credentials)
		if err != nil {
			return err
		}
		md.client.SetToken(at.Attributes["token"].(string))

	default:
		return errors.New("no suitable auth method information found")
	}

	return nil
}

func providerConfigure(p *schema.Provider) schema.ConfigureContextFunc {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		client, err := api.NewClient(nil)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		if url, ok := d.GetOk("addr"); ok {
			if err := client.SetAddr(url.(string)); err != nil {
				return nil, diag.FromErr(err)
			}
		}
		if client.Addr() == "" {
			return nil, diag.Errorf(`"no valid address could be determined from "addr" or "BOUNDARY_ADDR" env var`)
		}

		client.SetLimiter(5, 5)

		md := &metaData{
			client: client,
		}

		if err := providerAuthenticate(ctx, d, md); err != nil {
			return nil, diag.FromErr(err)
		}

		return md, nil
	}
}
