resource "boundary_worker" "controller_led" {
  scope_id    = "global"
  name        = "worker 1"
  description = "self managed worker with controller led auth"
}

resource "boundary_worker" "worker_led" {
  scope_id                    = "global"
  name                        = "worker 2"
  description                 = "self managed worker with worker led auth"
  worker_generated_auth_token = var.worker_generated_auth_token
}
