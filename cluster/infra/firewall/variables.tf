variable "name" {
  description = "The name of the firewall."
  type        = string
}

variable "master_nodes" {
  description = "A mapping of master node names to their respective server IDs."
  type        = map(string)
  default     = {}
}

variable "worker_nodes" {
  description = "A mapping of worker node names to their respective server IDs."
  type        = map(string)
  default     = {}
}

variable "admin_ips" {
  description = "A list of IP addresses that are allowed to access the firewall."
  type        = list(string)
  default     = []
}

variable "inbound_rules" {
  description = "A list of inbound rules for the firewall."
  type = list(object({
    protocol = string
    port     = string
    source   = list(string)
  }))
  default = [
    {
      protocol = "tcp"
      port     = "any"
      source   = ["nodes"]
    },
    {
      "protocol" = "udp"
      "port"     = "any"
      "source"   = ["nodes"]
    },
    {
      "protocol" = "tcp",
      "port"     = "22",
      "source"   = ["admin"]
    },
    {
      "protocol" = "tcp",
      "port"     = "6443",
      "source"   = ["admin"]
    },
    {
      "protocol" = "tcp",
      "port"     = "3000",
      "source"   = ["admin"]
    }
  ]
}

variable "outbound_rules" {
  description = "A list of outbound rules for the firewall."
  type = list(object({
    protocol    = string
    port        = number
    destination = list(string)
  }))
  default = []
}
