# Terraform Provider Anthropic

[![Tests](https://github.com/frank-bee/terraform-provider-anthropic/actions/workflows/test.yml/badge.svg)](https://github.com/frank-bee/terraform-provider-anthropic/actions/workflows/test.yml)
[![Release](https://github.com/frank-bee/terraform-provider-anthropic/actions/workflows/release.yml/badge.svg)](https://github.com/frank-bee/terraform-provider-anthropic/actions/workflows/release.yml)
[![Registry](https://img.shields.io/badge/terraform-registry-blueviolet)](https://registry.terraform.io/providers/frank-bee/anthropic/latest)

The Anthropic Provider enables Terraform to manage Anthropic Managed Agents, Skills, and Workspaces.

> Fork of [jianyuan/terraform-provider-anthropic](https://github.com/jianyuan/terraform-provider-anthropic) with Managed Agents and Skills support.

## Resources

### Managed Agents API (Beta)
- `anthropic_agent` — Create and manage agents with models, tools, MCP servers, and skills
- `anthropic_skill` — Create and manage custom skills (SKILL.md with frontmatter + instructions)
- `anthropic_environment` — Create and manage agent environments with networking and packages

### Admin API
- `anthropic_workspace` — Create and manage workspaces
- `anthropic_workspace_member` — Manage workspace members
- `anthropic_organization_invite` — Manage organization invites

### Data Sources
- `anthropic_agents` — List all managed agents
- `anthropic_skills` — List all skills (Anthropic-provided + custom)
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

resource "anthropic_skill" "reviewer" {
  display_title = "Code Reviewer"
  skill_name    = "code-reviewer"
  content       = <<-EOT
---
name: code-reviewer
description: Reviews code for quality issues and security vulnerabilities
---

# Instructions
When asked to review code, check for security issues and best practices.
EOT
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

## Releasing

See [RELEASING.md](RELEASING.md).

## License

MIT — see [LICENSE](LICENSE).
