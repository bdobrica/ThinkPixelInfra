locals {
  runcmds = {
    master = [
      "curl https://get.k3s.io | INSTALL_K3S_EXEC=\"server --disable traefik --cluster-domain ${var.cluster_domain}\" sh -",
      "chown cluster:cluster /etc/rancher/k3s/k3s.yaml",
      "chown cluster:cluster /var/lib/rancher/k3s/server/node-token",
    ],
    master_replica = [
      "until curl -k https://${var.master_ip}:6443; do sleep 5; done",
      "REMOTE_TOKEN=$(ssh -o StrictHostKeyChecking=accept-new cluster@${var.master_ip} sudo cat /var/lib/rancher/k3s/server/node-token)",
      "curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC=\"server --server https://${var.master_ip}:6443\" K3S_TOKEN=$REMOTE_TOKEN sh -"
    ],
    worker = [
      "until curl -k https://${var.master_ip}:6443; do sleep 5; done",
      "REMOTE_TOKEN=$(ssh -o StrictHostKeyChecking=accept-new cluster@${var.master_ip} sudo cat /var/lib/rancher/k3s/server/node-token)",
      "curl -sfL https://get.k3s.io | K3S_URL=https://${var.master_ip}:6443 K3S_TOKEN=$REMOTE_TOKEN sh -"
    ]
  }
  template = {
    packages = ["curl"]
    users = [
      {
        name                = "cluster"
        ssh_authorized_keys = [for ssh_authorized_key in var.ssh_authorized_keys : trimspace(ssh_authorized_key)]
        sudo                = "ALL=(ALL) NOPASSWD:ALL"
        shell               = "/bin/bash"
      }
    ]
    write_files = [
      {
        path        = "/root/.ssh/id_rsa"
        content     = format("%s\n", trimspace(var.ssh_private_key))
        permissions = "0600"
      }
    ]
    runcmd = concat([
      "apt-get update -y",
      ],
    local.runcmds[var.node_type])
  }
}
