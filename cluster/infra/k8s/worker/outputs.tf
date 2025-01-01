output "nodes" {
    value = {for server in hcloud_server.this : server.name => server.id}
}

output "ip" {
    value = module.load_balancer.ip
}
