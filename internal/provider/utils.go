package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mapFromTFMap converts a types.Map of string values into the pointer-to-map
// shape the API client expects. Returns nil when the Terraform value is null
// or unknown.
func mapFromTFMap(ctx context.Context, m types.Map, diags *diag.Diagnostics) *map[string]string {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}
	out := map[string]string{}
	d := m.ElementsAs(ctx, &out, false)
	diags.Append(d...)
	if d.HasError() {
		return nil
	}
	return &out
}

// buildAgentTools converts the Terraform tools + mcp_servers model into the
// flat slice of apiclient.AgentTool entries the API expects. For each MCP
// server an auto-generated `mcp_toolset` entry is appended.
func buildAgentTools(tools []AgentToolModel, servers []McpServerModel) []apiclient.AgentTool {
	var out []apiclient.AgentTool
	for _, t := range tools {
		at := apiclient.AgentTool{Type: t.Type.ValueString()}
		if dc := buildDefaultConfig(t.DefaultConfig); dc != nil {
			at.DefaultConfig = dc
		}
		out = append(out, at)
	}
	for _, s := range servers {
		at := apiclient.AgentTool{
			Type:          "mcp_toolset",
			McpServerName: ptrTo(s.Name.ValueString()),
		}
		if dc := buildDefaultConfig(s.DefaultConfig); dc != nil {
			at.DefaultConfig = dc
		}
		out = append(out, at)
	}
	return out
}

func buildDefaultConfig(m *AgentToolDefaultConfigModel) *apiclient.AgentToolDefaultConfig {
	if m == nil {
		return nil
	}
	out := &apiclient.AgentToolDefaultConfig{}
	if !m.Enabled.IsNull() && !m.Enabled.IsUnknown() {
		out.Enabled = m.Enabled.ValueBoolPointer()
	}
	if m.PermissionPolicy != nil && !m.PermissionPolicy.Type.IsNull() && !m.PermissionPolicy.Type.IsUnknown() {
		out.PermissionPolicy = &apiclient.AgentToolPermissionPolicy{
			Type: m.PermissionPolicy.Type.ValueString(),
		}
	}
	if out.Enabled == nil && out.PermissionPolicy == nil {
		return nil
	}
	return out
}

func BuildTwoPartId(a, b string) string {
	return fmt.Sprintf("%s/%s", a, b)
}

func ptrTo[T any](v T) *T {
	return &v
}

func SplitTwoPartId(id, a, b string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected %s/%s", id, a, b)
	}
	return parts[0], parts[1], nil
}
