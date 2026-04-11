data "anthropic_agents" "all" {}

output "agents" {
  value = data.anthropic_agents.all.agents
}
