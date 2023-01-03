# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "boundary_worker" "controller_led" {
  scope_id                    = "global"
  name                        = "worker 1"
  description                 = "self managed worker with controlled led auth"
  worker_generated_auth_token = var.worker_generated_auth_token
}

resource "boundary_self_managed_worker" "worker_led" {
  scope_id                    = "global"
  name                        = "worker 2"
  description                 = "self managed worker with controlled led auth"
  worker_generated_auth_token = var.worker_generated_auth_token
}