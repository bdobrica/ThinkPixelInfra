locals {
  rdns_records = toset(flatten([
    for name, app in var.exposed_apps : concat([
      format("%s.%s", name, var.cluster_domain)
      ],
      [
        for alias in var.cluster_domain_aliases :
        format("%s.%s", name, alias)
    ])
  ]))

  dns_record_parts = {
    for rdns_record in local.rdns_records :
    rdns_record => split(".", rdns_record)
  }

  dns_records = {
    for dns_record, parts in local.dns_record_parts :
    dns_record => {
      name = join(".", slice(parts, 0, length(parts) - 2))
      zone = join(".", slice(parts, length(parts) - 2, length(parts)))
    }
  }

  dns_zones = toset([
    for dns_record in local.dns_records :
    dns_record.zone
  ])
}
