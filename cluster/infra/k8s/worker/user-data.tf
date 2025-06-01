module "worker_user_data" {
  source = "../../user-data"

  for_each = var.server_pools

  cluster_domain = var.cluster_domain
  node_taint     = each.key
  node_type      = "worker"
  master_ip      = var.master_ip
  ssh_authorized_keys = [
    var.management_public_ssh_key,
  ]
  ssh_private_key = var.worker_private_ssh_key
}
