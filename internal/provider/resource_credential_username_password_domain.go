// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/credentials"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	credentialUsernamePasswordDomainUsernameKey     = "username"
	credentialUsernamePasswordDomainPasswordKey     = "password"
	credentialUsernamePasswordDomainPasswordHmacKey = "password_hmac"
	credentialUsernamePasswordDomainDomainKey       = "domain"
	credentialUsernamePasswordDomainCredentialType  = "username_password_domain"
)

func resourceCredentialUsernamePasswordDomain() *schema.Resource {
	return &schema.Resource{
		Description: "The username-password-domain credential resource allows you to configure a credential using a username, password and domain.",

		CreateContext: resourceCredentialUsernamePasswordDomainCreate,
		ReadContext:   resourceCredentialUsernamePasswordDomainRead,
		UpdateContext: resourceCredentialUsernamePasswordDomainUpdate,
		DeleteContext: resourceCredentialUsernamePasswordDomainDelete,
		CustomizeDiff: resourceCredentialUsernamePasswordDomainCustomizeDiff,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of this username-password-domain credential.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The name of this username-password-domain credential. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The description of this username-password-domain credential.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			credentialStoreIdKey: {
				Description: "The credential store in which to save this username-password-domain credential.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			credentialUsernamePasswordDomainUsernameKey: {
				Description: "This field is required even though it is marked as optional. The username of this username-password-domain credential. Can also contain a domain if provided as username@domain or domain\\username",
				Type:        schema.TypeString,
				Optional:    true, // Explicitly set as optional so that Terraform doesn't interpret this field as read-only.
				Computed:    true, // Required and Computed are mutually exclusive.
			},
			credentialUsernamePasswordDomainPasswordKey: {
				Description: "The password of this username-password-domain credential.",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			credentialUsernamePasswordDomainPasswordHmacKey: {
				Description: "The password hmac.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			credentialUsernamePasswordDomainDomainKey: {
				Description: "The domain of this username-password-domain credential. Can be provided as part of the username field instead (see username field description).",
				Type:        schema.TypeString,
				Optional:    true, // Explicitly set as optional so that Terraform doesn't interpret this field as read-only.
				Computed:    true, // Required and Computed are mutually exclusive.
			},
		},
	}
}

func setFromCredentialUsernamePasswordDomainResponseMap(d *schema.ResourceData, raw map[string]interface{}, fromRead bool) error {
	if err := d.Set(NameKey, raw[NameKey]); err != nil {
		return err
	}
	if err := d.Set(DescriptionKey, raw[DescriptionKey]); err != nil {
		return err
	}
	if err := d.Set(credentialStoreIdKey, raw[credentialStoreIdKey]); err != nil {
		return err
	}

	if attrsVal, ok := raw["attributes"]; ok {
		attrs := attrsVal.(map[string]interface{})
		if err := d.Set(credentialUsernamePasswordDomainUsernameKey, attrs[credentialUsernamePasswordDomainUsernameKey]); err != nil {
			return err
		}

		statePasswordHmac := d.Get(credentialUsernamePasswordDomainPasswordHmacKey)
		boundaryPasswordHmac := attrs[credentialUsernamePasswordDomainPasswordHmacKey].(string)
		if statePasswordHmac.(string) != boundaryPasswordHmac && fromRead {
			// PasswordHmac has changed in Boundary, therefore the password has changed.
			// Update password value to force tf to attempt update.
			if err := d.Set(credentialUsernamePasswordDomainPasswordKey, "(changed in Boundary)"); err != nil {
				return err
			}
		}
		if err := d.Set(credentialUsernamePasswordDomainPasswordHmacKey, boundaryPasswordHmac); err != nil {
			return err
		}

		if err := d.Set(credentialUsernamePasswordDomainDomainKey, attrs[credentialUsernamePasswordDomainDomainKey]); err != nil {
			return err
		}
	}

	d.SetId(raw["id"].(string))

	return nil
}

func resourceCredentialUsernamePasswordDomainCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)

	var opts []credentials.Option
	if v, ok := d.GetOk(NameKey); ok {
		opts = append(opts, credentials.WithName(v.(string)))
	}
	if v, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, credentials.WithDescription(v.(string)))
	}

	// Username and Domain can be set separately, but can also be set together
	// in the form of "username@domain" or "domain\\username".
	// We need to parse them out in order to set them correctly.
	var usernameKey, domainKey = "", ""
	if v, ok := d.GetOk(credentialUsernamePasswordDomainUsernameKey); ok {
		usernameKey = v.(string)
	}
	if v, ok := d.GetOk(credentialUsernamePasswordDomainDomainKey); ok {
		domainKey = v.(string)
	}

	username, domain, err := credentials.ParseUsernameDomain(usernameKey, domainKey)
	if err != nil {
		return diag.Errorf("error parsing username and domain: %v", err)
	}

	switch username {
	case "":
	default:
		opts = append(opts, credentials.WithUsernamePasswordDomainCredentialUsername(username))
	}
	switch domain {
	case "":
	default:
		opts = append(opts, credentials.WithUsernamePasswordDomainCredentialDomain(domain))
	}

	if v, ok := d.GetOk(credentialUsernamePasswordDomainPasswordKey); ok {
		opts = append(opts, credentials.WithUsernamePasswordDomainCredentialPassword(v.(string)))
	}

	var credentialStoreId string
	retrievedStoreId, ok := d.GetOk(credentialStoreIdKey)
	if ok {
		credentialStoreId = retrievedStoreId.(string)
	} else {
		return diag.Errorf("credential store id is unset")
	}

	client := credentials.NewClient(md.client)
	cr, err := client.Create(ctx, credentialUsernamePasswordDomainCredentialType, credentialStoreId, opts...)
	if err != nil {
		return diag.Errorf("error creating credential: %v", err)
	}
	if cr == nil {
		return diag.Errorf("nil credential after create")
	}

	if err := setFromCredentialUsernamePasswordDomainResponseMap(d, cr.GetResponse().Map, false); err != nil {
		return diag.Errorf("error generating credential from response map: %v", err)
	}

	return nil
}

func resourceCredentialUsernamePasswordDomainRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentials.NewClient(md.client)

	cr, err := client.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading credential: %v", err)
	}
	if cr == nil {
		return diag.Errorf("credential nil after read")
	}

	if err := setFromCredentialUsernamePasswordDomainResponseMap(d, cr.GetResponse().Map, true); err != nil {
		return diag.Errorf("error generating credential from response map: %v", err)
	}

	return nil
}

func resourceCredentialUsernamePasswordDomainUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentials.NewClient(md.client)

	var opts []credentials.Option
	if d.HasChange(NameKey) {
		opts = append(opts, credentials.DefaultName())
		nameVal, ok := d.GetOk(NameKey)
		if ok {
			opts = append(opts, credentials.WithName(nameVal.(string)))
		}
	}

	if d.HasChange(DescriptionKey) {
		opts = append(opts, credentials.DefaultDescription())
		descVal, ok := d.GetOk(DescriptionKey)
		if ok {
			opts = append(opts, credentials.WithDescription(descVal.(string)))
		}
	}

	// Username and Domain can be set separately, but can also be set together
	// in the form of "username@domain" or "domain\\username".
	// Since we do not know if they were originally set together or separately,
	// we need to check if either of them has changed.
	if d.HasChange(credentialUsernamePasswordDomainUsernameKey) || d.HasChange(credentialUsernamePasswordDomainDomainKey) {
		var usernameKey, domainKey = "", ""
		if v, ok := d.GetOk(credentialUsernamePasswordDomainUsernameKey); ok {
			usernameKey = v.(string)
		}
		if v, ok := d.GetOk(credentialUsernamePasswordDomainDomainKey); ok {
			domainKey = v.(string)
		}

		username, domain, err := credentials.ParseUsernameDomain(usernameKey, domainKey)
		if err != nil {
			return diag.Errorf("error parsing username and domain: %v", err)
		}

		switch username {
		case "":
		default:
			opts = append(opts, credentials.WithUsernamePasswordDomainCredentialUsername(username))
		}
		switch domain {
		case "":
		default:
			opts = append(opts, credentials.WithUsernamePasswordDomainCredentialDomain(domain))
		}
	}

	if d.HasChange(credentialUsernamePasswordDomainPasswordKey) {
		passwordVal, ok := d.GetOk(credentialUsernamePasswordDomainPasswordKey)
		if ok {
			opts = append(opts, credentials.WithUsernamePasswordDomainCredentialPassword(passwordVal.(string)))
		}
	}

	if len(opts) > 0 {
		opts = append(opts, credentials.WithAutomaticVersioning(true))
		crUpdate, err := client.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error updating credential: %v", err)
		}
		if crUpdate == nil {
			return diag.Errorf("credential nil after update")
		}

		if err = setFromCredentialUsernamePasswordDomainResponseMap(d, crUpdate.GetResponse().Map, false); err != nil {
			return diag.Errorf("error generating credential from response map: %v", err)
		}
	}

	return nil
}

func resourceCredentialUsernamePasswordDomainDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	client := credentials.NewClient(md.client)

	_, err := client.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting credential: %v", err)
	}

	return nil
}

func resourceCredentialUsernamePasswordDomainCustomizeDiff(ctx context.Context, rd *schema.ResourceDiff, _ interface{}) error {
	usernameRaw := rd.Get(credentialUsernamePasswordDomainUsernameKey).(string)
	domainRaw := rd.Get(credentialUsernamePasswordDomainDomainKey).(string)

	if usernameRaw != "" && domainRaw != "" {
		if strings.Contains(usernameRaw, "@") || strings.Contains(usernameRaw, "\\") {
			// This is an error condition for ParseUsernameDomain, so handle it
			// here by blanking the domain input. We've already asserted that
			// the username appears to contain a domain, so we'll extract it
			// below.
			domainRaw = ""
		}
	}

	username, domain, err := credentials.ParseUsernameDomain(usernameRaw, domainRaw)
	if err != nil {
		return fmt.Errorf("failed to parse username/domain fields: %w", err)
	}
	// We can't set the fields as Required in the config (because we
	// have to set Computed), so enforce it here.
	if username == "" {
		return fmt.Errorf("username field is required")
	}
	if domain == "" {
		return fmt.Errorf("domain field is required")
	}

	err = rd.Clear(credentialUsernamePasswordDomainUsernameKey)
	if err != nil {
		return fmt.Errorf("failed to clear username field: %w", err)
	}
	err = rd.SetNew(credentialUsernamePasswordDomainUsernameKey, username)
	if err != nil {
		return fmt.Errorf("failed to set new username: %w", err)
	}

	err = rd.Clear(credentialUsernamePasswordDomainDomainKey)
	if err != nil {
		return fmt.Errorf("failed to clear domain field: %w", err)
	}
	err = rd.SetNew(credentialUsernamePasswordDomainDomainKey, domain)
	if err != nil {
		return fmt.Errorf("failed to set new username: %w", err)
	}

	return nil
}
