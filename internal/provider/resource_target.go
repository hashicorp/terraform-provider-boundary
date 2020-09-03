package provider

import (
	"fmt"

	"github.com/hashicorp/boundary/api/scopes"
	"github.com/hashicorp/boundary/api/targets"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	targetNameKey        = "name"
	targetDescriptionKey = "description"
	targetScopeIDKey     = "scope_id"
	targetHostSetIDsKey  = "host_set_ids"
	targetProtoKey       = "proto"
	targetDefaultPortKey = "default_port"

	targetProtoTCP = "tcp"
)

func resourceTarget() *schema.Resource {
	return &schema.Resource{
		Create: resourceTargetCreate,
		Read:   resourceTargetRead,
		Update: resourceTargetUpdate,
		Delete: resourceTargetDelete,
		Schema: map[string]*schema.Schema{
			targetNameKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			targetDescriptionKey: {
				Type:     schema.TypeString,
				Optional: true,
			},
			targetProtoKey: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			targetScopeIDKey: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			targetHostSetIDsKey: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

// convertTargetToResourceData creates a ResourceData type from a Group
func convertTargetToResourceData(t *targets.Target, d *schema.ResourceData) error {
	if t.Name != "" {
		if err := d.Set(targetNameKey, t.Name); err != nil {
			return err
		}
	}

	if t.Description != "" {
		if err := d.Set(targetDescriptionKey, t.Description); err != nil {
			return err
		}
	}

	if t.Scope != nil && t.Scope.Id != "" {
		if err := d.Set(targetScopeIDKey, t.Scope.Id); err != nil {
			return err
		}
	}

	if t.HostSetIds != nil {
		if err := d.Set(targetHostSetIDsKey, t.HostSetIds); err != nil {
			return err
		}
	}

	d.SetId(t.Id)

	return nil
}

// convertResourceDataToTarget returns a localy built Group using the values provided in the ResourceData.
func convertResourceDataToTarget(d *schema.ResourceData, meta *metaData) (*targets.Target, string) {
	t := &targets.Target{Scope: &scopes.ScopeInfo{}}
	proto := targetProtoTCP

	if descVal, ok := d.GetOk(targetDescriptionKey); ok {
		t.Description = descVal.(string)
	}

	if nameVal, ok := d.GetOk(targetNameKey); ok {
		t.Name = nameVal.(string)
	}

	if scopeIDVal, ok := d.GetOk(targetScopeIDKey); ok {
		t.Scope.Id = scopeIDVal.(string)
	}

	if protoVal, ok := d.GetOk(targetProtoKey); ok {
		proto = protoVal.(string)
	}

	if val, ok := d.GetOk(targetHostSetIDsKey); ok {
		hostSetIds := val.(*schema.Set).List()
		for _, i := range hostSetIds {
			t.HostSetIds = append(t.HostSetIds, i.(string))
		}
	}

	if d.Id() != "" {
		t.Id = d.Id()
	}

	return t, proto
}

func resourceTargetCreate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	t, proto := convertResourceDataToTarget(d, md)
	tgts := targets.NewClient(client)

	hostSetIDs := t.HostSetIds

	t, apiErr, err := tgts.Create(
		ctx,
		proto,
		t.Scope.Id,
		targets.WithName(t.Name),
		targets.WithDescription(t.Description))
	if err != nil {
		return fmt.Errorf("error creating target: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error creating target: %s", apiErr.Message)
	}

	if len(hostSetIDs) > 0 {
		t, apiErr, err = tgts.SetHostSets(
			ctx,
			t.Id,
			t.Version,
			hostSetIDs,
			targets.WithScopeId(t.Scope.Id))
		if apiErr != nil {
			return fmt.Errorf("error setting host sets on target: %s\n", apiErr.Message)
		}
		if err != nil {
			return fmt.Errorf("error setting host sets on target: %s\n", err)
		}
	}

	return convertTargetToResourceData(t, d)
}

func resourceTargetRead(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	t, _ := convertResourceDataToTarget(d, md)
	tgts := targets.NewClient(client)

	t, apiErr, err := tgts.Read(ctx, t.Id, targets.WithScopeId(t.Scope.Id))
	if err != nil {
		return fmt.Errorf("error reading target: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error reading target: %s", apiErr.Message)
	}

	return convertTargetToResourceData(t, d)
}

func resourceTargetUpdate(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	t, _ := convertResourceDataToTarget(d, md)
	tgts := targets.NewClient(client)

	if d.HasChange(targetNameKey) {
		t.Name = d.Get(targetNameKey).(string)
	}

	if d.HasChange(targetDescriptionKey) {
		t.Description = d.Get(targetDescriptionKey).(string)
	}

	t.Scope.Id = d.Get(targetScopeIDKey).(string)

	t, apiErr, err := tgts.Update(
		ctx,
		t.Id,
		0,
		targets.WithScopeId(t.Scope.Id),
		targets.WithAutomaticVersioning(),
		targets.WithName(t.Name),
		targets.WithDescription(t.Description))
	if err != nil {
		return err
	}
	if apiErr != nil {
		return fmt.Errorf("%+v\n", apiErr.Message)
	}

	if d.HasChange(targetHostSetIDsKey) {
		hostSetIds := []string{}
		hostSets := d.Get(targetHostSetIDsKey).(*schema.Set).List()
		for _, hostSet := range hostSets {
			hostSetIds = append(hostSetIds, hostSet.(string))
		}

		t, apiErr, err = tgts.SetHostSets(
			ctx,
			t.Id,
			t.Version,
			hostSetIds,
			targets.WithScopeId(t.Scope.Id))
		if apiErr != nil || err != nil {
			return fmt.Errorf("error updating hostSets on target:\n  API Err: %+v\n  Err: %+v\n", *apiErr, err)
		}
	}

	return convertTargetToResourceData(t, d)
}

func resourceTargetDelete(d *schema.ResourceData, meta interface{}) error {
	md := meta.(*metaData)
	client := md.client
	ctx := md.ctx

	t, _ := convertResourceDataToTarget(d, md)
	tgts := targets.NewClient(client)

	_, apiErr, err := tgts.Delete(ctx, t.Id, targets.WithScopeId(t.Scope.Id))
	if err != nil {
		return fmt.Errorf("error deleting target: %s", err.Error())
	}
	if apiErr != nil {
		return fmt.Errorf("error deleting target: %s", apiErr.Message)
	}

	return nil
}
