variable "name" {
  description = "The name of the load balancer"
}

variable "location" {
  description = "The location of the load balancer"
  default     = "fsn1"
}

variable "network" {
  description = "The network to attach the load balancer to"
}

variable "subnet" {
  description = "The subnet to attach the load balancer to"
}

variable "nodes" {
  description = "A mapping of server names to server IDs"
  type        = map(string)
}

variable "protocols" {
  description = "The list of protocols to listen on"
  type        = set(string)
  default     = ["http", "https"]
}

variable "domain_names" {
  description = "The list of domain names to listen on"
  type        = list(string)
}
