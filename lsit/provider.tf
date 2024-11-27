terraform {
  required_version = ">= 0.13"
  required_providers {
    harvester = {
      source  = "harvester/harvester"
      version = "0.6.5"
    }
  }
}

provider "harvester" {
    kubeconfig = "${path.module}/compute-lsit.yaml"
}
