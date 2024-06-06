// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func apiErr(err *api.Error) diag.Diagnostics {
	detail := err.Message
	if err.Details != nil {
		var details []string
		for _, field := range err.Details.RequestFields {
			details = append(details, fmt.Sprintf("%s: %s", field.Name, field.Description))
		}
		detail = strings.Join(details, "\n")
	}

	return diag.Diagnostics{
		{
			Severity: diag.Error,
			Summary:  "An error occured while querying the API",
			Detail:   detail,
		},
	}
}

func set(schema map[string]*schema.Schema, d *schema.ResourceData, val map[string]interface{}) error {
	for k, v := range val {
		sch := schema[k]
		if sch == nil {
			continue
		}
		v = convert(sch, v)
		if err := d.Set(k, v); err != nil {
			return fmt.Errorf("failed to set '%s': %w", k, err)
		}
	}

	return nil
}

func convertResource(sch *schema.Resource, val map[string]interface{}) map[string]interface{} {
	res := map[string]interface{}{}
	for k, s := range sch.Schema {
		res[k] = convert(s, val[k])
	}
	return res
}

func convert(sch *schema.Schema, val interface{}) interface{} {
	switch ty := sch.Type; ty {
	case schema.TypeBool, schema.TypeInt, schema.TypeFloat, schema.TypeString:
		return val
	case schema.TypeList:
		switch val := val.(type) {
		case nil:
			return []interface{}{}
		case []interface{}:
			res := []interface{}{}
			for _, v := range val {
				if s, ok := sch.Elem.(*schema.Schema); ok {
					res = append(res, convert(s, v))
				} else {
					res = append(res, convertResource(sch.Elem.(*schema.Resource), v.(map[string]interface{})))
				}
			}
			return res
		case map[string]interface{}:
			// terraform-plugin-sdk does not know how to have an object in an
			// object so we use a list with one element
			if s, ok := sch.Elem.(*schema.Schema); ok {
				return []interface{}{
					convert(s, val),
				}
			} else {
				return []interface{}{
					convertResource(sch.Elem.(*schema.Resource), val),
				}
			}
		default:
			panic(fmt.Sprintf("unknown list type %T", val))
		}
	default:
		panic(fmt.Sprintf("unknown type %s", ty))
	}
}
