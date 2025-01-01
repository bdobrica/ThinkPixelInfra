module "load_balancer" {
    source = "../../load-balancer"

    name = format("%s-lb", var.name_prefix)
    protocols = ["https"]
    location = var.location
    network = var.network
    subnet = var.subnet
    nodes = {for server in hcloud_server.this : server.name => server.id}
}
