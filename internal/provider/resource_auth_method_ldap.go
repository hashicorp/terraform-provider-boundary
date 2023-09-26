// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	authMethodTypeLdap = "ldap"

	authMethodLdapStartTlsField                  = "start_tls"
	authMethodLdapInsecureTlsField               = "insecure_tls"
	authMethodLdapDiscoverDnField                = "discover_dn"
	authMethodLdapAnonGrpSearchField             = "anon_group_search"
	authMethodLdapUpnDomainField                 = "upn_domain"
	authMethodLdapUrlsField                      = "urls"
	authMethodLdapUserDnField                    = "user_dn"
	authMethodLdapUserAttrField                  = "user_attr"
	authMethodLdapUserFilterField                = "user_filter"
	authMethodLdapEnableGrpsField                = "enable_groups"
	authMethodLdapGroupDnField                   = "group_dn"
	authMethodLdapGroupAttrField                 = "group_attr"
	authMethodLdapGroupFilterField               = "group_filter"
	authMethodLdapCertificatesField              = "certificates"
	authMethodLdapClientCertField                = "client_certificate"
	authMethodLdapClientCertKeyField             = "client_certificate_key"
	authMethodLdapBindDnField                    = "bind_dn"
	authMethodLdapBindPasswordField              = "bind_password"
	authMethodLdapUseTokenGrpsField              = "use_token_groups"
	authMethodLdapAccountAttrMapsField           = "account_attribute_maps"
	authMethodLdapPrimaryAuthMethodForScopeField = "is_primary_for_scope"
	authMethodLdapStateField                     = "state"
	authMethodLdapMaxPageSizeField               = "maximum_page_size"
	authMethodLdapDerefAliasesField              = "dereference_aliases"

	// computed-only parameters
	authMethodLdapClientCertKeyHmacKey = "client_certificate_key_hmac"
	authMethodLdapBindPasswordHmacKey  = "bind_password_hmac"
)

func resourceAuthMethodLdap() *schema.Resource {
	return &schema.Resource{
		Description: "The LDAP auth method resource allows you to configure a Boundary auth_method_ldap.",

		CreateContext: resourceAuthMethodLdapCreate,
		ReadContext:   resourceAuthMethodLdapRead,
		UpdateContext: resourceAuthMethodLdapUpdate,
		DeleteContext: resourceAuthMethodDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the auth method.",
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

			// LDAP specific configurable parameters
			authMethodLdapStartTlsField: {
				Description: "Issue StartTLS command after connecting (optional).",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			authMethodLdapInsecureTlsField: {
				Description: "Skip the LDAP server SSL certificate validation (optional) - insecure and use with caution.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			authMethodLdapDiscoverDnField: {
				Description: "Use anon bind to discover the bind DN of a user (optional).",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			authMethodLdapAnonGrpSearchField: {
				Description: "Use anon bind when performing LDAP group searches (optional).",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			authMethodLdapUpnDomainField: {
				Description: "The userPrincipalDomain used to construct the UPN string for the authenticating user (optional).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authMethodLdapUrlsField: {
				Description: "The LDAP URLs that specify LDAP servers to connect to (required).  May be specified multiple times.",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			authMethodLdapUserDnField: {
				Description: "The base DN under which to perform user search (optional).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authMethodLdapUserAttrField: {
				Description: "The attribute on user entry matching the username passed when authenticating (optional).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authMethodLdapUserFilterField: {
				Description: "A go template used to construct a LDAP user search filter (optional).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authMethodLdapEnableGrpsField: {
				Description: "Find the authenticated user's groups during authentication (optional).",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			authMethodLdapGroupDnField: {
				Description: "The base DN under which to perform group search.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authMethodLdapGroupAttrField: {
				Description: "The attribute that enumerates a user's group membership from entries returned by a group search (optional).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authMethodLdapGroupFilterField: {
				Description: "A go template used to construct a LDAP group search filter (optional).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authMethodLdapCertificatesField: {
				Description: "PEM-encoded X.509 CA certificate in ASN.1 DER form that can be used as a trust anchor when connecting to an LDAP server(optional).  This may be specified multiple times",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			authMethodLdapClientCertField: {
				Description: "PEM-encoded X.509 client certificate in ASN.1 DER form that can be used to authenticate against an LDAP server(optional).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authMethodLdapClientCertKeyField: {
				Description: "PEM-encoded X.509 client certificate key in PKCS #8, ASN.1 DER form used with the client certificate (optional).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authMethodLdapBindDnField: {
				Description: "The distinguished name of entry to bind when performing user and group searches (optional).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authMethodLdapBindPasswordField: {
				Description: "The password to use along with bind-dn performing user and group searches (optional).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			authMethodLdapUseTokenGrpsField: {
				Description: "Use the Active Directory tokenGroups constructed attribute of the user to find the group memberships (optional).",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			authMethodLdapAccountAttrMapsField: {
				Description: "Account attribute maps fullname and email.",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			authMethodLdapStateField: {
				Description: "Can be one of 'inactive', 'active-private', or 'active-public'. Defaults to active-public.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authMethodLdapMaxPageSizeField: {
				Description: "MaximumPageSize specifies a maximum search result size to use when retrieving the authenticated user's groups (optional).",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			authMethodLdapDerefAliasesField: {
				Description: "Control how aliases are dereferenced when performing the search. Can be one of: NeverDerefAliases, DerefInSearching, DerefFindingBaseObj, and DerefAlways (optional).",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},

			// LDAP specific immutable and computed parameters
			authMethodLdapClientCertKeyHmacKey: {
				Description: "The HMAC of the client certificate key returned by the Boundary controller, which is used for comparison after initial setting of the value.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			authMethodLdapBindPasswordHmacKey: {
				Description: "The HMAC of the bind password returned by the Boundary controller, which is used for comparison after initial setting of the value.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},

			authMethodLdapPrimaryAuthMethodForScopeField: {
				Description: "When true, makes this auth method the primary auth method for the scope in which it resides. The primary auth method for a scope means the the user will be automatically created when they login using an LDAP account.",
				Type:        schema.TypeBool,
				Optional:    true,
			},

			TypeKey: {
				Description: "The type of auth method; hardcoded.",
				Type:        schema.TypeString,
				Default:     authMethodTypeLdap,
				Optional:    true,
			},
		},
	}
}

func setFromLdapAuthMethodResponseMap(d *schema.ResourceData, raw map[string]interface{}) error {
	if err := d.Set(NameKey, raw[NameKey]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw[DescriptionKey]); err != nil {
		return err
	}
	if err := d.Set(ScopeIdKey, raw[ScopeIdKey]); err != nil {
		return err
	}
	if err := d.Set(TypeKey, raw[TypeKey]); err != nil {
		return err
	}

	if attrsVal, ok := raw["attributes"]; ok {
		attrs := attrsVal.(map[string]interface{})

		// these are always set
		if err := d.Set(authMethodLdapUrlsField, attrs[authMethodLdapUrlsField].([]interface{})); err != nil {
			return err
		}
		if err := d.Set(authMethodLdapStateField, attrs[authMethodLdapStateField].(string)); err != nil {
			return err
		}

		// optional attributes
		if v, ok := attrs[authMethodLdapStartTlsField]; ok {
			if err := d.Set(authMethodLdapStartTlsField, v.(bool)); err != nil {
				return err
			}
		} else {
			d.Set(authMethodLdapStartTlsField, false)
		}

		if v, ok := attrs[authMethodLdapInsecureTlsField]; ok {
			if err := d.Set(authMethodLdapInsecureTlsField, v.(bool)); err != nil {
				return err
			}
		} else {
			if err := d.Set(authMethodLdapInsecureTlsField, false); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapDiscoverDnField]; ok {
			if err := d.Set(authMethodLdapDiscoverDnField, v.(bool)); err != nil {
				return err
			}
		} else {
			if err := d.Set(authMethodLdapDiscoverDnField, false); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapAnonGrpSearchField]; ok {
			if err := d.Set(authMethodLdapAnonGrpSearchField, v.(bool)); err != nil {
				return err
			}
		} else {
			if err := d.Set(authMethodLdapAnonGrpSearchField, false); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapUpnDomainField]; ok {
			if err := d.Set(authMethodLdapUpnDomainField, v.(string)); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapUserDnField]; ok {
			if err := d.Set(authMethodLdapUserDnField, v.(string)); err != nil {
				return err
			}
		}
		if v, ok := attrs[authMethodLdapUserAttrField]; ok {
			if err := d.Set(authMethodLdapUserAttrField, v.(string)); err != nil {
				return err
			}
		}
		if v, ok := attrs[authMethodLdapUserFilterField]; ok {
			if err := d.Set(authMethodLdapUserFilterField, v.(string)); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapEnableGrpsField]; ok {
			if err := d.Set(authMethodLdapEnableGrpsField, v.(bool)); err != nil {
				return err
			}
		} else {
			if err := d.Set(authMethodLdapEnableGrpsField, false); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapGroupDnField]; ok {
			if err := d.Set(authMethodLdapGroupDnField, v.(string)); err != nil {
				return err
			}
		}
		if v, ok := attrs[authMethodLdapGroupAttrField]; ok {
			if err := d.Set(authMethodLdapGroupAttrField, v.(string)); err != nil {
				return err
			}
		}
		if v, ok := attrs[authMethodLdapGroupFilterField]; ok {
			if err := d.Set(authMethodLdapGroupFilterField, v.(string)); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapCertificatesField]; ok {
			if err := d.Set(authMethodLdapCertificatesField, v.([]interface{})); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapClientCertField]; ok {
			if err := d.Set(authMethodLdapClientCertField, v.(string)); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapClientCertKeyField]; ok {
			if err := d.Set(authMethodLdapClientCertKeyField, v.(string)); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapClientCertKeyHmacKey]; ok {
			if err := d.Set(authMethodLdapClientCertKeyHmacKey, v.(string)); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapBindDnField]; ok {
			if err := d.Set(authMethodLdapBindDnField, v.(string)); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapBindPasswordField]; ok {
			if err := d.Set(authMethodLdapBindPasswordField, v.(string)); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapBindPasswordHmacKey]; ok {
			if err := d.Set(authMethodLdapBindPasswordHmacKey, v.(string)); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapUseTokenGrpsField]; ok {
			if err := d.Set(authMethodLdapUseTokenGrpsField, v.(bool)); err != nil {
				return err
			}
		} else {
			if err := d.Set(authMethodLdapUseTokenGrpsField, false); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapAccountAttrMapsField]; ok {
			if err := d.Set(authMethodLdapAccountAttrMapsField, v.([]interface{})); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapStateField]; ok {
			if err := d.Set(authMethodLdapStateField, v.(string)); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapMaxPageSizeField]; ok {
			if err := d.Set(authMethodLdapMaxPageSizeField, v.(json.Number)); err != nil {
				return err
			}
		}

		if v, ok := attrs[authMethodLdapDerefAliasesField]; ok {
			if err := d.Set(authMethodLdapDerefAliasesField, v.(string)); err != nil {
				return err
			}
		}
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceAuthMethodLdapCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var typeStr string
	if typeVal, ok := d.GetOk(TypeKey); ok {
		typeStr = typeVal.(string)
	} else {
		return diag.Errorf("no type provided")
	}

	opts := []authmethods.Option{}

	opts = append(opts, authmethods.WithAutomaticVersioning(true))

	if rawUrls, ok := d.GetOk(authMethodLdapUrlsField); ok {
		urls := []string{}
		for _, u := range rawUrls.([]interface{}) {
			urls = append(urls, u.(string))
		}
		opts = append(opts, authmethods.WithLdapAuthMethodUrls(urls))
	}

	if startTls, ok := d.GetOk(authMethodLdapStartTlsField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodStartTls(startTls.(bool)))
	}

	if insecureTls, ok := d.GetOk(authMethodLdapInsecureTlsField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodInsecureTls(insecureTls.(bool)))
	}

	if discoverDn, ok := d.GetOk(authMethodLdapDiscoverDnField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodDiscoverDn(discoverDn.(bool)))
	}

	if anonGrpSearch, ok := d.GetOk(authMethodLdapAnonGrpSearchField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodAnonGroupSearch(anonGrpSearch.(bool)))
	}

	if upnDomain, ok := d.GetOk(authMethodLdapUpnDomainField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodUpnDomain(upnDomain.(string)))
	}

	if userDn, ok := d.GetOk(authMethodLdapUserDnField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodUserDn(userDn.(string)))
	}

	if userAttr, ok := d.GetOk(authMethodLdapUserAttrField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodUserAttr(userAttr.(string)))
	}

	if userFilter, ok := d.GetOk(authMethodLdapUserFilterField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodUserFilter(userFilter.(string)))
	}

	if enableGrps, ok := d.GetOk(authMethodLdapEnableGrpsField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodEnableGroups(enableGrps.(bool)))
	}

	if groupDn, ok := d.GetOk(authMethodLdapGroupDnField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodGroupDn(groupDn.(string)))
	}

	if groupAttr, ok := d.GetOk(authMethodLdapGroupAttrField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodGroupAttr(groupAttr.(string)))
	}

	if groupFilter, ok := d.GetOk(authMethodLdapGroupFilterField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodGroupFilter(groupFilter.(string)))
	}

	if rawCerts, ok := d.GetOk(authMethodLdapCertificatesField); ok {
		certs := []string{}
		for _, u := range rawCerts.([]interface{}) {
			certs = append(certs, u.(string))
		}
		opts = append(opts, authmethods.WithLdapAuthMethodCertificates(certs))
	}

	if clientCert, ok := d.GetOk(authMethodLdapClientCertField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodClientCertificate(clientCert.(string)))
	}

	if clientCertKey, ok := d.GetOk(authMethodLdapClientCertKeyField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodClientCertificateKey(clientCertKey.(string)))
	}

	if bindDn, ok := d.GetOk(authMethodLdapBindDnField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodBindDn(bindDn.(string)))
	}

	if bindPassword, ok := d.GetOk(authMethodLdapBindPasswordField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodBindPassword(bindPassword.(string)))
	}

	if useTokenGrps, ok := d.GetOk(authMethodLdapUseTokenGrpsField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodUseTokenGroups(useTokenGrps.(bool)))
	}

	if maxPageSize, ok := d.GetOk(authMethodLdapMaxPageSizeField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodMaximumPageSize(uint32(maxPageSize.(int))))
	}

	if derefAliases, ok := d.GetOk(authMethodLdapDerefAliasesField); ok {
		opts = append(opts, authmethods.WithLdapAuthMethodDereferenceAliases(derefAliases.(string)))
	}

	if rawAccountAttrMaps, ok := d.GetOk(authMethodLdapAccountAttrMapsField); ok {
		accountAttrMaps := []string{}
		for _, u := range rawAccountAttrMaps.([]interface{}) {
			accountAttrMaps = append(accountAttrMaps, u.(string))
		}
		opts = append(opts, authmethods.WithLdapAuthMethodAccountAttributeMaps(accountAttrMaps))
	}

	state, ok := d.GetOk(authMethodLdapStateField)
	switch ok {
	case true:
		opts = append(opts, authmethods.WithLdapAuthMethodState(state.(string)))
	default:
		opts = append(opts, authmethods.WithLdapAuthMethodState("active-public"))
	}

	if nameVal, ok := d.GetOk(NameKey); ok {
		nameStr := nameVal.(string)
		opts = append(opts, authmethods.WithName(nameStr))
	}

	if descVal, ok := d.GetOk(DescriptionKey); ok {
		descStr := descVal.(string)
		opts = append(opts, authmethods.WithDescription(descStr))
	}

	var scopeId string
	scopeIdVal, ok := d.GetOk(ScopeIdKey)
	switch ok {
	case true:
		scopeId = scopeIdVal.(string)
	default:
		return diag.Errorf("no scope ID provided")
	}

	amClient := authmethods.NewClient(md.client)

	amcr, err := amClient.Create(ctx, typeStr, scopeId, opts...)
	switch {
	case err != nil:
		return diag.Errorf("error creating auth method: %v", err)
	case amcr == nil:
		return diag.Errorf("nil auth method after create")
	}

	if err := setFromLdapAuthMethodResponseMap(d, amcr.GetResponse().Map); err != nil {
		return diag.Errorf("%v", err)
	}

	amid := amcr.GetResponse().Map["id"].(string)

	// update scope when set to primary
	if p, ok := d.GetOk(authMethodLdapPrimaryAuthMethodForScopeField); ok {
		if p.(bool) {
			if err := updateScopeWithPrimaryAuthMethodId(ctx, scopeId, amid, meta); err != nil {
				return diag.Errorf("%v", err)
			}

			amcr.GetResponse().Map[authMethodLdapPrimaryAuthMethodForScopeField] = true
		}
	}

	if err := setFromLdapAuthMethodResponseMap(d, amcr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceAuthMethodLdapRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	amrr, err := amClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading auth method: %v", err)
	}
	if amrr == nil {
		return diag.Errorf("auth method nil after read")
	}

	serr, isPrimary := readScopeIsPrimaryAuthMethodId(ctx, amrr.GetResponse().Map["scope_id"].(string), amrr.GetResponse().Map["id"].(string), meta)
	if err != nil {
		return diag.Errorf("%v", serr)
	}

	if isPrimary {
		amrr.GetResponse().Map[authMethodLdapPrimaryAuthMethodForScopeField] = true
	}

	if err := setFromLdapAuthMethodResponseMap(d, amrr.GetResponse().Map); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceAuthMethodLdapUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	amClient := authmethods.NewClient(md.client)

	opts := []authmethods.Option{}

	if d.HasChange(NameKey) {
		opts = append(opts, authmethods.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			opts = append(opts, authmethods.WithName(nameVal.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, authmethods.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			opts = append(opts, authmethods.WithDescription(descVal.(string)))
		}
	}

	if d.HasChange(authMethodLdapUrlsField) {
		if rawUrls, ok := d.GetOk(authMethodLdapUrlsField); ok {
			urls := []string{}
			for _, u := range rawUrls.([]interface{}) {
				urls = append(urls, u.(string))
			}
			opts = append(opts, authmethods.WithLdapAuthMethodUrls(urls))
		}
	}

	if d.HasChange(authMethodLdapStartTlsField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodStartTls())
		if startTls, ok := d.GetOk(authMethodLdapStartTlsField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodStartTls(startTls.(bool)))
		}
	}

	if d.HasChange(authMethodLdapInsecureTlsField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodInsecureTls())
		if insecureTls, ok := d.GetOk(authMethodLdapInsecureTlsField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodInsecureTls(insecureTls.(bool)))
		}
	}

	if d.HasChange(authMethodLdapDiscoverDnField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodDiscoverDn())
		if discoverDn, ok := d.GetOk(authMethodLdapDiscoverDnField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodDiscoverDn(discoverDn.(bool)))
		}
	}

	if d.HasChange(authMethodLdapAnonGrpSearchField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodAnonGroupSearch())
		if AnonGrpSearch, ok := d.GetOk(authMethodLdapAnonGrpSearchField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodAnonGroupSearch(AnonGrpSearch.(bool)))
		}
	}

	if d.HasChange(authMethodLdapUpnDomainField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodUpnDomain())
		if upnDomain, ok := d.GetOk(authMethodLdapUpnDomainField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodUpnDomain(upnDomain.(string)))
		}
	}

	if d.HasChange(authMethodLdapUserDnField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodUserDn())
		if userDn, ok := d.GetOk(authMethodLdapUserDnField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodUserDn(userDn.(string)))
		}
	}

	if d.HasChange(authMethodLdapUserAttrField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodUserAttr())
		if userAttr, ok := d.GetOk(authMethodLdapUserAttrField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodUserAttr(userAttr.(string)))
		}
	}

	if d.HasChange(authMethodLdapUserFilterField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodUserFilter())
		if userFilter, ok := d.GetOk(authMethodLdapUserFilterField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodUserFilter(userFilter.(string)))
		}
	}

	if d.HasChange(authMethodLdapEnableGrpsField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodEnableGroups())
		if enableGrps, ok := d.GetOk(authMethodLdapEnableGrpsField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodEnableGroups(enableGrps.(bool)))
		}
	}

	if d.HasChange(authMethodLdapGroupDnField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodGroupDn())
		if groupDn, ok := d.GetOk(authMethodLdapGroupDnField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodGroupDn(groupDn.(string)))
		}
	}

	if d.HasChange(authMethodLdapGroupAttrField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodGroupAttr())
		if groupAttr, ok := d.GetOk(authMethodLdapGroupAttrField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodGroupAttr(groupAttr.(string)))
		}
	}

	if d.HasChange(authMethodLdapGroupFilterField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodGroupFilter())
		if groupFilter, ok := d.GetOk(authMethodLdapGroupFilterField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodGroupFilter(groupFilter.(string)))
		}
	}

	if d.HasChange(authMethodLdapCertificatesField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodCertificates())
		if rawCerts, ok := d.GetOk(authMethodLdapCertificatesField); ok {
			certs := []string{}
			for _, u := range rawCerts.([]interface{}) {
				certs = append(certs, u.(string))
			}
			opts = append(opts, authmethods.WithLdapAuthMethodCertificates(certs))
		}
	}

	if d.HasChange(authMethodLdapClientCertField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodClientCertificate())
		if clientCert, ok := d.GetOk(authMethodLdapClientCertField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodClientCertificate(clientCert.(string)))
		}
	}

	if d.HasChange(authMethodLdapClientCertKeyField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodClientCertificateKey())
		if clientCertKey, ok := d.GetOk(authMethodLdapClientCertKeyField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodClientCertificateKey(clientCertKey.(string)))
		}
	}

	if d.HasChange(authMethodLdapBindDnField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodBindDn())
		if bindDn, ok := d.GetOk(authMethodLdapBindDnField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodBindDn(bindDn.(string)))
		}
	}

	if d.HasChange(authMethodLdapBindPasswordField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodBindPassword())
		if bindPassword, ok := d.GetOk(authMethodLdapBindPasswordField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodBindPassword(bindPassword.(string)))
		}
	}

	if d.HasChange(authMethodLdapUseTokenGrpsField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodUseTokenGroups())
		if useTokenGrps, ok := d.GetOk(authMethodLdapUseTokenGrpsField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodUseTokenGroups(useTokenGrps.(bool)))
		}
	}

	if d.HasChange(authMethodLdapAccountAttrMapsField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodAccountAttributeMaps())
		if rawAccountAttrMaps, ok := d.GetOk(authMethodLdapAccountAttrMapsField); ok {
			accountAttrMaps := []string{}
			for _, u := range rawAccountAttrMaps.([]interface{}) {
				accountAttrMaps = append(accountAttrMaps, u.(string))
			}
			opts = append(opts, authmethods.WithLdapAuthMethodAccountAttributeMaps(accountAttrMaps))
		}
	}

	if d.HasChange(authMethodLdapStateField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodState())
		if state, ok := d.GetOk(authMethodLdapStateField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodState(state.(string)))
		}
	}

	if d.HasChange(authMethodLdapMaxPageSizeField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodMaximumPageSize())
		if maxPageSize, ok := d.GetOk(authMethodLdapMaxPageSizeField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodMaximumPageSize(uint32(maxPageSize.(int))))
		}
	}

	if d.HasChange(authMethodLdapDerefAliasesField) {
		opts = append(opts, authmethods.DefaultLdapAuthMethodDereferenceAliases())
		if derefAliases, ok := d.GetOk(authMethodLdapDerefAliasesField); ok {
			opts = append(opts, authmethods.WithLdapAuthMethodDereferenceAliases(derefAliases.(string)))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, authmethods.WithAutomaticVersioning(true))
		amur, err := amClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating auth method: %v", err)
		}

		if d.HasChange(authMethodLdapPrimaryAuthMethodForScopeField) {
			if p, ok := d.GetOk(authMethodLdapPrimaryAuthMethodForScopeField); ok {
				if p.(bool) {
					if err := updateScopeWithPrimaryAuthMethodId(
						ctx,
						amur.GetResponse().Map["scope_id"].(string),
						amur.GetResponse().Map["id"].(string),
						meta); err != nil {
						return diag.Errorf("%v", err)
					}

					amur.GetResponse().Map[authMethodLdapPrimaryAuthMethodForScopeField] = true
				}
			}
		}

		if err := setFromLdapAuthMethodResponseMap(d, amur.GetResponse().Map); err != nil {
			return diag.FromErr(err)
		}
		return nil
	}
	return nil
}
