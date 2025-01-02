## Requirements

No requirements.

## Providers

No providers.

## Modules

No modules.

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cluster_domain"></a> [cluster\_domain](#input\_cluster\_domain) | The domain name for the cluster | `string` | `"cluster.local"` | no |
| <a name="input_master_ip"></a> [master\_ip](#input\_master\_ip) | The IP address of the master node | `string` | `null` | no |
| <a name="input_node_type"></a> [node\_type](#input\_node\_type) | The type of node to create | `string` | n/a | yes |
| <a name="input_ssh_authorized_keys"></a> [ssh\_authorized\_keys](#input\_ssh\_authorized\_keys) | The SSH public keys to add to the cluster user | `list(string)` | `[]` | no |
| <a name="input_ssh_private_key"></a> [ssh\_private\_key](#input\_ssh\_private\_key) | The SSH private key to use for the cluster user | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_yaml"></a> [yaml](#output\_yaml) | n/a |
