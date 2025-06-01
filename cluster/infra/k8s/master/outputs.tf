output "nodes" {
  value = { for server in hcloud_server.this : server.name => server.id }
}

output "ip" {
  value = var.servers > 1 ? module.load_balancer.ip : local.master_ip
}

output "first_ip" {
  value = hcloud_server.this[0].ipv4_address
}
