package provider

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

const (
	authmethodTypeOidc                                 = "oidc"
	authmethodOidcIssuerKey                            = "issuer"
	authmethodOidcClientIdKey                          = "client_id"
	authmethodOidcClientSecretKey                      = "client_secret"
	authmethodOidcMaxAgeKey                            = "max_age"
	authmethodOidcApiUrlPrefixKey                      = "api_url_prefix"
	authmethodOidcCaCertificatesKey                    = "ca_certificates"
	authmethodOidcAllowedAudiencesKey                  = "allowed_audiences"
	authmethodOidcDisableDiscoveredConfigValidationKey = "disable_discovered_config_validation"
	authmethodOidcSigningAlgorithmsKey                 = "signing_algorithms"

	// computed-only parameters
	authmethodOidcCallbackUrlKey      = "callback_url"
	authmethodOidcClientSecretHmacKey = "client_secret_hmac"
	authmethodOidcStateKey            = "state"
)

func resourceAuthMethodOidc() *schema.Resource {
	return &schema.Resource{
		Description: "The OIDC auth method resource allows you to configure a Boundary auth_method_oidc.",

		CreateContext: resourceAuthMethodCreate,
		ReadContext:   resourceAuthMethodRead,
		UpdateContext: resourceAuthMethodUpdate,
		DeleteContext: resourceAuthMethodDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the account.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The auth method name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The auth method description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope ID.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},

			// OIDC specific configurable parameters
			authmethodOidcAllowedAudiencesKey: {
				Description: "OIDC allowed audiences",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				Computed: true,
			},
			authmethodOidcApiUrlPrefixKey: {
				Description: "OIDC API URL prefix",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcCaCertificatesKey: {
				Description: "OIDC CA certificates",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				Computed: true,
			},
			authmethodOidcClientIdKey: {
				Description: "OIDC client ID",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcClientSecretKey: {
				Description: "OIDC client secret",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcIssuerKey: {
				Description: "OIDC discovery URL",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcDisableDiscoveredConfigValidationKey: {
				Description: "OIDC disable the discovered configuration validation",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcMaxAgeKey: {
				Description: "OIDC max age",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcSigningAlgorithmsKey: {
				Description: "OIDC signing algorithms",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},

			// OIDC specific immutable and computed parameters
			authmethodOidcClientSecretHmacKey: {
				Description: "OIDC client secret HMAC",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcStateKey: {
				Description: "OIDC state",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcCallbackUrlKey: {
				Description: "OIDC callback URL",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			TypeKey: {
				Description: "The resource type, hardcoded per resource",
				Type:        schema.TypeString,
				Default:     authmethodTypeOidc,
				Optional:    true,
			},
		},
	}
}
