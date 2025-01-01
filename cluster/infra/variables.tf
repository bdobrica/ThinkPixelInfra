# Declare the hcloud_token variable from .tfvars
variable "hcloud_token" {
  sensitive   = true # Requires terraform >= 0.14
  description = "Value of the Hetzner Cloud API token"
}

variable "cluster_domain" {
  description = "The domain name of the cluster"
  default     = "cluster.local"
  type        = string
}

variable "cluster_prefix" {
  description = "The name prefix for all nodes. It will have the format <cluster_prefix>-<node_prefix>-<index>"
  default     = "k8s"
  type        = string
}

variable "location" {
  description = "The location to create the worker nodes"
  default     = "fsn1"
}

variable "os_image" {
  description = "The OS image to use for the servers"
  default     = "debian-12"
}

variable "master_pool" {
  description = "The master pool to create"
  type = object({
    server_type = string
    count       = number
  })
  default = {
    server_type = "cx22"
    count       = 1
  }
}

variable "worker_pools" {
  description = "The worker pools to create"
  type = map(object({
    server_type = string
    count       = number
  }))
  default = {
    "pool1" = {
      server_type = "cx22"
      count       = 3
    }
  }
}

variable "management_public_ssh_key" {
  description = "The public SSH key to use for management access"
  type        = string
}

variable "master_public_ssh_key" {
  description = "The public SSH key to use for master node access"
  type        = string
}

variable "master_private_ssh_key" {
  description = "The private SSH key to use for master node access"
  type        = string
}

variable "worker_public_ssh_key" {
  description = "The public SSH key to use for worker node access"
  type        = string
}

variable "worker_private_ssh_key" {
  description = "The private SSH key to use for worker node access"
  type        = string
}
