package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewAgentDataSource() datasource.DataSource {
	return &AgentDataSource{}
}

var _ datasource.DataSource = &AgentDataSource{}

type AgentDataSource struct {
	baseDataSource
}

func (d *AgentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

func (d *AgentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	toolConfigsAttr := schema.ListNestedAttribute{
		MarkdownDescription: "Per-tool overrides within the toolset.",
		Computed:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"name":    schema.StringAttribute{Computed: true, MarkdownDescription: "Name of the tool."},
				"enabled": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the tool is enabled."},
			},
		},
	}
	defaultConfigAttr := schema.SingleNestedAttribute{
		MarkdownDescription: "Default configuration applied to the toolset.",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the toolset is enabled by default."},
			"permission_policy": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Permission policy for the toolset.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{Computed: true, MarkdownDescription: "Type of the permission policy."},
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about a single Managed Agent by ID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the Agent.",
				Required:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Version of the Agent.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the Agent.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the Agent.",
				Computed:            true,
			},
			"system": schema.StringAttribute{
				MarkdownDescription: "System prompt of the Agent.",
				Computed:            true,
			},
			"model": schema.StringAttribute{
				MarkdownDescription: "Model ID of the Agent.",
				Computed:            true,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Arbitrary key-value metadata attached to the Agent.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"tools": schema.ListNestedAttribute{
				MarkdownDescription: "Toolsets configured on the Agent.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type":           schema.StringAttribute{Computed: true, MarkdownDescription: "Type of the toolset."},
						"default_config": defaultConfigAttr,
						"configs":        toolConfigsAttr,
					},
				},
			},
			"mcp_servers": schema.ListNestedAttribute{
				MarkdownDescription: "MCP servers configured on the Agent.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":           schema.StringAttribute{Computed: true, MarkdownDescription: "Name of the MCP server."},
						"type":           schema.StringAttribute{Computed: true, MarkdownDescription: "Type of the MCP server."},
						"url":            schema.StringAttribute{Computed: true, MarkdownDescription: "URL of the MCP server."},
						"default_config": defaultConfigAttr,
						"configs":        toolConfigsAttr,
					},
				},
			},
			"skills": schema.ListNestedAttribute{
				MarkdownDescription: "Skills attached to the Agent.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"skill_id": schema.StringAttribute{Computed: true, MarkdownDescription: "ID of the Skill."},
						"type":     schema.StringAttribute{Computed: true, MarkdownDescription: "Type of the Skill reference."},
						"version":  schema.StringAttribute{Computed: true, MarkdownDescription: "Version of the Skill."},
					},
				},
			},
		},
	}
}

func (d *AgentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AgentModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.client.GetAgentWithResponse(ctx, data.Id.ValueString(), withManagedAgentsBeta)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read agent, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read agent, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to read agent, got empty response body")
		return
	}

	if err := data.Fill(*httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fill data, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
