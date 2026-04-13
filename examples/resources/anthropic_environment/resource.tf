# Minimal environment with unrestricted networking.
resource "anthropic_environment" "basic" {
  name            = "dev-environment"
  networking_type = "unrestricted"
}

# Environment with apt packages (installed when the session boots).
resource "anthropic_environment" "with_apt" {
  name            = "gh-tools"
  networking_type = "unrestricted"
  apt_packages    = ["gh", "jq"]
}

# Limited-networking environment that can only reach a pinned host list and
# package-manager upstreams.
resource "anthropic_environment" "limited" {
  name                   = "ci-runner"
  networking_type        = "limited"
  allow_package_managers = true
  allowed_hosts          = ["api.github.com", "github.com"]
  apt_packages           = ["gh"]

  description = "Used by the CI failure-analysis agent"
  metadata = {
    owner = "sw-devops"
  }
}
