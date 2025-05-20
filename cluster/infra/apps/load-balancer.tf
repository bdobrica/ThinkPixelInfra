resource "hcloud_load_balancer_service" "this" {
  provider         = hcloud
  for_each         = var.exposed_apps
  load_balancer_id = var.worker_load_balancer.id
  protocol         = each.value.protocol
  listen_port      = each.value.port
  destination_port = each.value.port

  http {
    sticky_sessions = true
    cookie_name     = format("%s-session", each.key)
    certificates    = [hcloud_managed_certificate.this[each.key].id]
  }

  health_check {
    protocol = "http"
    port     = each.value.port
    interval = try(each.value.health_check.interval, 15) # Default interval for health checks
    timeout  = try(each.value.health_check.timeout, 10)  # Default timeout for health checks
    retries  = try(each.value.health_check.retries, 3)   # Default retries for health checks

    http {
      domain       = format("%s.%s", each.key, var.cluster_domain)
      path         = try(each.value.health_check.path, "/healthz")
      status_codes = ["2??", "3??"] # Accept 2xx and 3xx status codes
    }
  }
}
