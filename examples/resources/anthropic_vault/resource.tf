resource "anthropic_vault" "example" {
  display_name = "readme-updater credentials"

  metadata = {
    team = "platform"
    env  = "prod"
  }

  # Hard-delete the Vault on `terraform destroy` instead of the default
  # archive-only behavior.
  delete_on_destroy = true
}
