variable "cluster_domain" {
  description = "The domain name for the cluster"
  type        = string
  default     = "cluster.local"
}

variable "node_type" {
  description = "The type of node to create"
  type        = string
}

variable "master_ip" {
  description = "The IP address of the master node"
  type        = string
  default     = null
}

variable "ssh_authorized_keys" {
  description = "The SSH public keys to add to the cluster user"
  type        = list(string)
  default     = []
}

variable "ssh_private_key" {
  description = "The SSH private key to use for the cluster user"
  type        = string
  default     = ""
}
