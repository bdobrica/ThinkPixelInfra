## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_hcloud"></a> [hcloud](#requirement\_hcloud) | ~> 1.48 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_hcloud"></a> [hcloud](#provider\_hcloud) | ~> 1.48 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [hcloud_load_balancer.this](https://registry.terraform.io/providers/hetznercloud/hcloud/latest/docs/resources/load_balancer) | resource |
| [hcloud_load_balancer_network.this](https://registry.terraform.io/providers/hetznercloud/hcloud/latest/docs/resources/load_balancer_network) | resource |
| [hcloud_load_balancer_service.this](https://registry.terraform.io/providers/hetznercloud/hcloud/latest/docs/resources/load_balancer_service) | resource |
| [hcloud_load_balancer_target.this](https://registry.terraform.io/providers/hetznercloud/hcloud/latest/docs/resources/load_balancer_target) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_location"></a> [location](#input\_location) | The location of the load balancer | `string` | `"fsn1"` | no |
| <a name="input_name"></a> [name](#input\_name) | The name of the load balancer | `any` | n/a | yes |
| <a name="input_network"></a> [network](#input\_network) | The network to attach the load balancer to | `any` | n/a | yes |
| <a name="input_nodes"></a> [nodes](#input\_nodes) | A mapping of server names to server IDs | `map(string)` | n/a | yes |
| <a name="input_protocols"></a> [protocols](#input\_protocols) | The list of protocols to listen on | `set(string)` | <pre>[<br/>  "http",<br/>  "https"<br/>]</pre> | no |
| <a name="input_subnet"></a> [subnet](#input\_subnet) | The subnet to attach the load balancer to | `any` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_ip"></a> [ip](#output\_ip) | n/a |
| <a name="output_load_balancer"></a> [load\_balancer](#output\_load\_balancer) | n/a |
