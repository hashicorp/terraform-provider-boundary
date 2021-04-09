package provider

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

const (
	authmethodTypeOidc                                 = "oidc"
	authmethodOidcIssuerKey                            = "issuer"
	authmethodOidcClientIdKey                          = "client_id"
	authmethodOidcClientSecretKey                      = "client_secret"
	authmethodOidcMaxAgeKey                            = "max_age"
	authmethodOidcApiUrlPrefixKey                      = "api_url_prefix"
	authmethodOidcIdpCaCertsKey                        = "idp_ca_certs"
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
				Description: "Audiences for which the provider responses will be allowed",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			authmethodOidcApiUrlPrefixKey: {
				Description: "The API prefix to use when generating callback URLs for the provider. Should be set to an address at which the provider can reach back to the controller.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authmethodOidcIdpCaCertsKey: {
				Description: "A list of CA certificates to trust when validating the IdP's token signatures.",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			authmethodOidcClientIdKey: {
				Description: "The client ID assigned to this auth method from the provider.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authmethodOidcClientSecretKey: {
				Description: "The secret key assigned to this auth method from the provider. Once set, only the hash will be kept and the original value can be removed from configuration.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authmethodOidcIssuerKey: {
				Description: "The issuer corresponding to the provider, which must match the issuer field in generated tokens.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authmethodOidcDisableDiscoveredConfigValidationKey: {
				Description: "Disables validation logic ensuring that the OIDC provider's information from its discovery endpoint matches the information here. The validation is only performed at create or update time.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authmethodOidcMaxAgeKey: {
				Description: "The max age to provide to the provider, indicating how much time is allowed to have passed since the last authentication before the user is challenged again.",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			authmethodOidcSigningAlgorithmsKey: {
				Description: "Allowed signing algorithms for the provider's issued tokens.",
				Type:        schema.TypeString,
				Optional:    true,
			},

			// OIDC specific immutable and computed parameters
			authmethodOidcClientSecretHmacKey: {
				Description: "The HMAC of the client secret returned by the Boundary controller, which is used for comparison after initial setting of the value.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcStateKey: {
				Description: "The current state of the auth method.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authmethodOidcCallbackUrlKey: {
				Description: "The URL that should be provided to the IdP for callbacks.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			TypeKey: {
				Description: "The type of auth method; hardcoded.",
				Type:        schema.TypeString,
				Default:     authmethodTypeOidc,
				Optional:    true,
			},
		},
	}
}
