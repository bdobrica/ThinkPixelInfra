# Create worker nodes
locals {
    worker_nodes = flatten([
        for pool_name, pool in var.server_pools : [
        for i in range(pool.count) : {
            name        = format("%s-%s-%02d", var.name_prefix, pool_name, i + 1)
            server_type = pool.server_type
            os_image    = var.os_image
            location    = var.location
            network     = var.network
            user_data   = templatefile("${path.module}/data/01-worker.yaml", {
                master_ip = var.master_ip
                management_public_ssh_key = var.management_public_ssh_key
                worker_private_ssh_key = var.worker_private_ssh_key
            })
            labels = {
                "node_type" = "worker"
                "pool"      = pool_name
            }
        }
        ]
    ])
}

resource "hcloud_server" "this" {
  for_each = { for node in local.worker_nodes : node.name => node }
  
  # The name will be worker-node-0, worker-node-1, worker-node-2...
  name        = each.value.name
  image       = each.value.os_image
  server_type = each.value.server_type
  location    = each.value.location
  public_net {
    ipv4_enabled = true
    ipv6_enabled = true
  }
  network {
    network_id = each.value.network.id
  }
  user_data = each.value.user_data
  labels    = each.value.labels
}
