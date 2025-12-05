// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/policies"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	policyStorageRetainForDaysKey          = "retain_for_days"
	policyStorageRetainForOverridableKey   = "retain_for_overridable"
	policyStorageDeleteAfterDaysKey        = "delete_after_days"
	policyStorageDeleteAfterOverridableKey = "delete_after_overridable"
	policyTypeStorage                      = "storage"
	daysField                              = "days"
	overridableField                       = "overridable"
)

func resourcePolicyStorage() *schema.Resource {
	return &schema.Resource{
		Description: "The storage policy resource allows you to configure a Boundary storage policy. " +
			"Storage policies allow an admin to configure how long session recordings must be stored and when " +
			"to delete them. Storage policies must be applied to the global scope or an org scope in order to take effect.",
		CreateContext: resourcePolicyStorageCreate,
		ReadContext:   resourcePolicyStorageRead,
		UpdateContext: resourcePolicyStorageUpdate,
		DeleteContext: resourcePolicyStorageDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IDKey: {
				Description: "The ID of the policy.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			NameKey: {
				Description: "The policy name. Defaults to the resource name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			DescriptionKey: {
				Description: "The policy description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			ScopeIdKey: {
				Description: "The scope for this policy.",
				Type:        schema.TypeString,
				Required:    true,
			},
			policyStorageRetainForDaysKey: {
				Description:  "The number of days a session recording is required to be stored. Defaults to 0: allow deletions at any time. However, " + policyStorageRetainForDaysKey + " and " + policyStorageDeleteAfterDaysKey + " cannot both be 0.",
				Type:         schema.TypeInt,
				Optional:     true,
				AtLeastOneOf: []string{policyStorageDeleteAfterDaysKey},
			},
			policyStorageRetainForOverridableKey: {
				Description: "Whether or not the associated " + policyStorageRetainForDaysKey + " value can be overridden by org scopes. Note: if the associated " + policyStorageRetainForDaysKey + " value is 0, overridable is ignored.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			policyStorageDeleteAfterDaysKey: {
				Description:  "The number of days after which a session recording will be automatically deleted. Defaults to 0: never automatically delete. However, " + policyStorageDeleteAfterDaysKey + " and " + policyStorageRetainForDaysKey + " cannot both be 0.",
				Type:         schema.TypeInt,
				Optional:     true,
				AtLeastOneOf: []string{policyStorageRetainForDaysKey},
			},
			policyStorageDeleteAfterOverridableKey: {
				Description: "Whether or not the associated " + policyStorageDeleteAfterDaysKey + " value can be overridden by org scopes. Note: if the associated " + policyStorageDeleteAfterDaysKey + " value is 0, overridable is ignored",
				Type:        schema.TypeBool,
				Optional:    true,
			},
		},
	}
}

func setFromPolicyItem(d *schema.ResourceData, p *policies.Policy) error {
	d.SetId(p.Id)

	if err := d.Set(NameKey, p.Name); err != nil {
		return err
	}

	if err := d.Set(DescriptionKey, p.Description); err != nil {
		return err
	}

	if err := d.Set(ScopeIdKey, p.ScopeId); err != nil {
		return err
	}

	// parse attributes to the retain/delete values so that the tf syntax is simpler
	attributes, err := p.GetStoragePolicyAttributes()
	if err != nil {
		return err
	}

	// retain_for attribute
	if attributes.RetainFor != nil {
		if err := d.Set(policyStorageRetainForDaysKey, attributes.RetainFor.Days); err != nil {
			return err
		}
		if err := d.Set(policyStorageRetainForOverridableKey, attributes.RetainFor.Overridable); err != nil {
			return err
		}
	} else {
		// no RetainFor, set to nil
		if err := d.Set(policyStorageRetainForDaysKey, nil); err != nil {
			return err
		}
		if err := d.Set(policyStorageRetainForOverridableKey, nil); err != nil {
			return err
		}
	}

	// delete_after attribute
	if attributes.DeleteAfter != nil {
		if err := d.Set(policyStorageDeleteAfterDaysKey, attributes.DeleteAfter.Days); err != nil {
			return err
		}
		if err := d.Set(policyStorageDeleteAfterOverridableKey, attributes.DeleteAfter.Overridable); err != nil {
			return err
		}
	} else {
		// no DeleteAfter, set to nil
		if err := d.Set(policyStorageDeleteAfterDaysKey, nil); err != nil {
			return err
		}
		if err := d.Set(policyStorageDeleteAfterOverridableKey, nil); err != nil {
			return err
		}
	}

	return nil
}

func resourcePolicyStorageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	pClient := policies.NewClient(md.client)
	opts := []policies.Option{}

	var scopeId string
	if scopeIdVal, ok := d.GetOk(ScopeIdKey); ok {
		scopeId = scopeIdVal.(string)
	} else {
		return diag.Errorf("no scope ID provided")
	}

	if nameVal, ok := d.GetOk(NameKey); ok {
		opts = append(opts, policies.WithName(nameVal.(string)))
	}

	if descVal, ok := d.GetOk(DescriptionKey); ok {
		opts = append(opts, policies.WithDescription(descVal.(string)))
	}

	if retainDaysVal, ok := d.GetOk(policyStorageRetainForDaysKey); ok {
		opts = append(opts, policies.WithStoragePolicyRetainForDays(int32(retainDaysVal.(int))))
	} else {
		opts = append(opts, policies.DefaultStoragePolicyRetainForDays())
	}
	if retainOverrideVal, ok := d.GetOk(policyStorageRetainForOverridableKey); ok {
		opts = append(opts, policies.WithStoragePolicyRetainForOverridable(retainOverrideVal.(bool)))
	} else {
		opts = append(opts, policies.DefaultStoragePolicyRetainForOverridable())
	}

	if deleteDaysVal, ok := d.GetOk(policyStorageDeleteAfterDaysKey); ok {
		opts = append(opts, policies.WithStoragePolicyDeleteAfterDays(int32(deleteDaysVal.(int))))
	} else {
		opts = append(opts, policies.DefaultStoragePolicyDeleteAfterDays())
	}
	if deleteOverrideVal, ok := d.GetOk(policyStorageDeleteAfterOverridableKey); ok {
		opts = append(opts, policies.WithStoragePolicyDeleteAfterOverridable(deleteOverrideVal.(bool)))
	} else {
		opts = append(opts, policies.DefaultStoragePolicyDeleteAfterOverridable())
	}

	p, err := pClient.Create(ctx, policyTypeStorage, scopeId, opts...)
	if err != nil {
		return diag.Errorf("error creating storage policy: %v", err)
	}
	if p == nil {
		return diag.Errorf("nil storage policy after create")
	}

	if err := setFromPolicyItem(d, p.GetItem()); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourcePolicyStorageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	pClient := policies.NewClient(md.client)

	p, err := pClient.Read(ctx, d.Id())
	if err != nil {
		if apiErr := api.AsServerError(err); apiErr != nil && apiErr.Response().StatusCode() == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error reading policy: %v", err)
	}
	if p == nil {
		return diag.Errorf("policy nil after read")
	}

	if err := setFromPolicyItem(d, p.GetItem()); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourcePolicyStorageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	pClient := policies.NewClient(md.client)
	opts := []policies.Option{}

	if d.HasChange(NameKey) {
		if nameVal, ok := d.GetOk(NameKey); ok {
			name := nameVal.(string)
			opts = append(opts, policies.WithName(name))
		} else {
			opts = append(opts, policies.DefaultName())
		}
	}

	if d.HasChange(DescriptionKey) {
		if descriptionVal, ok := d.GetOk(DescriptionKey); ok {
			description := descriptionVal.(string)
			opts = append(opts, policies.WithDescription(description))
		} else {
			opts = append(opts, policies.DefaultDescription())
		}
	}

	if d.HasChanges(policyStorageRetainForDaysKey, policyStorageRetainForOverridableKey) {
		if retainDaysVal, ok := d.GetOk(policyStorageRetainForDaysKey); ok {
			opts = append(opts, policies.WithStoragePolicyRetainForDays(int32(retainDaysVal.(int))))
		} else {
			opts = append(opts, policies.DefaultStoragePolicyRetainForDays())
		}
		if retainOverrideVal, ok := d.GetOk(policyStorageRetainForOverridableKey); ok {
			opts = append(opts, policies.WithStoragePolicyRetainForOverridable(retainOverrideVal.(bool)))
		} else {
			opts = append(opts, policies.DefaultStoragePolicyRetainForOverridable())
		}
	}

	if d.HasChanges(policyStorageDeleteAfterDaysKey, policyStorageDeleteAfterOverridableKey) {
		if deleteDaysVal, ok := d.GetOk(policyStorageDeleteAfterDaysKey); ok {
			opts = append(opts, policies.WithStoragePolicyDeleteAfterDays(int32(deleteDaysVal.(int))))
		} else {
			opts = append(opts, policies.DefaultStoragePolicyDeleteAfterDays())
		}
		if deleteOverrideVal, ok := d.GetOk(policyStorageDeleteAfterOverridableKey); ok {
			opts = append(opts, policies.WithStoragePolicyDeleteAfterOverridable(deleteOverrideVal.(bool)))
		} else {
			opts = append(opts, policies.DefaultStoragePolicyDeleteAfterOverridable())
		}
	}

	if len(opts) > 0 {
		opts = append(opts, policies.WithAutomaticVersioning(true))
		p, err := pClient.Update(ctx, d.Id(), 0, opts...)
		if err != nil {
			return diag.Errorf("error creating storage policy: %v", err)
		}
		if p == nil {
			return diag.Errorf("nil storage policy after create")
		}

		if err := setFromPolicyItem(d, p.GetItem()); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourcePolicyStorageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	md := meta.(*metaData)
	pClient := policies.NewClient(md.client)

	_, err := pClient.Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error deleting policy: %v", err)
	}

	return nil
}
