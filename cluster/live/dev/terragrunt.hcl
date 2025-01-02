terraform {
    source = find_in_parent_folders("infra")
}

locals {
    secrets = read_terragrunt_config(".secrets.hcl")
    ssh_keys = {
        master_public_ssh_key = file(".master.key.pub")
        master_private_ssh_key = file(".master.key")
        worker_public_ssh_key = file(".worker.key.pub")
        worker_private_ssh_key = file(".worker.key")
        management_public_ssh_key = file(".management.key.pub")
    }
}

inputs = merge(
    {
        hcloud_token = "dummy"
        cluster_domain = "dev.thinkpixel.io"
        cluster_prefix = "k8s"
        location = "fsn1"
        os_image = "debian-12"
        master_pool = {
            server_type = "cx22"
            count = 1
        }
        worker_pools = {
            pool1 = {
                server_type = "cx22"
                count = 2
            }
        }
    },
    local.secrets.inputs,
    local.ssh_keys
)
