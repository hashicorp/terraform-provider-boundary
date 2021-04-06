package provider

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

const (
	authmethodTypePassword          = "password"
	authmethodMinLoginNameLengthKey = "min_login_name_length"
	authmethodMinPasswordLengthKey  = "min_password_length"
)

func resourceAuthMethodPassword() *schema.Resource {
	return &schema.Resource{
		Description: "The auth method resource allows you to configure a Boundary auth_method_password.",

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
				Description: "The resource type, hardcoded per resource",
				Type:        schema.TypeString,
				Default:     authmethodTypePassword,
				Optional:    true,
			},
			authmethodMinLoginNameLengthKey: {
				Description: "The minimum login name length.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			authmethodMinPasswordLengthKey: {
				Description: "The minimum password length.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
		},
	}
}
