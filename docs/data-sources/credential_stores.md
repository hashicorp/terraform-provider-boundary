---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "boundary_credential_stores Data Source - terraform-provider-boundary"
subcategory: ""
description: |-
  Lists all Credential Stores.
---

# boundary_credential_stores (Data Source)

Lists all Credential Stores.



<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- **filter** (String)
- **id** (String) The ID of this resource.
- **recursive** (Boolean)
- **scope_id** (String)

### Read-Only

- **items** (List of Object) (see [below for nested schema](#nestedatt--items))

<a id="nestedatt--items"></a>
### Nested Schema for `items`

Read-Only:

- **authorized_actions** (List of String)
- **created_time** (String)
- **description** (String)
- **id** (String)
- **name** (String)
- **scope** (List of Object) (see [below for nested schema](#nestedobjatt--items--scope))
- **scope_id** (String)
- **type** (String)
- **updated_time** (String)
- **version** (Number)

<a id="nestedobjatt--items--scope"></a>
### Nested Schema for `items.scope`

Read-Only:

- **description** (String)
- **id** (String)
- **name** (String)
- **parent_scope_id** (String)
- **type** (String)

