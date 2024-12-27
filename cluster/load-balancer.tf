# Setup load balancer
resource "hcloud_load_balancer" "load_balancer" {
  name               = "kubernetes-load-balancer"
  load_balancer_type = "lb11"
  location           = "fsn1"
  delete_protection  = true

  depends_on = [hcloud_network_subnet.private_network_subnet, hcloud_server.master-node]
}

resource "hcloud_load_balancer_service" "load_balancer_service" {
  load_balancer_id = hcloud_load_balancer.load_balancer.id
  protocol         = "http"

  depends_on = [hcloud_load_balancer.load_balancer]
}

resource "hcloud_load_balancer_network" "lb_network" {
  load_balancer_id = hcloud_load_balancer.load_balancer.id
  network_id       = hcloud_network.private_network.id

  depends_on = [hcloud_network_subnet.private_network_subnet, hcloud_load_balancer.load_balancer]
}

resource "hcloud_load_balancer_target" "load_balancer_target_worker" {
  for_each = { for idx, server in hcloud_server.worker-nodes : idx => server }
  
  type             = "server"
  load_balancer_id = hcloud_load_balancer.load_balancer.id
  server_id        = each.value.id
  use_private_ip   = true

  depends_on = [hcloud_network_subnet.private_network_subnet, hcloud_load_balancer_network.lb_network, hcloud_server.worker-nodes]
}
