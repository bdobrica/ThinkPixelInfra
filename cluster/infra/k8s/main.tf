module "master" {
  source = "./master"

  cluster_domain            = var.cluster_domain
  name_prefix               = format("%s-master", var.cluster_prefix)
  network                   = var.network
  subnet                    = var.subnet
  server_type               = var.master_pool.server_type
  servers                   = var.master_pool.count
  location                  = var.location
  os_image                  = var.os_image
  management_public_ssh_key = var.management_public_ssh_key
  master_public_ssh_key     = var.master_public_ssh_key
  master_private_ssh_key    = var.master_private_ssh_key
  worker_public_ssh_key     = var.worker_public_ssh_key
}

module "worker" {
  source = "./worker"

  cluster_domain            = var.cluster_domain
  name_prefix               = format("%s-worker", var.cluster_prefix)
  network                   = var.network
  subnet                    = var.subnet
  server_pools              = var.worker_pools
  location                  = var.location
  os_image                  = var.os_image
  management_public_ssh_key = var.management_public_ssh_key
  master_ip                 = module.master.ip
  worker_private_ssh_key    = var.worker_private_ssh_key
}

module "firewall" {
  source = "../firewall"

  name         = format("%s-firewall", var.cluster_prefix)
  master_nodes = module.master.nodes
  worker_nodes = module.worker.nodes
  admin_ips    = var.admin_ips
  depends_on   = [module.master, module.worker]
}
