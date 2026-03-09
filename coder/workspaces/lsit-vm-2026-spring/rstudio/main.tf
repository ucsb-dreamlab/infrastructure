terraform {
  required_version = ">= 1.0"

  required_providers {
    coder = {
      source  = "coder/coder"
      version = ">= 0.17"
    }
  }
}

variable "agent_id" {
  type        = string
  description = "The ID of a Coder agent."
}

variable "display_name" {
  type        = string
  description = "The display name for the VS Code Web application."
  default     = "VS Code Web"
}

variable "order" {
  type        = number
  description = "The order determines the position of app in the UI presentation. The lowest order is shown first and apps with equal order are sorted by name (ascending order)."
  default     = null
}

variable "rserver_user" {
  type        = string
  description = "rserver user"
  default     = "coder"
}

variable "settings" {
  type        = any
  description = "A map of settings to apply to VS Code web."
  default     = {}
}


data "coder_workspace_owner" "me" {}
data "coder_workspace" "env" {}

resource "coder_script" "vscode-web" {
  agent_id     = var.agent_id
  display_name = "RStudio"
  icon         = "/icon/rstudio.svg"
  script = templatefile("${path.module}/run.sh", {
    rserver_user : var.rserver_user,
  })
  run_on_start = true
}


resource "coder_app" "rstudio" {
  count        = data.coder_workspace.env.start_count
  agent_id     = var.agent_id
  slug         = "rstudio"
  display_name = "RStudio"
  url          = "http://localhost:8787"
  icon         = "/icon/rstudio.svg"
  subdomain    = true
  share        = "owner"
  order        = var.order
  healthcheck {
    url       = "http://localhost:8787/health-check"
    interval  = 4
    threshold = 40
  }
}