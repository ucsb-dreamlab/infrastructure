
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

module "aws" {
  source = "../modules/aws"
  name   = "dreamlab-test"
}
