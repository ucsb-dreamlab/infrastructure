terraform {
  backend "s3" {
    bucket = "dreamlab-tf"
    key    = "dreamlab-test.tfstate"
    region = "us-west-2"
  }
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

module "aws-vpc" {
  source = "../modules/aws-vpc"
  name   = "dreamlab-test"
}

locals {
  instance_az     = element(module.aws-vpc.azs, 0)
  instance_subnet = element(module.aws-vpc.public_subnets, 0)
  inance_sg_ids   = [module.aws-vpc.sg_ssh_id]
}

module "test-instance" {
  source                 = "../modules/aws-instance"
  name                   = "test-instance"
  public_key             = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINohiqm7xqUgn1S6ITjT4olz5dcMvAsdV7XT5ScywYew serickson@DRM04-L1DM"
  availability_zone      = local.instance_az
  subnet_id              = local.instance_az
  vpc_security_group_ids = local.inance_sg_ids
}

output "instance_ip" {
  value = module.test-instance.public_ip
}
