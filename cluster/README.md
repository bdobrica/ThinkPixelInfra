# Cluster Setup

This repository provides a Terraform configuration to deploy a cluster on Hetzner Cloud using the Hetzner provider (v~> 1.48). The configuration is organized into multiple modules that manage networking, load balancing, and the provisioning of master and worker nodes.

## Overview

The solution is composed of the following parts:

1. **Network Setup**: Creates a Hetzner network and subnet for the cluster. (See [Network Module Documentation](./infra/networking/README.md) for details.)
2. **Load Balancer**: Provisions a load balancer (with its associated network, services, and certificate) to front the cluster. (Refer to the [Load Balancer Module Documentation](./infra/load-balancer/README.md) for inputs and outputs.)
3. **Master and Worker Nodes**:
    1. **Master Module**: Deploys master nodes that manage the cluster. (See [Master Module Documentation](./infra/k8s/master/README.md) for further details.)
    2. **Worker Module**: Deploys worker nodes that run your workloads. (Refer to [Worker Module Documentation](./infra/k8s/worker/README.md) for more information.)
4. **User Data Modules**: Configure node initialization scripts for both master and worker nodes, ensuring they are set up with the proper configuration on boot. (Details can be found in the respective [user data module READMEs](./infra/user-data/README.md).)

## Prerequisites

Before using this configuration, ensure that you have:

- Terraform v1.x or higher installed.
- Terragrunt v0.31.8 or higher installed.
- A Hetzner Cloud account with an API token.
- SSH keys for management access, as well as for accessing master and worker nodes.
- The required variables (see below) configured either in variables.tf or via a terraform.tfvars file.

## Getting Started

### Clone the Repository

```bash
git clone https://github.com/bdobrica/ThinkPixelInfra.git
cd ThinkPixelInfra/cluster/live/dev
```

- There are two live environments: dev and prod. The dev environment is used for testing and development, while the prod environment is for production deployments.
- If you want to deploy to the prod environment, navigate to the `prod` directory instead.

### Create Required SSH Keys

Generate SSH keys for management, master, and worker nodes. For example:

```bash
ssh-keygen -t ecdsa -b 521 -C "master@<cluster_domain>" -f .master.key
ssh-keygen -t ecdsa -b 521 -C "worker@<cluster_domain>" -f .worker.key
ssh-keygen -t ecdsa -b 521 -C "management@<cluster_domain>" -f .management.key
```

- Replace `<cluster_domain>` with the domain name for your cluster (e.g., "cluster.local").
- The above commands will generate three key pairs: .master.key, .worker.key, and .management.key.
- The master key pair is used for master node access, the worker key pair for worker node access, and the management key pair for management access.

### Add Hetzner API Token

Create a .secrets.hcl file in the [cluster/live/{dev,prod}](./infra/live/dev) directory with the following content:

```hcl
inputs = {
    hcloud_token = "your-hetzner-api-token"
}
```

### Configure Variables

In the [cluster/live/dev](./live/dev) folder you'll have to update the terragrunt.hcl file with your custom values.

Example:

```hcl
terraform {
    source = find_in_parent_folders("infra")
}

# Include the secrets and SSH keys. If created as described above, the file paths should be correct.
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
        hcloud_token = "dummy" # no need to change; value will be overridden by secrets
        cluster_domain = "<cluster_domain>" # e.g., "cluster.local"
        cluster_prefix = "<cluster_prefix>" # e.g., "k8s", prefix for all nodes
        location = "<location>" # e.g., "fsn1"
        os_image = "<os_image>" # e.g., "debian-12"
        master_pool = { # there's only one master pool
            server_type = "<server_type>" # e.g., "cx21"
            count = <count> # e.g., 1; number of master nodes to create; must be multiple of 3
        }
        worker_pools = { # multiple worker pools can be defined, with different server types and counts
            <pool_name_01> = { # e.g., "pool01"
                server_type = "<server_type>" # e.g., "cax21"
                count = <count> # e.g., 2; number of worker nodes to create
            }
        }
    },
    local.secrets.inputs,
    local.ssh_keys
)
```

- Replace `<cluster_domain>` with the domain name for your cluster (e.g., "cluster.local").
- Replace `<cluster_prefix>` with a prefix for all nodes (e.g., "k8s"). Server names will be generated as `<cluster_prefix>-<node_type>-<index>`.
- Replace `<location>` with the location for the network (e.g., "fsn1"). Check the Hetzner Cloud documentation for available locations.
- Replace `<os_image>` with the OS image for the nodes (e.g., "debian-12"). Check the Hetzner Cloud documentation for available images.
- Replace `<server_type>` with the type of server to create (e.g., "cx21"). Check the Hetzner Cloud documentation for available server types.
- Replace `<count>` with the number of nodes to create for the master and worker pools.
- Add additional worker pools as needed or scale the existing ones by changing the count.

### Initialize Terraform

```bash
terragrunt init
```

### Plan and Apply the Configuration

```bash
terragrunt plan
terragrunt apply
```

## Modules Documentation

Each module included in this repository has been documented using terraform-docs. Below is a summary of each:

### Network Module

**Purpose**: Creates the Hetzner private network and associated subnet.

**Key Inputs**:
- `location`: The location for the network (default: "fsn1").
- `name`: The name of the cluster network.

**Key Outputs**:
- `network`: The created network resource.
- `subnet`: The created subnet within the network.

### Load Balancer Module

**Purpose**: Provisions a load balancer along with its networking configuration.

**Key Inputs**:
- `domain_names`: List of domain names the load balancer will listen on.
- `location`: The location (default: "fsn1").
- `name`: The load balancer’s name.
- `network` and `subnet`: Resources to which the load balancer is attached.
- `nodes`: Mapping of server names to server IDs.
- `protocols`: Protocols to be enabled (default: ["http", "https"]).

**Key Outputs**:
- `ip`: The load balancer's IP address.
- `load_balancer`: Resource details of the load balancer.

### Master Module

**Purpose**: Deploys master node pool for cluster management.

**Key Inputs**:
- `cluster_domain`: Cluster domain name (default: "cluster.local").
- `cluster_prefix`: Name prefix for all nodes.
- `management_public_ssh_key`: SSH key for management access.
- `master_pool`: Object defining the server type and count (default: one "cx22" server).
- `master_private_ssh_key` & `master_public_ssh_key`: SSH keys for master access.
- `network` and `subnet`: Resources for network attachment.
- `os_image`: OS image for master nodes (default: "debian-12").

**Key Outputs**:
- `master_ip`: The IP address of the master node.
- `master_nodes`: Details of the master nodes created.

### Worker Module

**Purpose**: Deploys worker nodes to run your workloads.

**Key Inputs**:
- `cluster_domain`: Cluster domain name (default: "cluster.local").
- `location`: Worker node location (default: "fsn1").
- `management_public_ssh_key`: SSH key for management.
- `master_ip`: The IP of the master node or load balancer.
- `name_prefix`: Name prefix for worker nodes (default: "worker").
- `network` and `subnet`: Networking details.
- `os_image`: OS image for worker nodes (default: "debian-12").
- `server_pools`: Mapping of worker pools (e.g., number and type of servers).
- `worker_private_ssh_key`: SSH key for worker access.

**Key Outputs**:
- `ip`: IP address allocated to worker nodes.
- `nodes`: Details of the worker nodes created.

### User Data Modules

These modules provide the necessary initialization scripts for both master and worker nodes. They ensure that the nodes are correctly configured during boot.

- **Master User Data**: Configures initialization for master nodes.
- **Replica/Worker User Data**: Configures initialization for worker nodes.

### Variables and Outputs

The main configuration exposes several key variables (defined in variables.tf):

**Inputs**:
- `cluster_domain`, `master_ip`, `node_type`
- `ssh_authorized_keys`, `ssh_private_key`

And additional module-specific variables (detailed in each module’s documentation).

**Outputs**:
- Cluster YAML configuration.
- Networking details (`network`, `subnet`).
- Master and worker node information.
- Load balancer IP and details.

## Hetzner Cloud Provider

This project uses the Hetzner Cloud provider (version ~> 1.48). Ensure your environment is set up with the proper API credentials to use these resources.
