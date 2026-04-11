# Basic environment with unrestricted networking
resource "anthropic_environment" "basic" {
  name            = "dev-environment"
  networking_type = "unrestricted"
}

# Restricted environment with packages
resource "anthropic_environment" "with_packages" {
  name            = "python-env"
  networking_type = "restricted"
  packages = {
    "python" = "3.12"
    "node"   = "20"
  }
}
