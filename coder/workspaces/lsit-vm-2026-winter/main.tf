data "harvester_image" "os_image" {
  display_name = "coder-rstudio-docker-20251218"
  namespace = var.namespace
}

locals {
  hostname   = lower(data.coder_workspace.env.name)
  linux_user = "coder"
  # names used for coder's resources in harvester: needs to be valid for
  # kubernetes IDs.
  k8s_name = lower(replace("coder-${data.coder_workspace_owner.me.name}-${data.coder_workspace.env.name}"," ","_"))
  workspace_agreement = "https://ucsb-dreamlab.github.io/coder-docs/#policies"
}


data "coder_parameter" "agreement" {
  name = "user_agreement"
  display_name = "User Agreement"
  order = 0
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

data "coder_parameter" "rstudio_stack" {
  name = "rstudio_stack"
  display_name = "Include RStudio"
  order = 5
  type = "string"
  mutable = true
  description = "Install and enable RStudio Server in the workspace?"
  default = "false"
  option {
    name = "Yes"
    value = "true"
  }
  option {
    name = "No"
    value = "false"
  }
}


data "coder_parameter" "jupyterlab_stack" {
  name = "jupyterlab_stack"
  display_name = "Include JupyterLab"
  order = 10
  type = "string"
  mutable = true
  description = "Install and enable JupyterLab in the workspace?"
  default = "false"
  option {
    name = "Yes"
    value = "true"
  }
  option {
    name = "No"
    value = "false"
  }
}

data "coder_parameter" "vscode_stack" {
  name = "vscode_stack"
  display_name = "Include Visual Studio Code (Desktop)"
  order = 15
  type = "string"
  mutable = true
  description = "Install and enable the Visual Studio Code service? Requires [Visual Studio Code](https://code.visualstudio.com/) on your personal computer."
  default = "false"
  option {
    name = "Yes"
    value = "true"
  }
  option {
    name = "No"
    value = "false"
  }
}

data "coder_parameter" "vscode_web_stack" {
  name = "vscode_web_stack"
  display_name = "Include Visual Studio Code (Web)"
  order = 20
  type = "string"
  mutable = true
  description = "Install and enable the browser-based Visual Studio Code service?"
  default = "false"
  option {
    name = "Yes"
    value = "true"
  }
  option {
    name = "No"
    value = "false"
  }
}



data "coder_workspace" "env" {}
data "coder_workspace_owner" "me" {}

resource "coder_agent" "workspace" {
  count          = data.coder_workspace.env.start_count
  arch           = "amd64"
  auth           = "token"
  os             = "linux"
  
  startup_script = <<-EOT
    # install pixi
    curl -fsSL https://pixi.sh/install.sh | sh

    # install uv 
    curl -LsSf https://astral.sh/uv/install.sh | sh
    source $HOME/.local/bin/env
  EOT

  display_apps {
    vscode          = data.coder_parameter.vscode_stack.value == "true" ? true : false
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

# enabled if vscode-web is selected
module "vscode-web" {
  count = data.coder_parameter.vscode_web_stack.value == "true" ? data.coder_workspace.env.start_count : 0
  source = "registry.coder.com/modules/vscode-web/coder"
  version = "1.4.1"
  order = 1
  agent_id = coder_agent.workspace[0].id
  accept_license = true
  folder = "/home/coder"
}

# only enabled if rstudio was selected
module "rstudio" {
  source =  "./rstudio"
  count = data.coder_parameter.rstudio_stack.value == "true" ? data.coder_workspace.env.start_count : 0
  agent_id = coder_agent.workspace[0].id
  rserver_user = local.linux_user
  order = 2
}

# only enabled if jupyterlab was selected
module "jupyterlab" {
  source = "./jupyterlab"
  count = data.coder_parameter.jupyterlab_stack.value == "true" ? data.coder_workspace.env.start_count : 0
  agent_id = coder_agent.workspace[0].id
  order = 3
}

module "filebrowser" {
  count    = data.coder_workspace.env.start_count
  source   = "registry.coder.com/modules/filebrowser/coder"
  version  = "1.1.3"
  order    = 4
  agent_id = coder_agent.workspace[0].id
  database_path = ".config/filebrowser.db"
}

resource "coder_app" "docs" {
  count = data.coder_workspace.env.start_count
  slug = "docs"
  agent_id = coder_agent.workspace[0].id
  display_name = "Help"
  open_in = "tab"
  order = 5
  url = "https://ucsb-dreamlab.github.io/coder-docs"
  external = true
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
    coder_agent_token = try(coder_agent.workspace[0].token, "")
    linux_user = local.linux_user
    hostname = local.hostname
    init_script_b64 = base64encode(try(coder_agent.workspace[0].init_script, ""))
  })
}

# operating system disk
resource "harvester_volume" "coder-disk" {
  name = local.k8s_name
  namespace = var.namespace
  image = data.harvester_image.os_image.id
  size = "15Gi"
}

# user data disk
resource "harvester_volume" "user-disk" {
  name = "${local.k8s_name}-userdata"
  namespace = var.namespace
  storage_class_name = "csi-rbd-sc"
  size = "64Gi"
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
        network_name = "harvester-public/1173"        # private
        # network_name = "harvester-public/2176"        # public
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