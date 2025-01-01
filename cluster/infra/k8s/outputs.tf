output "master_ip" {
  value = module.master.ip
}

output "master_nodes" {
  value = module.master.nodes
}

output "worker_ip" {
  value = module.worker.ip
}

output "worker_nodes" {
  value = module.worker.nodes
}
