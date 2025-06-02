locals {
  master_ipv4 = [for server in data.hcloud_server.master : server.ipv4_address]
  master_ipv6 = [for server in data.hcloud_server.master : server.ipv6_address]
  worker_ipv4 = [for server in data.hcloud_server.worker : server.ipv4_address]
  worker_ipv6 = [for server in data.hcloud_server.worker : server.ipv6_address]

  ip_maps = {
    "admin" = var.admin_ips
    "nodes" = concat(local.master_ipv4, local.worker_ipv4)
  }
}

data "hcloud_server" "master" {
  for_each = var.master_nodes
  name     = each.key
}

data "hcloud_server" "worker" {
  for_each = var.worker_nodes
  name     = each.key
}

resource "hcloud_firewall" "this" {
  name = var.name

  dynamic "rule" {
    for_each = var.inbound_rules
    content {
      direction  = "in"
      protocol   = rule.value.protocol
      port       = rule.value.port
      source_ips = toset(flatten([for ip_group in rule.value.source : local.ip_maps[ip_group]]))
    }
  }

  dynamic "rule" {
    for_each = var.outbound_rules
    content {
      direction       = "out"
      protocol        = rule.value.protocol
      port            = rule.value.port
      destination_ips = toset(flatten([for ip_group in rule.value.destination : local.ip_maps[ip_group]]))
    }
  }
}

resource "hcloud_firewall_attachment" "this" {
  firewall_id = hcloud_firewall.this.id
  server_ids = concat(
    [for server in data.hcloud_server.master : server.id],
    [for server in data.hcloud_server.worker : server.id]
  )
}
