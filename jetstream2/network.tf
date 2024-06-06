################
# Networking
################


# getting public network id for routing
data "openstack_networking_network_v2" "public" {
  name = "public"
}

#creating the virtual network
resource "openstack_networking_network_v2" "terraform_network" {
  name = "terraform_network"
  admin_state_up  = "true"
  tags = ["terraform"]
}

#creating the virtual subnet
resource "openstack_networking_subnet_v2" "terraform_subnet1" {
  name = "terraform_subnet1"
  network_id  = "${openstack_networking_network_v2.terraform_network.id}"
  cidr  = "192.168.5.0/24"
  ip_version  = 4
  tags = ["terraform"]
}

# setting up virtual router
resource "openstack_networking_router_v2" "terraform_router" {
  name = "terraform_router"
  admin_state_up  = true
  # id of public network at JS1/2
  external_network_id = data.openstack_networking_network_v2.public.id
  tags = ["terraform"]
}

# setting up virtual router interface
resource "openstack_networking_router_interface_v2" "terraform_router_interface_1" {
  router_id = "${openstack_networking_router_v2.terraform_router.id}"
  subnet_id = "${openstack_networking_subnet_v2.terraform_subnet1.id}"
}
