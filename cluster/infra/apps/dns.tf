# Specify a zone for a domain (example.com)
resource "hetznerdns_zone" "this" {
  for_each = local.dns_zones
  name     = each.key
  ttl      = 60
}

# Handle root (example.com)
resource "hetznerdns_record" "this" {
  for_each = local.dns_records
  zone_id  = hetznerdns_zone.this[each.value.zone].id
  name     = each.value.name
  value    = var.worker_load_balancer.ipv4
  type     = "A"
}
