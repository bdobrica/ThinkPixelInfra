locals {
  domain_pieces = split(".", var.cluster_domain)
  tld           = join(".", slice(local.domain_pieces, length(local.domain_pieces) - 2, length(local.domain_pieces)))
  domain_names  = tolist(toset([local.tld, var.cluster_domain]))
}

module "load_balancer" {
  source = "../../load-balancer"

  name         = format("%s-lb", var.name_prefix)
  protocols    = ["http"]
  domain_names = local.domain_names
  location     = var.location
  network      = var.network
  subnet       = var.subnet
  nodes        = { for server in hcloud_server.this : server.name => server.id }
}
