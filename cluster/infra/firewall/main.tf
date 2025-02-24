locals {
    cluster_ips = flatten([var.master_node_ips, [var.master_lb_ip], var.worker_node_ips, [var.worker_lb_ip]])
}

resource "hcloud_firewall" "this" {
  name = "${var.cluster_prefix}-${var.name}"

  rule {
    direction     = "in"
    source_ips    = [var.management_host_ip]
    protocol      = "tcp"
    port          = "22"
    description   = "Allow SSH from management host"
  }

  rule {
    direction     = "in"
    source_ips    = [var.management_host_ip]
    protocol      = "tcp"
    port          = "3000"
    description   = "Allow monitoring from management host"
  }

  rule {
    direction     = "in"
    source_ips    = [var.management_host_ip]
    protocol      = "tcp"
    port          = "6443"
    description   = "Allow Kubernetes API from management host"
  }

  rule {
    direction     = "in"
    source_ips    = local.cluster_ips
    protocol      = "tcp"
    description   = "Allow TCP internal traffic to Kubernetes cluster"
  }

  rule {
    direction     = "in"
    source_ips    = local.cluster_ips
    protocol      = "udp"
    description   = "Allow UDP internal traffic to Kubernetes cluster"
  }

  apply_to {
    label_selector = [
        for key, value in var.labels : "${key}=${value}"
    ]
  }
}
