terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
    aws = {
      source = "hashicorp/aws"
    }
  }
}


data "aws_subnets" "private" {
  tags = {
    Coder_Workspaces = "true"
  }
}

data "coder_parameter" "rocker_image" {
  name = "rocker_image"
  display_name = "Rocker Image"
  description = "see https://rocker-project.org/images/"
  default = "rocker/tidyverse"
  option {
    name = "rstudio"
    value = "rocker/rstudio"
  }
  option {
    name = "tidyverse"
    value = "rocker/tidyverse"
  }
  option {
    name = "geospatial"
    value = "rocker/geospatial"
  }
}

data "coder_parameter" "instance_type" {
  name         = "instance_type"
  display_name = "Instance type"
  description  = "What instance type should your workspace use?"
  default      = "t3.medium"
  mutable      = false
  option {
    name  = "2 vCPU, 4 GiB RAM"
    value = "t3.medium"
  }
  option {
    name  = "2 vCPU, 8 GiB RAM"
    value = "t3.large"
  }
  option {
    name  = "4 vCPU, 16 GiB RAM"
    value = "t3.xlarge"
  }
  option {
    name  = "8 vCPU, 32 GiB RAM"
    value = "t3.2xlarge"
  }
}



data "coder_parameter" "instance_disk" {
  name         = "instance_disk"
  type         = "number"
  display_name = "Instance Disk Size"
  description  = "How much disk space for your workspace?"
  default      = 24
  mutable      = false
  option {
    name  = "24 GiB"
    value = 24
  }
  option {
    name  = "64 GiB"
    value = 64
  }
  option {
    name  = "128 GiB"
    value = 128
  }
}


provider "aws" {
  region = "us-west-2"
}

data "coder_workspace" "me" {
}


resource "coder_agent" "dev" {
  count          = data.coder_workspace.me.start_count
  arch           = "amd64"
  auth           = "aws-instance-identity"
  os             = "linux"
  startup_script = <<-EOT
    set -e
    # run rstudio
    mkdir -p $HOME/workspace
    podman run --rm -d --name rstudio \
        -p 127.0.0.1:8787:8787 \
        -v $HOME/workspace:/root/workspace \
        -v $(echo $GIT_SSH_COMMAND | cut -d" " -f1):/tmp/coder/coder \
        -e DISABLE_AUTH=true \
        -e GIT_SSH_COMMAND='/tmp/coder/coder gitssh --' \
        -e 'CODER*' \
        docker.io/${data.coder_parameter.rocker_image.value}
  EOT

  shutdown_script = "podman stop rstudio"
  
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
  count        = data.coder_workspace.me.start_count
  agent_id     = coder_agent.dev[0].id
  slug         = "rstutio"
  display_name = "RStudio"
  url          = "http://localhost:8787"
  share        = "owner"
  subdomain    = true
  healthcheck {
    url       = "http://localhost:8787"
    interval  = 15
    threshold = 20
  }
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
  hostname: ${lower(data.coder_workspace.me.name)}
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
  apt update && apt install -y podman
  loginctl enable-linger $(id -u ${local.linux_user})
  sudo -u ${local.linux_user} sh -c '${try(coder_agent.dev[0].init_script, "")}'
  --//--
  EOT
}

resource "aws_instance" "dev" {
  ami               = "ami-0cf2b4e024cdb6960" # ubuntu
  availability_zone = "us-west-2a"
  instance_type     = data.coder_parameter.instance_type.value
  subnet_id = tolist(data.aws_subnets.private.ids)[0]
  associate_public_ip_address = false
  user_data = local.user_data
  root_block_device {
    volume_size = tonumber(data.coder_parameter.instance_disk.value)
  }
  tags = {
    Name = "coder-${data.coder_workspace.me.owner}-${data.coder_workspace.me.name}"
    # Required if you are using our example policy, see template README
    Coder_Provisioned = "true"
  }
  lifecycle {
    ignore_changes = [ami]
  }
}

resource "coder_metadata" "workspace_info" {
  resource_id = aws_instance.dev.id
  item {
    key   = "instance type"
    value = aws_instance.dev.instance_type
  }
  item {
    key   = "disk"
    value = "${aws_instance.dev.root_block_device[0].volume_size} GiB"
  }
}

resource "aws_ec2_instance_state" "dev" {
  instance_id = aws_instance.dev.id
  state       = data.coder_workspace.me.transition == "start" ? "running" : "stopped"
}
