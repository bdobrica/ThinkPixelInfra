variable "cluster_domain" {
  description = "The domain name for the cluster"
  type        = string
  default     = "cluster.local"
}

variable "cluster_domain_aliases" {
  description = "A list of domain aliases for the cluster"
  type        = list(string)
  default     = []
}

variable "worker_load_balancer" {
  description = "The load balancer configuration for the worker nodes."
  type        = any
}

variable "exposed_apps" {
  description = "A list of applications to expose via the firewall."
  type = map(object({
    protocol      = string
    external_port = number
    internal_port = number
    health_check = optional(object({
      path     = string
      interval = optional(number, 15) # Default interval for health checks
      timeout  = optional(number, 10) # Default timeout for health checks
      retries  = optional(number, 3)  # Default retries for health checks
    }))
  }))
  default = {}
}
