# Register GitHub Actions as an OIDC issuer (JWKS discovery).
resource "anthropic_federation_issuer" "github_actions" {
  name       = "github-actions"
  issuer_url = "https://token.actions.githubusercontent.com"

  jwks = {
    type = "discovery"
  }
}

# Explicit JWKS URL example:
# jwks = {
#   type = "explicit_url"
#   url  = "https://example.com/.well-known/jwks.json"
# }
#
# Inline keys (air-gapped issuers):
# jwks = {
#   type      = "inline"
#   keys_json = jsonencode([{ kty = "RSA", n = "...", e = "AQAB" }])
# }
