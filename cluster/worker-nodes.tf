# Create worker nodes
resource "hcloud_server" "worker-nodes" {
  count = 3
  
  # The name will be worker-node-0, worker-node-1, worker-node-2...
  name        = "worker-node-${count.index}"
  image       = "ubuntu-24.04"
  server_type = "cax11"
  location    = "fsn1"
  public_net {
    ipv4_enabled = true
    ipv6_enabled = true
  }
  network {
    network_id = hcloud_network.private_network.id
  }
  user_data = file("${path.module}/cloud-init/worker.yaml")

  depends_on = [hcloud_network_subnet.private_network_subnet, hcloud_server.master-node]
}
