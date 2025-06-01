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
        cluster_domain_aliases = [ "thinkpixel.io" ]
        cluster_prefix = "k8s"
        location = "fsn1"
        os_image = "debian-12"
        master_pool = {
            server_type = "cax21"
            count = 1
        }
        worker_pools = {
            api = {
                server_type = "cax21"
                count = 3
            }
            monitor = {
                server_type = "cax21"
                count = 1
            }
        }
        admin_ips = [ "135.181.209.167" ]
        exposed_apps = {
            "api" = {
                protocol = "https"
                port = 8080
                health_check = {
                    path = "/ping"
                }
            },
            "monitor" = {
                protocol = "https"
                port = 3000
                health_check = {
                    path = "/api/health"
                }
            }
        }
    },
    local.secrets.inputs,
    local.ssh_keys
)
