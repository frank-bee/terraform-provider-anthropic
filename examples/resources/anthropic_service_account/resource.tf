# Requires the provider's oauth_token (an org:admin OAuth bearer token).
resource "anthropic_service_account" "inference_worker" {
  name              = "inference-worker"
  organization_role = "developer"
}
