## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_hcloud"></a> [hcloud](#requirement\_hcloud) | ~> 1.48 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_hcloud"></a> [hcloud](#provider\_hcloud) | ~> 1.48 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_load_balancer"></a> [load\_balancer](#module\_load\_balancer) | ../../load-balancer | n/a |
| <a name="module_worker_user_data"></a> [worker\_user\_data](#module\_worker\_user\_data) | ../../user-data | n/a |

## Resources

| Name | Type |
|------|------|
| [hcloud_server.this](https://registry.terraform.io/providers/hetznercloud/hcloud/latest/docs/resources/server) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cluster_domain"></a> [cluster\_domain](#input\_cluster\_domain) | The domain name of the cluster | `string` | `"cluster.local"` | no |
| <a name="input_location"></a> [location](#input\_location) | The location to create the worker nodes | `string` | `"fsn1"` | no |
| <a name="input_management_public_ssh_key"></a> [management\_public\_ssh\_key](#input\_management\_public\_ssh\_key) | The public SSH key to use for management access | `string` | n/a | yes |
| <a name="input_master_ip"></a> [master\_ip](#input\_master\_ip) | The IP address of the master node or load balancer to master nodes | `any` | n/a | yes |
| <a name="input_name_prefix"></a> [name\_prefix](#input\_name\_prefix) | The name prefix of the worker nodes. It will have the format <name\_prefix>-<index> | `string` | `"worker"` | no |
| <a name="input_network"></a> [network](#input\_network) | The the network resource to attach the master nodes to | `any` | n/a | yes |
| <a name="input_os_image"></a> [os\_image](#input\_os\_image) | The OS image to use for the worker nodes | `string` | `"debian-12"` | no |
| <a name="input_server_pools"></a> [server\_pools](#input\_server\_pools) | The server pools to use as worker nodes | <pre>map(object({<br/>    server_type = string<br/>    count = number<br/>  }))</pre> | <pre>{<br/>  "pool1": {<br/>    "count": 3,<br/>    "server_type": "cx22"<br/>  }<br/>}</pre> | no |
| <a name="input_subnet"></a> [subnet](#input\_subnet) | Adding servers to the network requires a subnet. Not used, but required as a dependency | `any` | n/a | yes |
| <a name="input_worker_private_ssh_key"></a> [worker\_private\_ssh\_key](#input\_worker\_private\_ssh\_key) | The private SSH key to use for worker node access | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_ip"></a> [ip](#output\_ip) | n/a |
| <a name="output_nodes"></a> [nodes](#output\_nodes) | n/a |
