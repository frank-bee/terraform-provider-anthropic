package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewSkillsDataSource() datasource.DataSource {
	return &SkillsDataSource{}
}

var _ datasource.DataSource = &SkillsDataSource{}

type SkillsDataSource struct {
	baseDataSource
}

type SkillsDataSourceModel struct {
	Skills []SkillDataSourceModel `tfsdk:"skills"`
}

type SkillDataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	DisplayTitle  types.String `tfsdk:"display_title"`
	Source        types.String `tfsdk:"source"`
	LatestVersion types.String `tfsdk:"latest_version"`
}

func (d *SkillsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_skills"
}

func (d *SkillsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List all available Skills (both Anthropic-provided and custom).",

		Attributes: map[string]schema.Attribute{
			"skills": schema.ListNestedAttribute{
				MarkdownDescription: "List of skills.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "ID of the Skill.",
							Computed:            true,
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
							MarkdownDescription: "Latest version of the Skill.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *SkillsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SkillsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.skills.ListSkills(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list skills: %s", err))
		return
	}

	skills := make([]SkillDataSourceModel, len(result.Data))
	for i, s := range result.Data {
		skills[i] = SkillDataSourceModel{
			Id:            types.StringValue(s.Id),
			DisplayTitle:  types.StringValue(s.DisplayTitle),
			Source:        types.StringValue(s.Source),
			LatestVersion: types.StringValue(s.LatestVersion),
		}
	}

	data.Skills = skills
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
