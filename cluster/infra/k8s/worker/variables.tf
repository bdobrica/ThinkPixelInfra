variable "cluster_domain" {
  description = "The domain name of the cluster"
  default     = "cluster.local"
  type        = string
}

variable "name_prefix" {
  description = "The name prefix of the worker nodes. It will have the format <name_prefix>-<index>"
  default     = "worker"
  type        = string
}

variable "network" {
  description = "The the network resource to attach the master nodes to"
}

variable "subnet" {
  description = "Adding servers to the network requires a subnet. Not used, but required as a dependency"
}

variable "server_pools" {
  description = "The server pools to use as worker nodes"
  type = map(object({
    server_type = string
    count       = number
    taint       = optional(string, "")
  }))
  default = {
    "pool1" = {
      server_type = "cx22"
      count       = 3
      taint       = ""
    }
  }
}

variable "location" {
  description = "The location to create the worker nodes"
  default     = "fsn1"
}

variable "os_image" {
  description = "The OS image to use for the worker nodes"
  default     = "debian-12"
}

variable "master_ip" {
  description = "The IP address of the master node or load balancer to master nodes"
}

variable "management_public_ssh_key" {
  description = "The public SSH key to use for management access"
  type        = string
}

variable "worker_private_ssh_key" {
  description = "The private SSH key to use for worker node access"
  type        = string
}
