## 1.0.2 (Unreleased)

* Adds OIDC account resource
([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/105))
* Adds OIDC auth method resource
([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/105))
* Deprecates fields on `resource_auth_method` that will be replaced in the future with generic `attributes` attribute.

## 1.0.1 (February 02, 2021)

### New and Improved

* Adds worker filter to target resource
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/76))

## 1.0.0 (January 20, 2021)

We are bumping the version of the Boundary Terraform provider to v1.0.0 and will release new versions of the provider at its own cadence instead of keeping it in lockstep with Boundary.

### Bug Fixes

* During `terraform apply`, do not update existing user account passwords when the password field is updated in the tf file.
  ([Issue](https://github.com/hashicorp/terraform-provider-boundary/issues/71))
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/70))

## 0.1.4 (January 14, 2021)

### New and Improved

Update provider to handle new domain errors ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/63))

## 0.1.0 (October 14, 2020)

Initial release!
