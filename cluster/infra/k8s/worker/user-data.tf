module "worker_user_data" {
  source = "../../user-data"

  cluster_domain = ""
  node_type      = "worker"
    master_ip      = var.master_ip
  ssh_authorized_keys = [
    var.management_public_ssh_key,
  ]
  ssh_private_key     = var.worker_private_ssh_key
}
