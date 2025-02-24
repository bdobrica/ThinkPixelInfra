variable "cluster_prefix" {
  description = "The prefix to use for naming the firewall. Names will be generated as `${cluster_prefix}-${name}`"
  type = string
}

variable "name" {
  description = "The name of the firewall. Will be prefixed with the cluster prefix as `${cluster_prefix}-${name}`"
  type = string
}

variable "master_node_ips" {
  description = "The list of master node IPs to protect with the firewall"
  type = list(string)
}

variable "master_lb_ip" {
  description = "The IP address of the master load balancer to protect with the firewall"
  type = string
  default = null
}

variable "worker_node_ips" {
  description = "The list of worker node IPs to protect with the firewall"
  type = list(string)
}

variable "worker_lb_ip" {
  description = "The IP address of the worker load balancer to protect with the firewall"
  type = string
}

variable "labels" {
  description = "The server labels to apply the firewall to"
  type = map(string)
}

variable "management_host_ip" {
  description = "The IP address of the management host (e.g. your local machine) to allow access to the servers"
  type = string
}
