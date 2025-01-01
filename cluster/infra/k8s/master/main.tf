# Create the master node(s)
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
    # IP Used by the master node, needs to be static
    # Here the worker nodes will use 10.0.1.1 to communicate with the master node
    ip         = cidrhost(var.subnet.ip_range, count.index + 1)
  }
  user_data = templatefile(
    count.index > 0 ? "${path.module}/data/99-master.yaml" : "${path.module}/data/01-master.yaml", {
    cluster_domain = var.cluster_domain
    first_node_ip = cidrhost(var.subnet.ip_range, 1)
    management_public_ssh_key = var.management_public_ssh_key
    master_public_ssh_key = var.master_public_ssh_key
    master_private_ssh_key = var.master_private_ssh_key
    worker_public_ssh_key = var.worker_public_ssh_key
  })
  labels = {
    "node_type" = "master"
  }
}
