# Create worker nodes
locals {
  ip_offset = 21
  worker_nodes = flatten([
    for pool_name, pool in var.server_pools : [
      for i in range(pool.count) : {
        name        = format("%s-%s-%02d", var.name_prefix, pool_name, i + 1)
        server_type = pool.server_type
        pool_name   = pool_name
        labels = {
          "node_type" = "worker"
          "pool"      = pool_name
        }
      }
    ]
  ])
}

resource "hcloud_server" "this" {
  count = length(local.worker_nodes)

  # The name will be worker-node-0, worker-node-1, worker-node-2...
  name        = local.worker_nodes[count.index].name
  image       = var.os_image
  server_type = local.worker_nodes[count.index].server_type
  location    = var.location
  public_net {
    ipv4_enabled = true
    ipv6_enabled = true
  }
  network {
    network_id = var.network.id
    ip         = cidrhost(var.subnet.ip_range, count.index + local.ip_offset)
  }
  user_data = module.worker_user_data[local.worker_nodes[count.index].pool_name].yaml
  labels    = local.worker_nodes[count.index].labels
}
