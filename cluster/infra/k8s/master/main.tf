# Create the master node(s)
locals {
    ip_offset = 11
    master_ip = cidrhost(var.subnet.ip_range, local.ip_offset)
}

resource "hcloud_server" "this" {
  count       = var.servers
  name        = format("%s-%02d", var.name_prefix, count.index + 1)
  image       = var.os_image
  server_type = var.server_type
  location    = var.location
  public_net {
    ipv4_enabled = true
    ipv6_enabled = true
  }
  network {
    network_id = var.network.id
    ip         = cidrhost(var.subnet.ip_range, count.index + local.ip_offset)
  }
  user_data = count.index == 0 ? module.master_user_data.yaml : module.replica_user_data.yaml
  labels = {
    "node_type" = "master"
  }
}
