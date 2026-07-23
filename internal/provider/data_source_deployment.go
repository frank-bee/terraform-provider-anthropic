package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewDeploymentDataSource() datasource.DataSource {
	return &DeploymentDataSource{}
}

var _ datasource.DataSource = &DeploymentDataSource{}

type DeploymentDataSource struct {
	baseDataSource
}

func (d *DeploymentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (d *DeploymentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about a single Deployment by ID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the Deployment.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the Deployment.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the Deployment.",
				Computed:            true,
			},
			"agent_id": schema.StringAttribute{
				MarkdownDescription: "ID of the Agent the Deployment runs.",
				Computed:            true,
			},
			"agent_version": schema.StringAttribute{
				MarkdownDescription: "Version of the Agent the Deployment runs.",
				Computed:            true,
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "ID of the Environment the Deployment runs in.",
				Computed:            true,
			},
			"vault_ids": schema.ListAttribute{
				MarkdownDescription: "IDs of the Vaults attached to the Deployment.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Arbitrary key-value metadata attached to the Deployment.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"initial_events": schema.ListAttribute{
				MarkdownDescription: "Initial events sent to the agent on each run, as JSON strings.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"resources": schema.ListAttribute{
				MarkdownDescription: "Resources made available to the Deployment, as JSON strings.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"schedule": schema.SingleNestedAttribute{
				MarkdownDescription: "Schedule on which the Deployment runs.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"expression": schema.StringAttribute{Computed: true, MarkdownDescription: "Cron expression for the schedule."},
					"timezone":   schema.StringAttribute{Computed: true, MarkdownDescription: "Timezone of the schedule."},
					"last_run_at": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "RFC 3339 datetime of the last run.",
					},
					"upcoming_runs_at": schema.ListAttribute{
						Computed:            true,
						MarkdownDescription: "RFC 3339 datetimes of upcoming runs.",
						ElementType:         types.StringType,
					},
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the Deployment.",
				Computed:            true,
			},
			"archived_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the Deployment was archived, or null if active.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the Deployment was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the Deployment was last updated.",
				Computed:            true,
			},
		},
	}
}

func (d *DeploymentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DeploymentModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.client.GetDeploymentWithResponse(ctx, data.Id.ValueString(), withManagedAgentsBeta)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read deployment, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read deployment, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to read deployment, got empty response body")
		return
	}

	if err := data.Fill(ctx, *httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fill data, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
