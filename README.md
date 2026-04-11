# Terraform Provider Anthropic

Terraform provider for Anthropic — manages Admin API resources (workspaces, members, invites) and **Managed Agents API** resources (agents, environments).

> Fork of [jianyuan/terraform-provider-anthropic](https://github.com/jianyuan/terraform-provider-anthropic) with Managed Agents support added.

## Resources

### Admin API
- `anthropic_workspace` — Create and manage workspaces
- `anthropic_workspace_member` — Manage workspace members
- `anthropic_organization_invite` — Manage organization invites

### Managed Agents API (Beta)
- `anthropic_agent` — Create and manage agents with models, tools, MCP servers, and skills
- `anthropic_environment` — Create and manage agent environments with networking and packages

### Data Sources
- `anthropic_agents` — List all managed agents
- `anthropic_environments` — List all environments
- `anthropic_workspace`, `anthropic_workspaces`, `anthropic_user`, `anthropic_users`, etc.

## Usage

```hcl
terraform {
  required_providers {
    anthropic = {
      source = "frank-bee/anthropic"
    }
  }
}

provider "anthropic" {
  # Set via ANTHROPIC_API_KEY env var
}

resource "anthropic_agent" "assistant" {
  name  = "my-assistant"
  model = "claude-sonnet-4-5"

  tools {
    type = "agent_toolset_20251212"
  }
}

resource "anthropic_environment" "dev" {
  name            = "dev-environment"
  networking_type = "unrestricted"
  packages = {
    "python" = "3.12"
  }
}
```

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22

## Building

```shell
go install
```

## Testing

```shell
# Unit tests
make test

# Acceptance tests (requires ANTHROPIC_API_KEY)
make testacc
```

## License

MIT — see [LICENSE](LICENSE).
