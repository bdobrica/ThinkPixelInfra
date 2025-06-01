variable "cluster_domain" {
  description = "The domain name of the cluster"
  default     = "cluster.local"
  type        = string
}

variable "name_prefix" {
  description = "The name prefix of the master nodes. It will have the format <name_prefix>-<index>"
  default     = "master"
  type        = string
}

variable "network" {
  description = "The the network resource to attach the master nodes to"
}

variable "subnet" {
  description = "Adding servers to the network requires a subnet. Not used, but required as a dependency"
}

variable "servers" {
  description = "The number of master nodes to create"
  default     = 1

  validation {
    condition     = var.servers > 0
    error_message = "The no. of master nodes must be at least 1"
  }

  validation {
    condition     = var.servers <= 9
    error_message = "The no. of master nodes must be at most 9"
  }

  validation {
    condition     = var.servers % 2 == 1
    error_message = "The no. of master nodes must be an odd number, to ensure a majority"
  }
}

variable "server_type" {
  description = "The server type to use for the master nodes. All master nodes must have the same server type"
  default     = "cx22"
}

variable "location" {
  description = "The location to create the server"
  default     = "fsn1"
}

variable "os_image" {
  description = "The OS image to use for the master nodes"
  default     = "debian-12"
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
