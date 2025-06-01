resource "hcloud_managed_certificate" "this" {
  provider = hcloud
  for_each = var.exposed_apps

  name = format("%s-cert", each.key)
  domain_names = flatten(concat(
    [
      format("%s.%s", each.key, var.cluster_domain),
      format("*.%s.%s", each.key, var.cluster_domain)
    ],
    [
      for alias in var.cluster_domain_aliases : [
        format("%s.%s", each.key, alias),
        format("*.%s.%s", each.key, alias)
      ]
    ]
  ))
}
