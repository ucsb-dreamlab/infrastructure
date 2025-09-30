terraform {
  required_version = ">= 0.13"
  required_providers {
    coder = {
      source = "coder/coder"
      version = "2.11.0"
    }
    harvester = {
      source  = "harvester/harvester"
      version = "1.6.0"
    }
  }
}

provider "harvester" {
    kubeconfig = "/etc/coder/kubeconfig/outerrim.yaml"
}
