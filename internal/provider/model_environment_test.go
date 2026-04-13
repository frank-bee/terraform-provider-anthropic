package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
)

// TestEnvironmentModel_Fill_RealAPIShape verifies that Fill() can decode
// the exact JSON shape returned by GET /v1/environments/{id} today.
// The fixture was copied from a live API response — if this test breaks,
// the API schema changed and apiclient.Environment needs to follow.
func TestEnvironmentModel_Fill_RealAPIShape(t *testing.T) {
	fixture := []byte(`{
		"id": "env_01H18yMMD9cHEYy1HuzDjxDt",
		"type": "environment",
		"name": "qm-test-env",
		"description": "",
		"created_at": "2026-04-10T17:36:24.691394Z",
		"updated_at": "2026-04-10T19:04:34.343352Z",
		"archived_at": null,
		"state": "active",
		"config": {
			"type": "cloud",
			"packages": {
				"type": "packages",
				"pip": [],
				"npm": [],
				"apt": ["gh", "jq"],
				"cargo": [],
				"gem": [],
				"go": []
			},
			"networking": {"type": "unrestricted"},
			"init_script": "echo hi",
			"environment": {"SLACK_BOT_TOKEN": "xoxb-redacted"}
		},
		"metadata": {"owner": "sw-devops"}
	}`)

	var env apiclient.Environment
	if err := json.Unmarshal(fixture, &env); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	var m EnvironmentModel
	if err := m.Fill(context.Background(), env); err != nil {
		t.Fatalf("Fill: %v", err)
	}

	if got := m.Id.ValueString(); got != "env_01H18yMMD9cHEYy1HuzDjxDt" {
		t.Errorf("Id = %q", got)
	}
	if got := m.Name.ValueString(); got != "qm-test-env" {
		t.Errorf("Name = %q", got)
	}
	if !m.Description.IsNull() {
		t.Errorf("Description should be null when API returned \"\"")
	}
	if m.Metadata.IsNull() {
		t.Errorf("Metadata should be populated")
	}
	if got := m.NetworkingType.ValueString(); got != "unrestricted" {
		t.Errorf("NetworkingType = %q", got)
	}
	if !m.AllowedHosts.IsNull() {
		t.Errorf("AllowedHosts should be null when omitted")
	}
	if m.AllowMcpServers.ValueBool() {
		t.Errorf("AllowMcpServers should default to false for unrestricted")
	}
	if m.AllowPackageManagers.ValueBool() {
		t.Errorf("AllowPackageManagers should default to false for unrestricted")
	}
	if got := m.InitScript.ValueString(); got != "echo hi" {
		t.Errorf("InitScript = %q", got)
	}
	if m.Environment.IsNull() {
		t.Errorf("Environment should be populated")
	}
	if m.AptPackages.IsNull() {
		t.Errorf("AptPackages should be populated")
	}
	// Empty package lists should surface as null, not an empty list, so
	// plans don't flap against an unset optional attribute.
	if !m.PipPackages.IsNull() {
		t.Errorf("PipPackages should be null when API returned []")
	}
	if !m.NpmPackages.IsNull() {
		t.Errorf("NpmPackages should be null when API returned []")
	}
}

// TestEnvironmentModel_Fill_LimitedNetworking covers the allowed_hosts /
// allow_mcp_servers / allow_package_managers shape for
// networking.type = "limited".
func TestEnvironmentModel_Fill_LimitedNetworking(t *testing.T) {
	fixture := []byte(`{
		"id": "env_x",
		"type": "environment",
		"name": "limited",
		"config": {
			"type": "cloud",
			"networking": {
				"type": "limited",
				"allow_mcp_servers": false,
				"allow_package_managers": true,
				"allowed_hosts": ["api.github.com", "github.com"]
			}
		}
	}`)

	var env apiclient.Environment
	if err := json.Unmarshal(fixture, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var m EnvironmentModel
	if err := m.Fill(context.Background(), env); err != nil {
		t.Fatalf("Fill: %v", err)
	}

	if got := m.NetworkingType.ValueString(); got != "limited" {
		t.Errorf("NetworkingType = %q", got)
	}
	if m.AllowedHosts.IsNull() {
		t.Fatalf("AllowedHosts should be populated")
	}
	if n := len(m.AllowedHosts.Elements()); n != 2 {
		t.Errorf("AllowedHosts len = %d, want 2", n)
	}
	if m.AllowMcpServers.ValueBool() {
		t.Errorf("AllowMcpServers should be false")
	}
	if !m.AllowPackageManagers.ValueBool() {
		t.Errorf("AllowPackageManagers should be true")
	}
}
