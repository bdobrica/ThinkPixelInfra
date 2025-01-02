output "yaml" {
  value = format("#cloud-config\n%s", yamlencode(local.template))
}
