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
