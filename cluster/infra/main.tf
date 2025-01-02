# Configure the Hetzner Cloud Provider with your token
terraform {
  required_version = ">= 1.10.0"
}
provider "hcloud" {
  token = var.hcloud_token
}

module "network" {
  source   = "./networking"
  name     = format("%s-net", var.cluster_prefix)
  location = var.location
}

module "k8s" {
  source = "./k8s"

  cluster_domain            = var.cluster_domain
  cluster_prefix            = var.cluster_prefix
  network                   = module.network.network
  subnet                    = module.network.subnet
  master_pool               = var.master_pool
  worker_pools              = var.worker_pools
  location                  = var.location
  os_image                  = var.os_image
  management_public_ssh_key = var.management_public_ssh_key
  master_public_ssh_key     = var.master_public_ssh_key
  master_private_ssh_key    = var.master_private_ssh_key
  worker_public_ssh_key     = var.worker_public_ssh_key
  worker_private_ssh_key    = var.worker_private_ssh_key
}
