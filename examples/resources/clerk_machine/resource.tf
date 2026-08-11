resource "clerk_machine" "worker" {
  name              = "worker"
  default_token_ttl = 3600
}

resource "clerk_machine" "api" {
  name = "api"

  # The api machine can mint tokens for the worker machine.
  scoped_machines = [clerk_machine.worker.id]
}
