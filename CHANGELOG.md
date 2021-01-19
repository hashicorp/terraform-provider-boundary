## 1.0.0 (Unreleased)

We are bumping the version of the boundary terraform provider to 1.0.0 and will release new versions of the provider at its own cadence instead of keeping it in lockstep with Boundary.

### Bug Fixes

* During `terraform apply`, do not update existing user account passwords when the password field is updated in tf file.
  ([Issue](https://github.com/hashicorp/terraform-provider-boundary/issues/71))
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/70))

## 0.1.4 (January 14, 2021)

### New and Improved

Update to `Boundary API 0.0.3` and `Boundary 0.1.4` ([see boundary changelog](https://github.com/hashicorp/boundary/blob/main/CHANGELOG.md#014-20210105))

## 0.1.0 (October 14, 2020)

Initial release!
