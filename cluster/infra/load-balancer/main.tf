locals {
  protocols = {
    http = {
      protocol    = "http"
      listen_port = 80
    }
    https = {
      protocol    = "https"
      listen_port = 443
    }
    ssh = {
      protocol    = "tcp"
      listen_port = 22
    }
  }
}

resource "hcloud_load_balancer" "this" {
  name               = var.name
  load_balancer_type = length(var.nodes) <= 25 ? "lb11" : (length(var.nodes) <= 75 ? "lb21" : "lb31")
  location           = var.location
  delete_protection  = true
}

resource "hcloud_load_balancer_service" "this" {
  for_each         = toset(var.protocols)
  load_balancer_id = hcloud_load_balancer.this.id
  protocol         = local.protocols[each.key].protocol
  listen_port      = local.protocols[each.key].listen_port
}

resource "hcloud_load_balancer_network" "this" {
  load_balancer_id = hcloud_load_balancer.this.id
  network_id       = var.network.id
}

resource "hcloud_load_balancer_target" "this" {
  for_each = var.nodes

  type             = "server"
  load_balancer_id = hcloud_load_balancer.this.id
  server_id        = each.value
  use_private_ip   = true
}
