# Basic agent with a model
resource "anthropic_agent" "basic" {
  name  = "my-assistant"
  model = "claude-sonnet-4-5"
}

# Agent with system prompt and tools
resource "anthropic_agent" "with_tools" {
  name   = "devops-agent"
  model  = "claude-sonnet-4-5"
  system = "You are a DevOps assistant that helps with infrastructure tasks."

  tools {
    type = "agent_toolset_20251212"
  }
}

# Agent with MCP server
resource "anthropic_agent" "with_mcp" {
  name  = "mcp-agent"
  model = "claude-sonnet-4-5"

  tools {
    type = "agent_toolset_20251212"
  }

  mcp_servers {
    name = "my-server"
    type = "url"
    url  = "https://mcp.example.com/sse"
  }
}

# Agent that pins permission policies so unattended (fire-and-forget) sessions
# don't stall on `requires_action`. By default the API returns
# `permission_policy.type = "always_ask"` for each tool, which causes
# unattended sessions to block waiting for human approval.
resource "anthropic_agent" "with_permission_policy" {
  name        = "unattended-agent"
  model       = "claude-sonnet-4-5"
  description = "Headless analyzer used in CI."

  metadata = {
    owner = "devops"
    env   = "ci"
  }

  tools {
    type = "agent_toolset_20260401"

    default_config {
      enabled = true
      permission_policy {
        type = "always_allow"
      }
    }
  }

  mcp_servers {
    name = "internal-tools"
    type = "url"
    url  = "https://mcp.example.com/internal/mcp"

    # Mirrors `default_config` onto the auto-generated `mcp_toolset` entry
    # for this server so the same policy applies.
    default_config {
      enabled = true
      permission_policy {
        type = "always_allow"
      }
    }
  }
}

# Agent with skills
resource "anthropic_agent" "with_skills" {
  name  = "skilled-agent"
  model = "claude-opus-4-5"

  skills {
    skill_id = "computer_use"
    type     = "anthropic"
    version  = "1.0"
  }
}
