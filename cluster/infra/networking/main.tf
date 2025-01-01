# Create a private network and a subnet
resource "hcloud_network" "this" {
  name     = var.name
  ip_range = "172.16.0.0/12"
}

resource "hcloud_network_subnet" "this" {
  type         = "cloud"
  network_id   = hcloud_network.this.id
  network_zone = "eu-central"
  ip_range     = "172.16.0.0/16"
}
