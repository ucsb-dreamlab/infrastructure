terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

module "ec2_key_pair" {
  source     = "terraform-aws-modules/key-pair/aws"
  key_name   = "${var.name}-key-pair"
  public_key = var.public_key
}

module "ec2_instance" {
  source                      = "terraform-aws-modules/ec2-instance/aws"
  name                        = var.name
  instance_type               = var.instance_type # arm64
  ami                         = var.ami           # debian 12 arm64
  key_name                    = module.ec2_key_pair.key_pair_name
  availability_zone           = var.availability_zone
  subnet_id                   = var.subnet_id
  vpc_security_group_ids      = var.vpc_security_group_ids
  associate_public_ip_address = var.associate_public_ip_address
  enable_volume_tags          = false
  root_block_device = [
    {
      volume_type = "gp3"
      throughput  = 200
      volume_size = var.root_disk_size
      tags = {
        Tofu = "true"
        Name = "${var.name}-root-disk"
      }
    },
  ]
  tags = {
    Tofu = "true"
    Name = "${var.name}-instance"
  }
}
