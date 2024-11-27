data "harvester_image" "os_image" {
  display_name = "Ubuntu Minimal 24.04 LTS"
  namespace = "dreamlab"
}

locals {
  hostname   = lower(data.coder_workspace.env.name)
  linux_user = "coder"
}

data "coder_workspace" "env" {}
data "coder_workspace_owner" "me" {}

resource "coder_agent" "dev" {
  count          = data.coder_workspace.env.start_count
  arch           = "amd64"
  auth           = "token"
  os             = "linux"
  startup_script = <<-EOT
    set -e
    # install and start code-server
    curl -fsSL https://code-server.dev/install.sh | sh -s -- --method=standalone --prefix=/tmp/code-server --version 4.11.0
    /tmp/code-server/bin/code-server --auth none --port 13337 >/tmp/code-server.log 2>&1 &
  EOT

  metadata {
    key          = "cpu"
    display_name = "CPU Usage"
    interval     = 5
    timeout      = 5
    script       = "coder stat cpu"
  }
  metadata {
    key          = "memory"
    display_name = "Memory Usage"
    interval     = 5
    timeout      = 5
    script       = "coder stat mem"
  }
  metadata {
    key          = "disk"
    display_name = "Disk Usage"
    interval     = 600 # every 10 minutes
    timeout      = 30  # df can take a while on large filesystems
    script       = "coder stat disk --path $HOME"
  }
}

resource "coder_app" "code-server" {
  count        = data.coder_workspace.env.start_count
  agent_id     = coder_agent.dev[0].id
  slug         = "code-server"
  display_name = "code-server"
  url          = "http://localhost:13337/?folder=/home/coder"
  icon         = "/icon/code.svg"
  subdomain    = false
  share        = "owner"

  healthcheck {
    url       = "http://localhost:13337/healthz"
    interval  = 3
    threshold = 10
  }
}

# harvester requires an ssh key in the vm's userdata. This isn't actually used.
resource "harvester_ssh_key" "key" {
    name      = "coder-${data.coder_workspace_owner.me.name}-${data.coder_workspace.env.name}"
    namespace = "${var.namespace}"
    public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKJ1r1Y/1mC/oyWLxb7fdeRiri3ZtSirJZPkmwEzKpNO"
}

# the coder startup script is too large to be included directly in the vm
# resource userdata, so we need to create a separate resource for it.
resource "harvester_cloudinit_secret" "coder-userdata" {
  name = "coder-${data.coder_workspace_owner.me.name}-${data.coder_workspace.env.name}"
  namespace = var.namespace
  user_data = templatefile("${path.module}/cloud-init/cloud-config.yaml.tftpl", {
        ssh_authorized_key = var.ssh_authorized_key
        linux_user = local.linux_user
        hostname = local.hostname
        startup = base64encode(templatefile("${path.module}/cloud-init/startup.sh.tftpl",{
          init_script = try(coder_agent.dev[0].init_script, "")
          coder_agent_token = try(coder_agent.dev[0].token, "")
          linux_user = local.linux_user
        }))
      })
}

# resource "harvester_volume" "coder-disk" {
#   name = "coder-${data.coder_workspace_owner.me.name}-${data.coder_workspace.env.name}"
#   namespace = var.namespace
#   image = data.harvester_image.os_image.id
#   size = "60Gi"
# }

resource "harvester_virtualmachine" "coder-vm" {
    count = data.coder_workspace.env.start_count
    name        = "coder-${data.coder_workspace_owner.me.name}-${data.coder_workspace.env.name}"
    description =  "coder vm: ${data.coder_workspace_owner.me.name} ${data.coder_workspace.env.name}"
    namespace   = var.namespace
    restart_after_update = true
    cpu    = 16
    memory = "32Gi"
    ssh_keys = [harvester_ssh_key.key.id]
    disk {
        name       = "rootdisk"
        type       = "disk"
        size       = "25Gi"
        bus        = "virtio"
        boot_order = 1
        image       = data.harvester_image.os_image.id
        auto_delete = true
    }
    network_interface {
        name         = "default"
        model        = "virtio"
        type         = "bridge"
        network_name = "default/2176"
    }
    cloudinit {
      user_data_secret_name  = harvester_cloudinit_secret.coder-userdata.name
    }
    lifecycle {
      ignore_changes = [ cloudinit ]
    }
}