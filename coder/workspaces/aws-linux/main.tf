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

locals {
  hostname   = lower(data.coder_workspace.dev.name)
  aws_region = "us-west-2"
  linux_user = "coder"
}

provider "aws" {
  region = local.aws_region
}

data "aws_subnets" "private" {
  tags = {
    "dreamlab:service:coder" = "workers"
  }
}

data "coder_parameter" "instance_type" {
  name         = "instance_type"
  display_name = "Instance type"
  description  = "What instance type should your workspace use?"
  default      = "t3.micro"
  mutable      = false
  option {
    name  = "2 vCPU, 1 GiB RAM"
    value = "t3.micro"
  }
  option {
    name  = "2 vCPU, 2 GiB RAM"
    value = "t3.small"
  }
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

data "coder_workspace" "dev" {}

data "coder_workspace_owner" "me" {}

resource "coder_agent" "dev" {
  count          = data.coder_workspace.dev.start_count
  arch           = "amd64"
  auth           = "aws-instance-identity"
  os             = "linux"
  startup_script = <<-EOT
    set -e
    # Install the latest code-server.
    # Append "--version x.x.x" to install a specific version of code-server.
    curl -fsSL https://code-server.dev/install.sh | sh -s -- --method=standalone --prefix=/tmp/code-server
    # Start code-server in the background.
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
  count        = data.coder_workspace.dev.start_count
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

data "cloudinit_config" "user_data" {
  gzip          = false
  base64_encode = false
  boundary = "//"
  part {
    filename     = "cloud-config.yaml"
    content_type = "text/cloud-config"

    content = templatefile("${path.module}/cloud-init/cloud-config.yaml.tftpl", {
      hostname   = local.hostname
      linux_user = local.linux_user
    })
  }
  part {
    filename     = "userdata.sh"
    content_type = "text/x-shellscript"
    content = templatefile("${path.module}/cloud-init/userdata.sh.tftpl", {
      linux_user = local.linux_user
      init_script = try(coder_agent.dev[0].init_script, "")
    })
  }
}


resource "aws_instance" "dev" {
  ami               = "ami-0cf2b4e024cdb6960" # ubuntu
  availability_zone = "${local.aws_region}a"
  instance_type     = data.coder_parameter.instance_type.value
  subnet_id = tolist(data.aws_subnets.private.ids)[0]
  user_data =  data.cloudinit_config.user_data.rendered
  root_block_device {
    volume_size = tonumber(data.coder_parameter.instance_disk.value)
  }
  tags = {
    Name = "coder-${data.coder_workspace_owner.me.name}-${data.coder_workspace.dev.name}"
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
    key   = "region"
    value = local.aws_region
  }
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
  state       = data.coder_workspace.dev.transition == "start" ? "running" : "stopped"
}
