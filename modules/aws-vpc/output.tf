output "vpc_id" {
  value = module.vpc.vpc_id
}

output "public_subnets" {
  value = module.vpc.public_subnets
}

output "azs" {
  value = module.vpc.azs
}

output "sg_ssh_id" {
  value = resource.aws_security_group.ssh.id
}

output "sg_http-ssh_id" {
  value = resource.aws_security_group.http-ssh.id
}
