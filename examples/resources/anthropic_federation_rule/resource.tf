# Bind GitHub Actions main-branch deploys to a service account.
resource "anthropic_federation_rule" "gha_deploy" {
  name      = "gha-deploy"
  issuer_id = anthropic_federation_issuer.github_actions.id

  match = {
    subject_prefix = "repo:my-org/my-repo:ref:refs/heads/main"
    claims = {
      repository_owner = "my-org"
    }
  }

  target = {
    service_account_id = anthropic_service_account.inference_worker.id
  }

  workspace_id           = "wrkspc_xxxxx"
  oauth_scope            = "workspace:developer"
  token_lifetime_seconds = 600
}
