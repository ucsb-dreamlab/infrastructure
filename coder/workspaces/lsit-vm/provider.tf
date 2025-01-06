terraform {
  required_version = ">= 0.13"
  required_providers {
    coder = {
      source = "coder/coder"
    }
    harvester = {
      source  = "harvester/harvester"
      version = "0.6.5"
    }
  }
}

provider "harvester" {
    kubeconfig = "/etc/coder/lsit-kubeconfig.yaml"
}
