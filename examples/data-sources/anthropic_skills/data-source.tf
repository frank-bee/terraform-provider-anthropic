# List all available skills (both Anthropic-provided and custom)
data "anthropic_skills" "all" {}

output "skills" {
  value = data.anthropic_skills.all.skills
}
