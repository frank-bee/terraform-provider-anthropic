data "anthropic_environments" "all" {}

output "environments" {
  value = data.anthropic_environments.all.environments
}
