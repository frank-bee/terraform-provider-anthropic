package provider

import (
	"context"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EnvironmentModel struct {
	Id                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Metadata             types.Map    `tfsdk:"metadata"`
	ConfigType           types.String `tfsdk:"config_type"`
	NetworkingType       types.String `tfsdk:"networking_type"`
	AllowMcpServers      types.Bool   `tfsdk:"allow_mcp_servers"`
	AllowPackageManagers types.Bool   `tfsdk:"allow_package_managers"`
	AllowedHosts         types.List   `tfsdk:"allowed_hosts"`
	InitScript           types.String `tfsdk:"init_script"`
	Environment          types.Map    `tfsdk:"environment"`
	AptPackages          types.List   `tfsdk:"apt_packages"`
	PipPackages          types.List   `tfsdk:"pip_packages"`
	NpmPackages          types.List   `tfsdk:"npm_packages"`
	CargoPackages        types.List   `tfsdk:"cargo_packages"`
	GemPackages          types.List   `tfsdk:"gem_packages"`
	GoPackages           types.List   `tfsdk:"go_packages"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

// stringListOrNull converts an API list-of-strings into a types.List.
// Empty API lists are treated as null so they don't show up as drift
// against an unset optional attribute.
func stringListOrNull(p *[]string) types.List {
	if p == nil || len(*p) == 0 {
		return types.ListNull(types.StringType)
	}
	elems := make([]attr.Value, 0, len(*p))
	for _, s := range *p {
		elems = append(elems, types.StringValue(s))
	}
	return types.ListValueMust(types.StringType, elems)
}

func (m *EnvironmentModel) Fill(ctx context.Context, e apiclient.Environment) error {
	m.Id = types.StringValue(e.Id)
	m.Name = types.StringValue(e.Name)
	m.ConfigType = types.StringValue(e.Config.Type)
	m.NetworkingType = types.StringValue(e.Config.Networking.Type)
	m.CreatedAt = types.StringPointerValue(e.CreatedAt)
	m.UpdatedAt = types.StringPointerValue(e.UpdatedAt)

	// description: API returns "" rather than omitting the field, treat as null
	// so an unset attribute doesn't show as a permanent diff.
	if e.Description != nil && *e.Description != "" {
		m.Description = types.StringValue(*e.Description)
	} else {
		m.Description = types.StringNull()
	}

	if e.Metadata != nil && len(*e.Metadata) > 0 {
		elems := make(map[string]attr.Value, len(*e.Metadata))
		for k, v := range *e.Metadata {
			elems[k] = types.StringValue(v)
		}
		m.Metadata = types.MapValueMust(types.StringType, elems)
	} else {
		m.Metadata = types.MapNull(types.StringType)
	}

	// Networking. AllowMcpServers/AllowPackageManagers are Computed+Optional
	// so they must always be known after Read — fall back to false if the
	// API omitted them (unrestricted networking case).
	if e.Config.Networking.AllowMcpServers != nil {
		m.AllowMcpServers = types.BoolValue(*e.Config.Networking.AllowMcpServers)
	} else {
		m.AllowMcpServers = types.BoolValue(false)
	}
	if e.Config.Networking.AllowPackageManagers != nil {
		m.AllowPackageManagers = types.BoolValue(*e.Config.Networking.AllowPackageManagers)
	} else {
		m.AllowPackageManagers = types.BoolValue(false)
	}
	m.AllowedHosts = stringListOrNull(e.Config.Networking.AllowedHosts)

	// init_script / environment are read-only in the current API. Normalize
	// empty values to null so an unset attribute doesn't flap.
	if e.Config.InitScript != nil && *e.Config.InitScript != "" {
		m.InitScript = types.StringValue(*e.Config.InitScript)
	} else {
		m.InitScript = types.StringNull()
	}

	if e.Config.Environment != nil && len(*e.Config.Environment) > 0 {
		elems := make(map[string]attr.Value, len(*e.Config.Environment))
		for k, v := range *e.Config.Environment {
			elems[k] = types.StringValue(v)
		}
		m.Environment = types.MapValueMust(types.StringType, elems)
	} else {
		m.Environment = types.MapNull(types.StringType)
	}

	if e.Config.Packages != nil {
		m.AptPackages = stringListOrNull(e.Config.Packages.Apt)
		m.PipPackages = stringListOrNull(e.Config.Packages.Pip)
		m.NpmPackages = stringListOrNull(e.Config.Packages.Npm)
		m.CargoPackages = stringListOrNull(e.Config.Packages.Cargo)
		m.GemPackages = stringListOrNull(e.Config.Packages.Gem)
		m.GoPackages = stringListOrNull(e.Config.Packages.Go)
	} else {
		m.AptPackages = types.ListNull(types.StringType)
		m.PipPackages = types.ListNull(types.StringType)
		m.NpmPackages = types.ListNull(types.StringType)
		m.CargoPackages = types.ListNull(types.StringType)
		m.GemPackages = types.ListNull(types.StringType)
		m.GoPackages = types.ListNull(types.StringType)
	}

	return nil
}
