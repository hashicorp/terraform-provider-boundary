// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/boundary/sdk/wrapper"
	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
	"github.com/hashicorp/go-secure-stdlib/configutil/v2"
	"github.com/hashicorp/go-secure-stdlib/pluginutil/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	kms_plugin_assets "github.com/hashicorp/terraform-provider-boundary/plugins/kms"
)

const (
	PASSWORD_AUTH_METHOD_PREFIX = "ampw"
	LDAP_AUTH_METHOD_PREFIX     = "amldap"
	DEFAULT_PROVIDER_SCOPE      = "global"
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
				Description: "The auth method ID e.g. ampw_1234567890. If not set, the default auth method for the given scope ID will be used.",
			},
			"password_auth_method_login_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The auth method login name for password-style auth methods",
				Deprecated:  "Use auth_method_login_name instead",
			},
			"password_auth_method_password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The auth method password for password-style auth methods",
				Deprecated:  "Use auth_method_password instead",
			},
			"auth_method_login_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The auth method login name for password-style or ldap-style auth methods",
			},
			"auth_method_password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The auth method password for password-style or ldap-style auth methods",
			},
			"tls_insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "When set to true, does not validate the Boundary API endpoint certificate",
			},
			"plugin_execution_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: `Specifies a directory that the Boundary provider can use to write and execute its built-in plugins.`,
			},
			"scope_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: `The scope ID for the default auth method.`,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"boundary_account":                                  resourceAccount(),
			"boundary_account_password":                         resourceAccountPassword(),
			"boundary_account_oidc":                             resourceAccountOidc(),
			"boundary_account_ldap":                             resourceAccountLdap(),
			"boundary_alias_target":                             resourceAliasTarget(),
			"boundary_auth_method":                              resourceAuthMethod(),
			"boundary_auth_method_password":                     resourceAuthMethodPassword(),
			"boundary_auth_method_oidc":                         resourceAuthMethodOidc(),
			"boundary_auth_method_ldap":                         resourceAuthMethodLdap(),
			"boundary_credential_library_vault":                 resourceCredentialLibraryVault(),
			"boundary_credential_library_vault_ssh_certificate": resourceCredentialLibraryVaultSshCertificate(),
			"boundary_credential_store_vault":                   resourceCredentialStoreVault(),
			"boundary_credential_store_static":                  resourceCredentialStoreStatic(),
			"boundary_credential_username_password":             resourceCredentialUsernamePassword(),
			"boundary_credential_username_password_domain":      resourceCredentialUsernamePasswordDomain(),
			"boundary_credential_ssh_private_key":               resourceCredentialSshPrivateKey(),
			"boundary_credential_json":                          resourceCredentialJson(),
			"boundary_managed_group":                            resourceManagedGroup(),
			"boundary_managed_group_ldap":                       resourceManagedGroupLdap(),
			"boundary_group":                                    resourceGroup(),
			"boundary_host":                                     resourceHost(),
			"boundary_host_static":                              resourceHostStatic(),
			"boundary_host_catalog":                             resourceHostCatalog(),
			"boundary_host_catalog_static":                      resourceHostCatalogStatic(),
			"boundary_host_catalog_plugin":                      resourceHostCatalogPlugin(),
			"boundary_host_set":                                 resourceHostSet(),
			"boundary_host_set_static":                          resourceHostSetStatic(),
			"boundary_host_set_plugin":                          resourceHostSetPlugin(),
			"boundary_policy_storage":                           resourcePolicyStorage(),
			"boundary_scope_policy_attachment":                  resourceScopePolicyAttachment(),
			"boundary_role":                                     resourceRole(),
			"boundary_scope":                                    resourceScope(),
			"boundary_storage_bucket":                           resourceStorageBucket(),
			"boundary_target":                                   resourceTarget(),
			"boundary_user":                                     resourceUser(),
			"boundary_worker":                                   resourceWorker(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"boundary_account":     dataSourceAccount(),
			"boundary_auth_method": dataSourceAuthMethod(),
			"boundary_group":       dataSourceGroup(),
			"boundary_scope":       dataSourceScope(),
			"boundary_user":        dataSourceUser(),
			"boundary_role":        dataSourceRole(),
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
	amClient := authmethods.NewClient(md.client)

	var providerScope string
	scopeId, scopeIdOk := d.GetOk("scope_id")
	switch {
	case scopeIdOk:
		providerScope = scopeId.(string)
	default:
		providerScope = DEFAULT_PROVIDER_SCOPE
	}

	recoveryKmsHcl, recoveryKmsHclOk := d.GetOk("recovery_kms_hcl")
	if token, ok := d.GetOk("token"); ok {
		md.client.SetToken(token.(string))
	}

	// If auth_method_id is not set, get the default auth method ID for the given scope ID.
	// Skip fetching default auth method when recovery_kms_hcl is set or the client token isn't null.
	authMethodId, authMethodIdOk := d.GetOk("auth_method_id")
	if !authMethodIdOk && !recoveryKmsHclOk && md.client.Token() == "" {
		defaultAuthMethodId, err := getDefaultAuthMethodId(ctx, amClient, providerScope, "")
		if err != nil {
			return err
		}
		authMethodIdOk = true
		authMethodId = defaultAuthMethodId
	}

	switch {
	case recoveryKmsHclOk:
		recoveryHclStr, _, err := ReadPathOrContents(recoveryKmsHcl.(string))
		if err != nil {
			return fmt.Errorf(`error reading data from "recovery_kms_hcl": %v`, err)
		}

		opts := []pluginutil.Option{
			pluginutil.WithPluginsMap(kms_plugin_assets.BuiltinKmsPlugins()),
			pluginutil.WithPluginsFilesystem(kms_plugin_assets.KmsPluginPrefix, kms_plugin_assets.FileSystem()),
		}

		if execDir, ok := d.GetOk("plugin_execution_dir"); ok {
			opts = append(opts, pluginutil.WithPluginExecutionDirectory(execDir.(string)))
		}

		// TODO: cleanup plugin when finished
		wrapper, _, err := wrapper.GetWrapperFromHcl(
			ctx,
			recoveryHclStr,
			"recovery",
			configutil.WithPluginOptions(opts...))
		if err != nil {
			return fmt.Errorf(`error reading wrappers from "recovery_kms_hcl": %v`, err)
		}
		if wrapper == nil {
			return errors.New(`No "kms" block with purpose "recovery" found in "recovery_kms_hcl"`)
		}

		md.recoveryKmsWrapper = wrapper
		md.client.SetRecoveryKmsWrapper(wrapper)
		return nil

	case md.client.Token() != "":
		// Use the token sourced from the conf file or env var

	case authMethodIdOk:
		switch {
		case strings.HasPrefix(authMethodId.(string), PASSWORD_AUTH_METHOD_PREFIX) || strings.HasPrefix(authMethodId.(string), LDAP_AUTH_METHOD_PREFIX):
			// Password-style & LDAP-style
			authMethodLoginName, ok := d.GetOk("auth_method_login_name")
			if !ok {
				authMethodLoginName, ok = d.GetOk("password_auth_method_login_name")
				if !ok {
					return errors.New("auth method login name not set, please set auth_method_login_name on the provider")
				}
			}
			authMethodPassword, ok := d.GetOk("auth_method_password")
			if !ok {
				authMethodPassword, ok = d.GetOk("password_auth_method_password")
				if !ok {
					return errors.New("auth method password not set, please set auth_method_password on the provider")
				}
			}
			credentials = map[string]interface{}{
				"login_name": authMethodLoginName,
				"password":   authMethodPassword,
			}
		case strings.HasPrefix(authMethodId.(string), "amoidc"):
			// OIDC-style
			return errors.New("OIDC auth method is currently not supported by Boundary Terraform Provider. only password auth method is supported at this time")
		default:
			return errors.New("no suitable typed auth method information found")
		}

		at, err := amClient.Authenticate(ctx, authMethodId.(string), "login", credentials)
		if err != nil {
			if apiErr := api.AsServerError(err); apiErr != nil {
				statusCode := apiErr.Response().StatusCode()
				if statusCode == http.StatusNotFound {
					return fmt.Errorf("unknown auth_method_id: %s", err.Error())
				}
				if statusCode == http.StatusUnauthorized {
					return fmt.Errorf("invalid login name or password: %s", err.Error())
				}
			}
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

		if tlsInsecure, ok := d.GetOk("tls_insecure"); ok {
			if client.SetTLSConfig(&api.TLSConfig{Insecure: tlsInsecure.(bool)}) != nil {
				return nil, diag.Errorf("could not set insecure tls")
			}
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

// getDefaultAuthMethodId iterates over boundary client.List() to find the default auth method ID for the given scopeId.
// If there is only one auth method, it'll return it even if it's not the primary auth method
// If scope ID is empty or no primary auth method is found, it returns an error.
func getDefaultAuthMethodId(ctx context.Context, client *authmethods.Client, scopeId, amType string) (string, error) {
	if scopeId == "" {
		return "", fmt.Errorf("must pass a non empty scope ID string to get default auth method ID")
	}
	authMethodListResult, err := client.List(ctx, scopeId)
	if err != nil {
		return "", err
	}

	authMethodItems := authMethodListResult.GetItems()

	// If there is only one auth method that matches the auth method prefix, return it even if it's not the primary auth method
	if len(authMethodItems) == 1 {
		authMethod := authMethodItems[0]
		if !strings.HasPrefix(authMethod.Id, amType) {
			return "", fmt.Errorf("error looking up default auth method for scope ID: '%s'. got '%s' but the provider requires an auth method prefix of '%s'", scopeId, authMethod.Id, amType)
		}

		return authMethod.Id, nil
	}

	// find the primary auth method that matches auth method prefix
	for _, m := range authMethodItems {
		if m.IsPrimary {
			if !strings.HasPrefix(m.Id, amType) {
				return "", fmt.Errorf("error looking up primary auth method for scope ID: '%s'. got '%s' but the provider requires an auth method prefix of '%s'", scopeId, m.Id, amType)
			}

			return m.Id, nil
		}
	}
	return "", fmt.Errorf("default auth method not found for scope ID: '%s'. Please set a primary auth method on this scope or pass one explicitly using the `auth_method_id` field", scopeId)
}
