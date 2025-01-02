# https://docs.hetzner.com/cloud/general/locations/
locals {
  zones = {
    "eu-central" = [
      "fsn1",
      "hel1",
      "nbg1"
    ],
    "us-east" = [
      "ash"
    ],
    "us-west" = [
      "hil"
    ],
    "ap-southeast" = [
      "sin"
    ]
  }
  zone = [
    for zone, locations in local.zones : zone
    if contains(locations, var.location)
  ][0]
}
