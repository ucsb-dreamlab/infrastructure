terraform {
  required_version = ">= 0.13"
  required_providers {
    coder = {
      source = "coder/coder"
      version = "2.1.2"
    }
    harvester = {
      source  = "harvester/harvester"
      version = "0.6.6"
    }
  }
}

provider "harvester" {
    kubeconfig = "/etc/coder/lsit-kubeconfig.yaml"
}
