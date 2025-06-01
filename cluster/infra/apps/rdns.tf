
resource "hcloud_rdns" "this" {
  provider = hcloud
  for_each = local.rdns_records

  ip_address       = var.worker_load_balancer.ipv4
  load_balancer_id = var.worker_load_balancer.id

  dns_ptr = each.key
}
