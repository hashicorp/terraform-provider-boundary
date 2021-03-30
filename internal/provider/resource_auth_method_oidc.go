package provider

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

const (
	authmethodTypeOidc                              = "oidc"
	authmethodOidcStateKey                          = "state"
	authmethodOidcDiscoveryUrlKey                   = "discovery_url"
	authmethodOidcClientIdKey                       = "client_id"
	authmethodOidcClientSecretKey                   = "client_secret"
	authmethodOidcClientSecretHmacKey               = "client_secret_hmac"
	authmethodOidcMaxAgeKey                         = "max_age"
	authmethodOidcSigningAlgorithmsKey              = "signing_algorithms"
	authmethodOidcApiUrlPrefixKey                   = "api_url_prefix"
	authmethodOidcCallbackUrlKey                    = "callback_url"
	authmethodOidcCertificatesKey                   = "certificates"
	authmethodOidcAllowedAudiencesKey               = "allowed_audiences"
	authmethodOidcOverrideOidcDiscoveryUrlConfigKey = "override_oidc_discovery_url_config"
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
			TypeKey: {
				Description: "The resource type.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},

			// OIDC method specific parameters
			authmethodOidcStateKey: {
				Description: "OIDC state",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcDiscoveryUrlKey: {
				Description: "OIDC discovery URL",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
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
			authmethodOidcClientSecretHmacKey: {
				Description: "OIDC client secret HMAC",
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
			authmethodOidcApiUrlPrefixKey: {
				Description: "OIDC API URL prefix",
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
			authmethodOidcCertificatesKey: {
				Description: "OIDC certificates",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcAllowedAudiencesKey: {
				Description: "OIDC allowed audiences",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcOverrideOidcDiscoveryUrlConfigKey: {
				Description: "OIDC discovery URL override configuration",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
		},
	}
}
