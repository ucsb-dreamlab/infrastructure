data "harvester_image" "os_image" {
  display_name = "almalinux-9-genericcloud-9.5-20241120.x86_64"
  namespace = var.namespace
}

locals {
  hostname   = lower(data.coder_workspace.env.name)
  linux_user = "coder"
  # names used for coder's resources in harvester: needs to be valid for
  # kubernetes IDs.
  k8s_name = lower(replace("coder-${data.coder_workspace_owner.me.name}-${data.coder_workspace.env.name}"," ","_"))
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
    name      = local.k8s_name
    namespace = var.namespace
    public_key = var.ssh_authorized_key
}

# Coder configuration is attached to the vm using cloud init. Keep in mind:
# harverster uses NoCloud data provider; it only allows yaml-based cloud config
# in user data. Multipart is not supported. 
resource "harvester_cloudinit_secret" "coder-userdata" {
  name = local.k8s_name
  namespace = var.namespace
  user_data = templatefile("${path.module}/cloud-init/cloud-config.yaml.tftpl", {
    ssh_authorized_key = var.ssh_authorized_key
    coder_agent_token = try(coder_agent.dev[0].token, "")
    linux_user = local.linux_user
    hostname = local.hostname
    init_script_b64 = base64encode(try(coder_agent.dev[0].init_script, ""))
  })
}

resource "harvester_volume" "coder-disk" {
  name = local.k8s_name
  namespace = var.namespace
  image = data.harvester_image.os_image.id
  size = "60Gi"
}

resource "harvester_virtualmachine" "coder-vm" {
    name = local.k8s_name
    namespace = var.namespace
    description = "coder vm: ${data.coder_workspace_owner.me.name} ${data.coder_workspace.env.name}"
    run_strategy = data.coder_workspace.env.transition == "start" ? "RerunOnFailure" : "Halted"
    cpu = 8
    memory = "16Gi"
    disk {
        name       = "rootdisk"
        type       = "disk"
        bus        = "scsi"
        boot_order = 1
        existing_volume_name = harvester_volume.coder-disk.name
        auto_delete          = false
    }
    network_interface {
        name         = "default"
        model        = "virtio"
        type         = "bridge"
        # using public network as it's faster
        network_name = "default/2176"
    }
    ssh_keys = [harvester_ssh_key.key.id]
    cloudinit {
      user_data_secret_name  = harvester_cloudinit_secret.coder-userdata.name
    }
    lifecycle {
      ignore_changes = [ cloudinit ]
    }
}