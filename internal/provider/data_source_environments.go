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

func NewEnvironmentsDataSource() datasource.DataSource {
	return &EnvironmentsDataSource{}
}

var _ datasource.DataSource = &EnvironmentsDataSource{}

type EnvironmentsDataSource struct {
	baseDataSource
}

type EnvironmentsDataSourceModel struct {
	Environments []EnvironmentDataSourceModel `tfsdk:"environments"`
}

type EnvironmentDataSourceModel struct {
	Id             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	ConfigType     types.String `tfsdk:"config_type"`
	NetworkingType types.String `tfsdk:"networking_type"`
}

func (d *EnvironmentsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environments"
}

func (d *EnvironmentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List all Managed Agent Environments.",

		Attributes: map[string]schema.Attribute{
			"environments": schema.ListNestedAttribute{
				MarkdownDescription: "List of environments.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "ID of the Environment.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the Environment.",
							Computed:            true,
						},
						"config_type": schema.StringAttribute{
							MarkdownDescription: "Configuration type.",
							Computed:            true,
						},
						"networking_type": schema.StringAttribute{
							MarkdownDescription: "Networking type.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *EnvironmentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var allEnvs []EnvironmentDataSourceModel

	var page *string
	for {
		params := &apiclient.ListEnvironmentsParams{}
		if page != nil {
			params.Page = page
		}

		httpResp, err := d.client.ListEnvironmentsWithResponse(ctx, params, withEnvironmentsBeta)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list environments, got error: %s", err))
			return
		}

		if httpResp.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list environments, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
			return
		}

		if httpResp.JSON200 == nil {
			break
		}

		for _, e := range httpResp.JSON200.Data {
			allEnvs = append(allEnvs, EnvironmentDataSourceModel{
				Id:             types.StringValue(e.Id),
				Name:           types.StringValue(e.Name),
				ConfigType:     types.StringValue(e.Config.Type),
				NetworkingType: types.StringValue(e.Config.Networking.Type),
			})
		}

		if httpResp.JSON200.NextPage == nil || *httpResp.JSON200.NextPage == "" {
			break
		}
		page = httpResp.JSON200.NextPage
	}

	data.Environments = allEnvs
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
