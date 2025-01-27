data "harvester_image" "os_image" {
  display_name = "Ubuntu Minimal 24.04 LTS"
  namespace = var.namespace
}

locals {
  hostname   = lower(data.coder_workspace.env.name)
  linux_user = "coder"
  # names used for coder's resources in harvester: needs to be valid for
  # kubernetes IDs.
  k8s_name = lower(replace("coder-${data.coder_workspace_owner.me.name}-${data.coder_workspace.env.name}"," ","_"))
  workspace_agreement = "https://coder.dreamlab.ucsb.edu/templates/lsit-vm/docs"
}


data "coder_parameter" "agreement" {
  name = "user_agreement"
  display_name = "User agreement"
  type = "string"
  mutable = true
  description = <<-EOT
    Please confirm that you have read and understand the [usage policies for the workspace](${local.workspace_agreement}).
    EOT

  option {
    name = "I understand the usage policies."
    value = "agree"
  }

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

  display_apps {
    vscode          = true
    vscode_insiders = false
    web_terminal    = true
    ssh_helper      = false
  }

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

resource "coder_app" "rstudio" {
  count        = data.coder_workspace.env.start_count
  agent_id     = coder_agent.dev[0].id
  slug         = "rstudio"
  display_name = "RStudio"
  url          = "http://localhost:8787"
  icon         = "/icon/rstudio.svg"
  subdomain    = true
  share        = "owner"
  order        = 1
  healthcheck {
    url       = "http://localhost:8787/health-check"
    interval  = 4
    threshold = 40
  }
}

module "filebrowser" {
  count    = data.coder_workspace.env.start_count
  source   = "registry.coder.com/modules/filebrowser/coder"
  version  = "1.0.23"
  order        = 2
  agent_id = coder_agent.dev[0].id
  database_path = ".config/filebrowser.db"
}

module "vscode-web" {
  count = data.coder_workspace.env.start_count
  source = "registry.coder.com/modules/vscode-web/coder"
  version = "1.0.26"
  order = 3
  agent_id = coder_agent.dev[0].id
  accept_license = true
  folder = "/home/coder"
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

# operating system disk
resource "harvester_volume" "coder-disk" {
  name = local.k8s_name
  namespace = var.namespace
  image = data.harvester_image.os_image.id
  size = "20Gi"
}

# user data disk
resource "harvester_volume" "user-disk" {
  name = "${local.k8s_name}-userdata"
  namespace = var.namespace
  storage_class_name = "csi-rbd-sc"
  size = "100Gi"
}

resource "harvester_virtualmachine" "coder-vm" {
    name = local.k8s_name
    namespace = var.namespace
    description = "coder vm: ${data.coder_workspace_owner.me.name} ${data.coder_workspace.env.name}"
    run_strategy = data.coder_workspace.env.transition == "start" ? "RerunOnFailure" : "Halted"
    cpu = 4
    memory = "16Gi"
    disk {
        name       = "rootdisk"
        type       = "disk"
        bus        = "scsi"
        boot_order = 1
        existing_volume_name = harvester_volume.coder-disk.name
        auto_delete          = false
    }
    disk {
        name       = "user-disk"
        type       = "disk"
        bus        = "virtio"
        existing_volume_name = harvester_volume.user-disk.name
        auto_delete = false
    }
    network_interface {
        # management network is default
        name         = "default"
        model        = "virtio"
        network_name = "harvester-public/1173"
        type = "bridge"        
    }
    ssh_keys = [harvester_ssh_key.key.id]
    cloudinit {
      user_data_secret_name  = harvester_cloudinit_secret.coder-userdata.name
    }
    lifecycle {
      ignore_changes = [ cloudinit ]
    }
}