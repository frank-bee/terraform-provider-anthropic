resource "anthropic_memory_store" "example" {
  name        = "readme-updater state"
  description = "Tracks the last commit SHA the readme-updater agent has documented."

  metadata = {
    team = "docs"
  }

  # Hard-delete on `terraform destroy` instead of the default soft-delete
  # (archive). Archived memory stores are recoverable via the API; deleted
  # ones are not.
  delete_on_destroy = false
}
