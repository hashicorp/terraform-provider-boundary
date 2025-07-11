# Boundary Terraform Provider CHANGELOG

Canonical reference for changes, improvements, and bugfixes for the Boundary Terraform provider.

## 1.3.1 (July 11th, 2025)

### Bug Fix

* Fixes a problem where the KMS plugins were not being bundled in the provider's
  binary ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/717))

## 1.3.0 (July 9th, 2025)

### Known Issues

* `Error configuring kms: plugin is nil`: This error message may occur when you
  attempt to use KMS plugin functionality. The [KMS plugins](plugins/kms/mains)
  were not bundled in this Terraform provider release and thus any functionality
  related to this will not work.

### New and Improved

* Updates various dependencies across the provider
  ([Example PR](https://github.com/hashicorp/terraform-provider-boundary/pull/708))

## 1.2.0 (October 21, 2024)

### New and Improved

* Introduces support for specifying a worker filter in dynamic host catalogs
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/626))

### Deprecations/Changes

* With Boundary 0.15, a deprecation notice was put under the `grant_scope_id`
  field, and a new `grant_scope_ids` field was introduced to replace it. With
  Boundary v0.17.1 and Boundary API v0.0.52, `grant_scope_id` support was
  entirely removed. `grant_scope_id` support has now been removed from this TF
  provider.

## 1.1.15 (May 1, 2024)

### New and Improved

* Add support for a target alias as a resource
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/571))

## 1.1.14 (February 14, 2024)

### New and Improved

* Support the multi-value `grant_scope_ids` field in the role provider
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/562))

* Support Boundary [Storage Policies](https://developer.hashicorp.com/boundary/docs/configuration/session-recording/configure-storage-policy)
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/558))

## 1.1.13 (February 1, 2024)

### New and Improved

* Allow dynamic credentials when configuring storage buckets
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/549))

## 1.1.12 (January 8, 2024)

### New and Improved

* Add support to configure valid_principals with Vault SSH Certificate Credential Library
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/512))

## 1.1.11 (December 13, 2023)

### New and Improved

* Add support for OIDC prompts. Using prompts, the Relying Party (RP) can customize the authentication and authorization flow to suit their specific needs and improve the user experience. [OIDC Authentication request](https://openid.net/specs/openid-connect-core-1_0.html#AuthRequest) server.
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/519))
* Add boundary_auth_method data source
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/505))
* Add boundary_group data source
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/503))
* Add boundary_account data source
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/493))
* Add boundary_user data source
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/468))

### Bug Fix
* Fix boundary_worker overwriting worker generated auth token during
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/461)) 

## 1.1.10 (October 11, 2023)

### New and Improved

* Add support for Scope datasource
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/474))
* LDAP: Add support for maximum_page_size and dereference_aliases
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/453))


## 1.1.9 (July 19, 2023)

### New and Improved

* Add support for a storage bucket as a resource
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/417))
* Add option to enable session recording on a target resource
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/421))
* Update docs for host set plugin filters examples
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/420))  

## 1.1.8 (June 13, 2023)

### New and Improved

* Add support for target default client port
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/379))
* Add support for using ldap primary auth method
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/392))

### Deprecations/Changes

* Deprecate `password_auth_method_login_name` & `password_auth_method_password` for Terraform Provider.
  `password_auth_method_login_name` & `password_auth_method_password` fields have been set to deprecated 
  with a recommendation to use `auth_method_login_name` & `auth_method_password` fields instead.
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/392))
* Deprecate type field for `boundary_account_password`
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/396))
* Deprecate type field for `boundary_account_ldap`
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/400))

## 1.1.7 (May 12, 2023)

### Bug Fix
* Fix default auth method with recovery kms
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/406))  

## 1.1.6 (May 5, 2023)

### New and Improved
* Add support for using default auth method if no auth method ID is provided for provider
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/385))
* Fix typo in Managed Group resource page
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/370))

### Bug Fix
* Force new resource on credential_type change
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/389))

## 1.1.5 (April 21, 2023)

### New and Improved
* Add support for credential store vault worker filters ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/375))

### Bug Fix
* Allow users to set OIDC maxAge value to 0 to require immediate reauth ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/364))

## 1.1.4 (February 15, 2023)

### New and Improved

* Add support for worker egress and ingress filters
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/319))
* Add support for vault ssh certificate credential libraries
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/320))
* Add support for targets with address configurations
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/308))

## 1.1.3 (November 29, 2022)

### New and Improved

* Add support for a workers as a resource
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/293)).

## 1.1.2 (October 18, 2022)

### New and Improved

* Add support for setting mapping overrides for vault credential libraries
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/287)).

### Bug Fixes

* Improve error message when authenticating to boundary
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/290)).
* Set state before returning an error when creating a resource
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/289))

## 1.1.1 (October 12, 2022)

### Bug Fixes

* The plugin cleanup function is being called before the entire Terraform workflow is complete.
  ([Issue](https://github.com/hashicorp/terraform-provider-boundary/issues/282)),
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/285)).

## 1.1.0 (October 4, 2022)

### New and Improved

* Add support for JSON credentials
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/271)).
* Add support for setting the plugin execution directory from the config
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/280)).

### Deprecations/Changes

* Fix panic resulting from expired Vault credential store tokens
  ([Issue](https://github.com/hashicorp/terraform-provider-boundary/issues/279),
  [PR](https://github.com/hashicorp/terraform-provider-boundary/pull/277)).
* Remove `application_credential_source_ids` of the `target` resource which was deprecated
  in 1.0.12 ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/273)).
* Remove `default_role` from the `role` resource, this schema was never supported and was
  included mistakenly ([Issue](https://github.com/hashicorp/terraform-provider-boundary/issues/130),
  [PR](https://github.com/hashicorp/terraform-provider-boundary/pull/269)).

## 1.0.12 (September 13, 2022)

### New and Improved

* Add support for SSH targets
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/264)).

### Deprecations/Changes

* Deprecate `application_credential_source_ids` of the `target` resource
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/260)).

## 1.0.11 (August 26, 2022)

### New and Improved

* Add support for SSH private key credentials
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/257)).
* Add support for credential type in Vault libraries
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/257)).

## 1.0.10 (August 10, 2022)

### New and Improved

* Adds support for static credential stores
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/236)).
* Adds support for username password credentials 
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/242)).

## 1.0.9 (June 6, 2022)

### Bug Fixes

* The bug fix released in 1.0.8 to resolve the `plugin is nil` error only worked for 
  Linux AMD64. This was due to a build issue where the plugin binaries were only built for 
  Linux AMD64. Other platforms would receive an error similar to:

            Error: error reading wrappers from "recovery_kms_hcl":
            Error configuring kms: error fetching kms plugin rpc client: 
            fork/exec boundary-plugin-kms-awskms.gz: exec format error
  ([Issue](https://github.com/hashicorp/terraform-provider-boundary/issues/209)),
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/216)).

## 1.0.8 (June 1, 2022)

### Bug Fixes

* After moving to go-kms-wrapping V2, the Boundary Terraform Provider
  did not load all KMS plugins resulting in an error when trying to
  create a wrapper for any type other than 'aead':

            Error: error reading wrappers from "recovery_kms_hcl":
            Error configuring kms: plugin is nil
  ([Issue](https://github.com/hashicorp/terraform-provider-boundary/issues/209)),
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/213)).

## 1.0.7 (May 16, 2022)

### Deprecations/Changes

* Undoes an erroneous deprecation of the `login_name` and `password` fields in `resource_account_password` and `resource_account`. 
  Deprecates `resource_account` that was replaced with `resource_account_password`
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/201)).

## 1.0.6 (January 21, 2022)

### New and Improved

* Adds dynamic host plugin catalog/set
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/159)).
* Adds support for insecure TLS to Boundary 
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/163)).

 ### Deprecations/Changes

* Removes fields `host_set_ids` and `application_credential_library_ids` of the 
  `target` resource, which were deprecated in 1.0.5 
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/150)).

## 1.0.5 (September 09, 2021)

### Deprecations/Changes

* Deprecate fields `host_set_ids` and `application_credential_library_ids` of the 
  `target` resource. See boundary 0.5.0 [changelog](https://github.com/hashicorp/boundary/blob/main/CHANGELOG.md#deprecationschanges) for more detail on the deprecation.
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/134)).

## 1.0.4 (August 19, 2021)

### New and Improved

* Adds managed groups resource
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/118)).

## 1.0.3 (June 30, 2021)

### New and Improved

* Adds credential library resource for Vault
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/114)).
* Adds credential store resource for Vault
  ([PR 1](https://github.com/hashicorp/terraform-provider-boundary/pull/114)),
  ([PR 2](https://github.com/hashicorp/terraform-provider-boundary/pull/125)).
* Adds claim scopes attribute to OIDC auth method
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/112)).
* Adds account claim maps attribute to OIDC auth method
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/111)).

### Bug Fixes

* Make OIDC account attribute for subject ForceNew
  ([Issue](https://github.com/hashicorp/terraform-provider-boundary/issues/119)),
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/122)).
* Update static type attribute for host catalog resource
  ([Issue](https://github.com/hashicorp/terraform-provider-boundary/issues/115)),
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/121)).

## 1.0.2 (May 06, 2021)

### New and Improved

* Adds OIDC account resource
([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/105)).
* Adds OIDC auth method resource
([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/105)).

### Deprecations/Changes

* Deprecates fields on `resource_auth_method` that will be replaced in the future with generic `attributes` attribute.

## 1.0.1 (February 02, 2021)

### New and Improved

* Adds worker filter to target resource
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/76)).

## 1.0.0 (January 20, 2021)

We are bumping the version of the Boundary Terraform provider to v1.0.0 and will release new versions of the provider at its own cadence instead of keeping it in lockstep with Boundary.

### Bug Fixes

* During `terraform apply`, do not update existing user account passwords when the password field is updated in the tf file.
  ([Issue](https://github.com/hashicorp/terraform-provider-boundary/issues/71)),
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/70)).

## 0.1.4 (January 14, 2021)

### New and Improved

Update provider to handle new domain errors ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/63)).

## 0.1.0 (October 14, 2020)

Initial release!
