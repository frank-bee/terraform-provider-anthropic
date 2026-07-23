package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

var _ datasource.DataSource = &EnvironmentDataSource{}

type EnvironmentDataSource struct {
	baseDataSource
}

func (d *EnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *EnvironmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	strList := func(desc string) schema.ListAttribute {
		return schema.ListAttribute{MarkdownDescription: desc, Computed: true, ElementType: types.StringType}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about a single Managed Agent Environment by ID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the Environment.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the Environment.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the Environment.",
				Computed:            true,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Arbitrary key-value metadata attached to the Environment.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"config_type": schema.StringAttribute{
				MarkdownDescription: "Configuration type of the Environment.",
				Computed:            true,
			},
			"networking_type": schema.StringAttribute{
				MarkdownDescription: "Networking type of the Environment.",
				Computed:            true,
			},
			"allow_mcp_servers": schema.BoolAttribute{
				MarkdownDescription: "Whether MCP servers are reachable from the Environment.",
				Computed:            true,
			},
			"allow_package_managers": schema.BoolAttribute{
				MarkdownDescription: "Whether package managers are reachable from the Environment.",
				Computed:            true,
			},
			"allowed_hosts": strList("Hosts allowed for network egress from the Environment."),
			"init_script": schema.StringAttribute{
				MarkdownDescription: "Init script run when the Environment starts.",
				Computed:            true,
			},
			"environment": schema.MapAttribute{
				MarkdownDescription: "Environment variables set inside the Environment.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"apt_packages":   strList("apt packages installed in the Environment."),
			"pip_packages":   strList("pip packages installed in the Environment."),
			"npm_packages":   strList("npm packages installed in the Environment."),
			"cargo_packages": strList("cargo packages installed in the Environment."),
			"gem_packages":   strList("gem packages installed in the Environment."),
			"go_packages":    strList("go packages installed in the Environment."),
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the Environment was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the Environment was last updated.",
				Computed:            true,
			},
		},
	}
}

func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.client.GetEnvironmentWithResponse(ctx, data.Id.ValueString(), withManagedAgentsBeta)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to read environment, got empty response body")
		return
	}

	if err := data.Fill(ctx, *httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fill data, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
