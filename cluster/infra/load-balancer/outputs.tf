output "load_balancer" {
  value = hcloud_load_balancer.this
}

output "ip" {
  value = hcloud_load_balancer.this.ipv4
}
