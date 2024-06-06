terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
    openstack = {
      source  = "terraform-provider-openstack/openstack"
      version = "~> 1.53.0"
    }
  }
}

provider "openstack" {}


data "coder_parameter" "instance_type" {
  name         = "instance_type"
  display_name = "Instance type"
  description  = "What instance type should your workspace use?"
  default      = 3
  mutable      = false
  option {
    name  = "m3.tiny: 1 CPU, 3GB ram, 20GB disk"
    value = 1
  }
  option {
    name  = "m3.small: 2 CPU, 6GB RAM, 20GB DISK"
    value = 2
  }
  option {
    name  = "m3.quad: 4 CPU, 15GB RAM, 20GB DISK"
    value = 3
  }
  option {
    name  = "m3.medium: 8 CPU, 30GB RAM, 60GB disk"
    value = 4
  }
  option {
    name  = "m3.large: 16 CPU, 60GB RAM, 60GB disk"
    value = 5
  }
}

data "openstack_networking_network_v2" "terraform" {
  name = "terraform_network"
}

locals {
  linux_user = "coder"
  user_data  = <<-EOT
  Content-Type: multipart/mixed; boundary="//"
  MIME-Version: 1.0

  --//
  Content-Type: text/cloud-config; charset="us-ascii"
  MIME-Version: 1.0
  Content-Transfer-Encoding: 7bit
  Content-Disposition: attachment; filename="cloud-config.txt"

  #cloud-config
  cloud_final_modules:
  - [scripts-user, always]
  hostname: ${lower(data.coder_workspace.env.name)}
  users:
  - name: ${local.linux_user}
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash

  --//
  Content-Type: text/x-shellscript; charset="us-ascii"
  MIME-Version: 1.0
  Content-Transfer-Encoding: 7bit
  Content-Disposition: attachment; filename="userdata.txt"

  #!/bin/bash
  
  # packages we need
  apt-get update
  apt-get install -y jq

  # Install Docker
  if ! command -v docker &> /dev/null
  then
    echo "Docker not found, installing..."
    curl -fsSL https://get.docker.com -o get-docker.sh && sh get-docker.sh 2>&1 >/dev/null
    usermod -aG docker ${local.linux_user}
    newgrp docker
  else
    echo "Docker is already installed."
  fi
  
  # Grabs token via the internal metadata server. This IP address is the same for all instances, no need to change it
  # https://docs.openstack.org/nova/rocky/user/metadata-service.html
  export CODER_AGENT_TOKEN=$(curl -s http://169.254.169.254/openstack/latest/meta_data.json | jq -r .meta.coder_agent_token)

  # Run coder agent
  sudo --preserve-env=CODER_AGENT_TOKEN -u ${local.linux_user} sh -c 'export  && ${try(coder_agent.dev[0].init_script, "")}'
  --//--
  EOT
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


# creating Ubuntu22 instance
resource "openstack_compute_instance_v2" "vm" {
  name ="coder-${data.coder_workspace_owner.me.name}-${data.coder_workspace.env.name}"
  image_id  = "77df35aa-bfcb-433c-b333-4e4f2ccf0cc2"
  flavor_id = data.coder_parameter.instance_type.value
  security_groups   = ["default"]
  metadata = {
    coder_agent_token = try(coder_agent.dev[0].token, "")
  }
  user_data = local.user_data
  network {
    name = data.openstack_networking_network_v2.terraform.name
  }
  lifecycle {
    ignore_changes = [ user_data ]
  }
  power_state = data.coder_workspace.env.transition == "start" ? "active" : "shutoff"
  tags = ["Name=coder-${data.coder_workspace_owner.me.name}-${data.coder_workspace.env.name}", "Coder_Provisioned=true"]  
}

# resource "coder_metadata" "workspace_info" {
#   resource_id = aws_instance.dev.id
#   item {
#     key   = "region"
#     value = local.aws_region
#   }
#   item {
#     key   = "instance type"
#     value = aws_instance.dev.instance_type
#   }
#   item {
#     key   = "disk"
#     value = "${aws_instance.dev.root_block_device[0].volume_size} GiB"
#   }
# }
