variable "name" {
  type        = string
  description = "instance name"
}

variable "public_key" {
  type        = string
  description = "ssh public key"
}

variable "instance_type" {
  type        = string
  description = "instance type"
  default     = "t4g.medium" # arm64
}

variable "ami" {
  type        = string
  description = "ID of AMI to use for the inance"
  default     = "ami-07564a05443c48891" # debian 12 arm64
}

variable "availability_zone" {
  type        = string
  description = "AZ to start the instance in"
}

variable "subnet_id" {
  type        = string
  description = "VPC Subnet ID to launch in"
}

variable "vpc_security_group_ids" {
  type        = list(string)
  description = "A list of security group IDs"
}

variable "root_disk_size" {
  type        = number
  description = "size of root disk in GB"
  default     = 50
}

variable "associate_public_ip_address" {
  type        = bool
  description = "whether to associate the instance with a public IP in the VPC"
  default     = true
}
