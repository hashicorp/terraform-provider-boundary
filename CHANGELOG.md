# Boundary Terraform Provider CHANGELOG

Canonical reference for changes, improvements, and bugfixes for the Boundary Terraform provider.

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
