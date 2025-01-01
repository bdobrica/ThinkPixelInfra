## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_hcloud"></a> [hcloud](#requirement\_hcloud) | ~> 1.48 |

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_master"></a> [master](#module\_master) | ./master | n/a |
| <a name="module_worker"></a> [worker](#module\_worker) | ./worker | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cluster_domain"></a> [cluster\_domain](#input\_cluster\_domain) | The domain name of the cluster | `string` | `"cluster.local"` | no |
| <a name="input_cluster_prefix"></a> [cluster\_prefix](#input\_cluster\_prefix) | The name prefix for all nodes. It will have the format <cluster\_prefix>-<node\_prefix>-<index> | `string` | `"k8s"` | no |
| <a name="input_location"></a> [location](#input\_location) | The location to create the worker nodes | `string` | `"fsn1"` | no |
| <a name="input_management_public_ssh_key"></a> [management\_public\_ssh\_key](#input\_management\_public\_ssh\_key) | The public SSH key to use for management access | `string` | n/a | yes |
| <a name="input_master_pool"></a> [master\_pool](#input\_master\_pool) | The master pool to create | <pre>object({<br/>    server_type = string<br/>    count       = number<br/>  })</pre> | <pre>{<br/>  "count": 1,<br/>  "server_type": "cx22"<br/>}</pre> | no |
| <a name="input_master_private_ssh_key"></a> [master\_private\_ssh\_key](#input\_master\_private\_ssh\_key) | The private SSH key to use for master node access | `string` | n/a | yes |
| <a name="input_master_public_ssh_key"></a> [master\_public\_ssh\_key](#input\_master\_public\_ssh\_key) | The public SSH key to use for master node access | `string` | n/a | yes |
| <a name="input_network"></a> [network](#input\_network) | The the network resource to attach the master nodes to | `any` | n/a | yes |
| <a name="input_os_image"></a> [os\_image](#input\_os\_image) | The OS image to use for the worker nodes | `string` | `"debian-12"` | no |
| <a name="input_subnet"></a> [subnet](#input\_subnet) | Adding servers to the network requires a subnet. Not used, but required as a dependency | `any` | n/a | yes |
| <a name="input_worker_pools"></a> [worker\_pools](#input\_worker\_pools) | The worker pools to create | <pre>map(object({<br/>    server_type = string<br/>    count       = number<br/>  }))</pre> | <pre>{<br/>  "pool1": {<br/>    "count": 3,<br/>    "server_type": "cx22"<br/>  }<br/>}</pre> | no |
| <a name="input_worker_private_ssh_key"></a> [worker\_private\_ssh\_key](#input\_worker\_private\_ssh\_key) | The private SSH key to use for worker node access | `string` | n/a | yes |
| <a name="input_worker_public_ssh_key"></a> [worker\_public\_ssh\_key](#input\_worker\_public\_ssh\_key) | The public SSH key to use for worker node access | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_master_ip"></a> [master\_ip](#output\_master\_ip) | n/a |
| <a name="output_master_nodes"></a> [master\_nodes](#output\_master\_nodes) | n/a |
| <a name="output_worker_ip"></a> [worker\_ip](#output\_worker\_ip) | n/a |
| <a name="output_worker_nodes"></a> [worker\_nodes](#output\_worker\_nodes) | n/a |
