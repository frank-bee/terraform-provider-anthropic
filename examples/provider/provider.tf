# Configure the Anthropic provider
provider "anthropic" {
  # Admin API key for standard resources (or set ANTHROPIC_API_KEY).
  api_key = "sk-ant-adminxx-xxxxx-xxxxx-xxxxx-xxxxx"

  # An org:admin OAuth bearer token (or set ANTHROPIC_OAUTH_TOKEN). Required
  # only by the Workload Identity Federation resources (anthropic_service_account,
  # anthropic_federation_issuer, anthropic_federation_rule), which the Admin API
  # key cannot access. Obtain via `ant auth print-credentials --access-token`.
  oauth_token = "sk-ant-oat01-xxxxx-xxxxx"
}

# Create a new workspace
resource "anthropic_workspace" "example" {
  name = "Workspace Name"
}

# Retrieve a user by ID
data "anthropic_user" "example" {
  id = "user_xxxxx"
}

# Add a user to the workspace
resource "anthropic_workspace_member" "example" {
  workspace_id   = anthropic_workspace.example.id
  user_id        = data.anthropic_user.example.id
  workspace_role = "workspace_developer"
}
