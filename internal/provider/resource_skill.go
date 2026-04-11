package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewSkillResource() resource.Resource {
	return &SkillResource{}
}

var _ resource.Resource = &SkillResource{}
var _ resource.ResourceWithImportState = &SkillResource{}

type SkillResource struct {
	baseResource
}

type SkillResourceModel struct {
	Id            types.String `tfsdk:"id"`
	DisplayTitle  types.String `tfsdk:"display_title"`
	SkillName     types.String `tfsdk:"skill_name"`
	Content       types.String `tfsdk:"content"`
	Source        types.String `tfsdk:"source"`
	LatestVersion types.String `tfsdk:"latest_version"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *SkillResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_skill"
}

func (r *SkillResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom Anthropic Skill.\n\nSkills are uploaded as a SKILL.md file with YAML frontmatter containing `name` and `description`, followed by markdown instructions.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the Skill.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_title": schema.StringAttribute{
				MarkdownDescription: "Display title for the Skill.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"skill_name": schema.StringAttribute{
				MarkdownDescription: "Name of the skill (must match the `name` field in SKILL.md frontmatter). Max 64 chars, lowercase letters/numbers/hyphens only.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "Full content of SKILL.md including YAML frontmatter and markdown body.\n\nExample:\n```\n---\nname: my-skill\ndescription: What this skill does\n---\n\n# Instructions\n...\n```",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "Source of the skill (`custom` or `anthropic`).",
				Computed:            true,
			},
			"latest_version": schema.StringAttribute{
				MarkdownDescription: "Latest version identifier of the skill.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the skill was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the skill was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *SkillResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SkillResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	skill, err := r.skills.CreateSkill(
		ctx,
		data.DisplayTitle.ValueString(),
		data.SkillName.ValueString(),
		data.Content.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create skill: %s", err))
		return
	}

	data.Id = types.StringValue(skill.Id)
	data.Source = types.StringValue(skill.Source)
	data.LatestVersion = types.StringValue(skill.LatestVersion)
	data.CreatedAt = types.StringValue(skill.CreatedAt)
	data.UpdatedAt = types.StringValue(skill.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SkillResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SkillResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	skill, statusCode, err := r.skills.GetSkill(ctx, data.Id.ValueString())
	if err != nil {
		if statusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read skill: %s", err))
		return
	}

	data.Id = types.StringValue(skill.Id)
	data.DisplayTitle = types.StringValue(skill.DisplayTitle)
	data.Source = types.StringValue(skill.Source)
	data.LatestVersion = types.StringValue(skill.LatestVersion)
	data.CreatedAt = types.StringValue(skill.CreatedAt)
	data.UpdatedAt = types.StringValue(skill.UpdatedAt)
	// skill_name and content are not returned by GET — preserve from state

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SkillResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All mutable attributes have RequiresReplace, so Update should never be called.
	resp.Diagnostics.AddError("Internal Error", "Update called but all attributes require replacement")
}

func (r *SkillResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SkillResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.skills.DeleteSkillWithVersions(ctx, data.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete skill: %s", err))
		return
	}
}

func (r *SkillResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
