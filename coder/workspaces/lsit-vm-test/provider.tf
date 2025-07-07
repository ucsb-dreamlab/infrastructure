terraform {
  required_version = ">= 0.13"
  required_providers {
    coder = {
      source = "coder/coder"
      version = "2.8.0"
    }
    harvester = {
      source  = "harvester/harvester"
      version = "0.6.7"
    }
  }
}

provider "harvester" {
    kubeconfig = "/etc/coder/lsit-kubeconfig.yaml"
}
