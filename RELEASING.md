# Releasing

Releases are automated via GitHub Actions and GoReleaser. Pushing a semver tag triggers the pipeline.

## Release Process

1. **Ensure main is green** — check that the [Tests workflow](https://github.com/frank-bee/terraform-provider-anthropic/actions/workflows/test.yml) passes on main.

2. **Tag the release**:
   ```bash
   git checkout main
   git pull
   git tag v0.2.0
   git push origin v0.2.0
   ```

3. **Pipeline runs automatically**:
   - **Test job** — builds, lints, runs acceptance tests against the real Anthropic API
   - **GoReleaser job** (runs only after tests pass) — cross-compiles for all platforms, signs checksums with GPG, creates the GitHub Release with assets

4. **Verify** at https://github.com/frank-bee/terraform-provider-anthropic/releases

5. **Terraform Registry** picks up new releases automatically once the provider is published.

## Versioning

Follow [Semantic Versioning](https://semver.org/):
- **Patch** (`v0.1.1`) — bug fixes, doc updates
- **Minor** (`v0.2.0`) — new resources, data sources, attributes
- **Major** (`v1.0.0`) — breaking changes (removed attributes, renamed resources)

## Prerequisites

These are already configured in the repository:

| Secret | Purpose |
|--------|---------|
| `ANTHROPIC_API_KEY` | Acceptance tests against real API |
| `GPG_PRIVATE_KEY` | Signs release checksums |
| `PASSPHRASE` | GPG key passphrase (empty) |

## Troubleshooting

### Release fails with "already_exists"
A previous failed release left partial assets. Delete the stale release first:
```bash
gh release delete v0.x.y --repo frank-bee/terraform-provider-anthropic --yes
git push origin --delete v0.x.y
git push origin v0.x.y
```

### Tests fail in release pipeline
The release job (`goreleaser`) depends on the test job. If tests fail, no release is created. Fix the tests on main first, then re-tag.
