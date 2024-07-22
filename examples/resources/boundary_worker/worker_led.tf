resource "boundary_worker" "worker_led" {
  scope_id                    = "global"
  name                        = "worker-led-worker-1"
  description                 = "self managed worker with worker led auth"
  worker_generated_auth_token = var.worker_generated_auth_token
}
