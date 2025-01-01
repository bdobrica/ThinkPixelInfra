## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_hcloud"></a> [hcloud](#requirement\_hcloud) | ~> 1.48 |

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_k8s"></a> [k8s](#module\_k8s) | ./k8s | n/a |
| <a name="module_network"></a> [network](#module\_network) | ./networking | n/a |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cluster_domain"></a> [cluster\_domain](#input\_cluster\_domain) | The domain name of the cluster | `string` | `"cluster.local"` | no |
| <a name="input_cluster_prefix"></a> [cluster\_prefix](#input\_cluster\_prefix) | The name prefix for all nodes. It will have the format <cluster\_prefix>-<node\_prefix>-<index> | `string` | `"k8s"` | no |
| <a name="input_hcloud_token"></a> [hcloud\_token](#input\_hcloud\_token) | Value of the Hetzner Cloud API token | `any` | n/a | yes |
| <a name="input_location"></a> [location](#input\_location) | The location to create the worker nodes | `string` | `"fsn1"` | no |
| <a name="input_management_public_ssh_key"></a> [management\_public\_ssh\_key](#input\_management\_public\_ssh\_key) | The public SSH key to use for management access | `string` | n/a | yes |
| <a name="input_master_pool"></a> [master\_pool](#input\_master\_pool) | The master pool to create | <pre>object({<br/>    server_type = string<br/>    count       = number<br/>  })</pre> | <pre>{<br/>  "count": 1,<br/>  "server_type": "cx22"<br/>}</pre> | no |
| <a name="input_master_private_ssh_key"></a> [master\_private\_ssh\_key](#input\_master\_private\_ssh\_key) | The private SSH key to use for master node access | `string` | n/a | yes |
| <a name="input_master_public_ssh_key"></a> [master\_public\_ssh\_key](#input\_master\_public\_ssh\_key) | The public SSH key to use for master node access | `string` | n/a | yes |
| <a name="input_os_image"></a> [os\_image](#input\_os\_image) | The OS image to use for the servers | `string` | `"debian-12"` | no |
| <a name="input_worker_pools"></a> [worker\_pools](#input\_worker\_pools) | The worker pools to create | <pre>map(object({<br/>    server_type = string<br/>    count       = number<br/>  }))</pre> | <pre>{<br/>  "pool1": {<br/>    "count": 3,<br/>    "server_type": "cx22"<br/>  }<br/>}</pre> | no |
| <a name="input_worker_private_ssh_key"></a> [worker\_private\_ssh\_key](#input\_worker\_private\_ssh\_key) | The private SSH key to use for worker node access | `string` | n/a | yes |
| <a name="input_worker_public_ssh_key"></a> [worker\_public\_ssh\_key](#input\_worker\_public\_ssh\_key) | The public SSH key to use for worker node access | `string` | n/a | yes |

## Outputs

No outputs.
