resource "harvester_ssh_key" "key" {
    name      = "test-key"
    namespace = "${var.namespace}"
    public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKJ1r1Y/1mC/oyWLxb7fdeRiri3ZtSirJZPkmwEzKpNO"
}

data "harvester_image" "os_image" {
  display_name   = "Ubuntu Minimal 24.04 LTS"
  namespace = "dreamlab"
}

locals {
  linux_user = "ubuntu"
  hostname = "dreamer"
}


# resource "harvester_virtualmachine" "dream01" {
#     name        = "dream01"
#     description = "test machine"
#     namespace   = var.namespace
#     restart_after_update = true
#     cpu    = 4
#     memory = "8Gi"
#     ssh_keys = [harvester_ssh_key.key.id]
#     disk {
#         name       = "rootdisk"
#         type       = "disk"
#         size       = "25Gi"
#         bus        = "virtio"
#         boot_order = 1
#         image       = data.harvester_image.os_image.id
#         auto_delete = true
#     }
#     network_interface {
#         name         = "default"
#         model        = "virtio"
#         type         = "bridge"
#         network_name = "default/2176"
#     }
#     cloudinit {
#       user_data = templatefile("${path.module}/cloud-init/cloud-config.yaml.tftpl", {
#         ssh_authorized_key = var.ssh_authorized_key
#         startup = base64encode(templatefile("${path.module}/cloud-init/startup.sh.tftpl",{
#           # here = "here"
#         }))
#       })
#     }
# }