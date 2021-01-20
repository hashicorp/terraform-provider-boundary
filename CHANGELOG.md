## 1.0.0 (Unreleased)

We are bumping the version of the Boundary Terraform provider to v1.0.0 and will release new versions of the provider at its own cadence instead of keeping it in lockstep with Boundary.

### Bug Fixes

* During `terraform apply`, do not update existing user account passwords when the password field is updated in the tf file.
  ([Issue](https://github.com/hashicorp/terraform-provider-boundary/issues/71))
  ([PR](https://github.com/hashicorp/terraform-provider-boundary/pull/70))

## 0.1.4 (January 14, 2021)

### New and Improved

Update to the Boundary Go SDK v0.0.3

## 0.1.0 (October 14, 2020)

Initial release!
