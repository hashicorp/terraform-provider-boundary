---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "boundary_group Data Source - terraform-provider-boundary"
subcategory: ""
description: |-
  The boundary_group data source allows you to find a Boundary group.
---

# boundary_group (Data Source)

The boundary_group data source allows you to find a Boundary group.

## Example Usage

```terraform
# Retrieve a user from the global scope
data "boundary_group" "global_group" {
  name = "admin"
}

# User from an org scope
data "boundary_scope" "org" {
  name     = "org"
  scope_id = "global"
}

data "boundary_group" "org_group" {
  name     = "username"
  scope_id = data.boundary_scope.org.id
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the group to retrieve.

### Optional

- `scope_id` (String) The scope ID in which the resource is created. Defaults `global` if unset.

### Read-Only

- `description` (String) The description of the retrieved group.
- `id` (String) The ID of the retrieved group.
- `member_ids` (Set of String) Resource IDs for group members, these are most likely boundary users.
- `scope` (List of Object) (see [below for nested schema](#nestedatt--scope))

<a id="nestedatt--scope"></a>
### Nested Schema for `scope`

Read-Only:

- `description` (String)
- `id` (String)
- `name` (String)
- `parent_scope_id` (String)
- `type` (String)
