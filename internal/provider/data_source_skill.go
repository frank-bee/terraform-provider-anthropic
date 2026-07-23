package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewSkillDataSource() datasource.DataSource {
	return &SkillDataSource{}
}

var _ datasource.DataSource = &SkillDataSource{}

type SkillDataSource struct {
	baseDataSource
}

// SingleSkillDataSourceModel is the flat read model for a single skill. It is
// distinct from SkillDataSourceModel (used by the list data source) and from
// SkillModel (an agent's skill reference).
type SingleSkillDataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	DisplayTitle  types.String `tfsdk:"display_title"`
	Source        types.String `tfsdk:"source"`
	LatestVersion types.String `tfsdk:"latest_version"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (d *SkillDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_skill"
}

func (d *SkillDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about a single Skill by ID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the Skill.",
				Required:            true,
			},
			"display_title": schema.StringAttribute{
				MarkdownDescription: "Display title of the Skill.",
				Computed:            true,
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "Source of the Skill (`anthropic` or `custom`).",
				Computed:            true,
			},
			"latest_version": schema.StringAttribute{
				MarkdownDescription: "Latest version identifier of the Skill.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the Skill was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the Skill was last updated.",
				Computed:            true,
			},
		},
	}
}

func (d *SkillDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SingleSkillDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	skill, statusCode, err := d.skills.GetSkill(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read skill, got error: %s", err))
		return
	}
	if statusCode != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read skill, got status code: %d", statusCode))
		return
	}
	if skill == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to read skill, got empty response body")
		return
	}

	data.Id = types.StringValue(skill.Id)
	data.DisplayTitle = types.StringValue(skill.DisplayTitle)
	data.Source = types.StringValue(skill.Source)
	data.LatestVersion = types.StringValue(skill.LatestVersion)
	data.CreatedAt = types.StringValue(skill.CreatedAt)
	data.UpdatedAt = types.StringValue(skill.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
