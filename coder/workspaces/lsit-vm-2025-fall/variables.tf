
variable "namespace" {
  # harvester namespace
  default = "dreamlab"
}

variable ssh_authorized_key {
  # harvester requires an ssh key, but 
  # it's not actually used to access the vm.
  default = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKJ1r1Y/1mC/oyWLxb7fdeRiri3ZtSirJZPkmwEzKpNO"
}