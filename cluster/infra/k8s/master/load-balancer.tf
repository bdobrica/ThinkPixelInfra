module "load_balancer" {
    source = "../../load-balancer"
    count = var.servers > 1 ? 1 : 0

    name = format("%s-lb", var.name_prefix)
    protocols = ["http", "https", "ssh"]
    location = var.location
    network = var.network
    subnet = var.subnet
    nodes = {for server in hcloud_server.this : server.name => server.id}
}
