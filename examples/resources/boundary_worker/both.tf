resource "boundary_worker" "controller_led" {
  scope_id = "global"
  name     = "controller-led-worker-1"
}

resource "boundary_worker" "worker_led" {
  scope_id                    = "global"
  name                        = "worker-led-worker-1"
  worker_generated_auth_token = var.worker_generated_auth_token
}
