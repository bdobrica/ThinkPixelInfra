module "master_user_data" {
  source = "../../user-data"

  cluster_domain = var.cluster_domain
  node_type      = "master"
    master_ip      = local.master_ip
  ssh_authorized_keys = [
    var.management_public_ssh_key,
    var.master_public_ssh_key,
    var.worker_public_ssh_key,
  ]
  ssh_private_key     = var.master_private_ssh_key
}

module "replica_user_data" {
    source = "../../user-data"
    
    cluster_domain = var.cluster_domain
    node_type      = "master_replica"
    master_ip      = local.master_ip
    ssh_authorized_keys = [
        var.management_public_ssh_key,
        var.master_public_ssh_key,
        var.worker_public_ssh_key,
    ]
    ssh_private_key     = var.master_private_ssh_key
}
