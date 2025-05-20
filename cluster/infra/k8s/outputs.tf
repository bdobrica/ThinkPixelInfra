output "master_ip" {
  value = module.master.ip
}

output "master_nodes" {
  value = module.master.nodes
}

output "worker_nodes" {
  value = module.worker.nodes
}

output "worker_ip" {
  value = module.worker.ip
}

output "worker_load_balancer" {
  value = module.worker.load_balancer.load_balancer
}
