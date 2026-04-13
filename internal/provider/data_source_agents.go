package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewAgentsDataSource() datasource.DataSource {
	return &AgentsDataSource{}
}

var _ datasource.DataSource = &AgentsDataSource{}

type AgentsDataSource struct {
	baseDataSource
}

type AgentsDataSourceModel struct {
	Agents []AgentDataSourceModel `tfsdk:"agents"`
}

type AgentDataSourceModel struct {
	Id      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Model   types.String `tfsdk:"model"`
	Version types.String `tfsdk:"version"`
}

func (d *AgentsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agents"
}

func (d *AgentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List all Managed Agents.",

		Attributes: map[string]schema.Attribute{
			"agents": schema.ListNestedAttribute{
				MarkdownDescription: "List of agents.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "ID of the Agent.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the Agent.",
							Computed:            true,
						},
						"model": schema.StringAttribute{
							MarkdownDescription: "Model ID of the Agent.",
							Computed:            true,
						},
						"version": schema.StringAttribute{
							MarkdownDescription: "Version of the Agent.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *AgentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AgentsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var allAgents []AgentDataSourceModel

	var page *string
	for {
		params := &apiclient.ListAgentsParams{}
		if page != nil {
			params.Page = page
		}

		httpResp, err := d.client.ListAgentsWithResponse(ctx, params)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list agents, got error: %s", err))
			return
		}

		if httpResp.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list agents, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
			return
		}

		if httpResp.JSON200 == nil {
			break
		}

		for _, a := range httpResp.JSON200.Data {
			model := types.StringNull()
			if a.Model != nil {
				model = types.StringValue(a.Model.Id)
			}
			allAgents = append(allAgents, AgentDataSourceModel{
				Id:      types.StringValue(a.Id),
				Name:    types.StringValue(a.Name),
				Model:   model,
				Version: types.StringValue(a.Version),
			})
		}

		if httpResp.JSON200.NextPage == nil || *httpResp.JSON200.NextPage == "" {
			break
		}
		page = httpResp.JSON200.NextPage
	}

	data.Agents = allAgents
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
