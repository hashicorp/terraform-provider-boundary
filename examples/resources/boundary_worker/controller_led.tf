resource "boundary_worker" "controller_led" {
  scope_id    = "global"
  name        = "controller-led-worker-1"
  description = "self managed worker with controller led auth"
}
