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
  depends_on         = [var.subnet]
}

resource "hcloud_managed_certificate" "this" {
  count = contains(var.protocols, "https") ? length(var.domain_names) : 0

  name = format("%s-cert-%02d", var.name, count.index + 1)
  domain_names = [
    format("*.%s", var.domain_names[count.index]),
    var.domain_names[count.index]
  ]
}

resource "hcloud_load_balancer_service" "this" {
  for_each         = toset(var.protocols)
  load_balancer_id = hcloud_load_balancer.this.id
  protocol         = local.protocols[each.key].protocol
  listen_port      = local.protocols[each.key].listen_port

  dynamic "http" {
    for_each = each.key == "https" ? [1] : []
    content {
      certificates  = [for cert in hcloud_managed_certificate.this : cert.id]
      redirect_http = true
    }
  }
}

resource "hcloud_load_balancer_network" "this" {
  load_balancer_id = hcloud_load_balancer.this.id
  subnet_id        = var.subnet.id
}

resource "hcloud_load_balancer_target" "this" {
  for_each = var.nodes

  type             = "server"
  load_balancer_id = hcloud_load_balancer.this.id
  server_id        = each.value
  use_private_ip   = true

  depends_on = [var.subnet, hcloud_load_balancer_network.this]
}
