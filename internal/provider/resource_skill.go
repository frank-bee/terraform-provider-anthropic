package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
var _ resource.ResourceWithModifyPlan = &SkillResource{}

type SkillResource struct {
	baseResource
}

type SkillResourceModel struct {
	Id            types.String `tfsdk:"id"`
	DisplayTitle  types.String `tfsdk:"display_title"`
	SkillName     types.String `tfsdk:"skill_name"`
	Content       types.String `tfsdk:"content"`
	SourceDir     types.String `tfsdk:"source_dir"`
	SourceHash    types.String `tfsdk:"source_hash"`
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
		MarkdownDescription: "Manages a custom Anthropic Skill.\n\n" +
			"A skill consists of a `SKILL.md` file (with YAML frontmatter `name` and `description`) " +
			"and optionally additional files (scripts, references, etc.) under the same folder.\n\n" +
			"Use either `content` for a single-file SKILL.md, or `source_dir` to point at a directory " +
			"that contains SKILL.md plus any companion files.\n\n" +
			"When `content` or `source_dir` changes, a new skill version is uploaded — the skill keeps " +
			"its ID, so attached agents do not need to be re-attached.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the Skill.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_title": schema.StringAttribute{
				MarkdownDescription: "Display title for the Skill. Must be globally unique within the workspace.",
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
				MarkdownDescription: "Full content of SKILL.md including YAML frontmatter and markdown body. " +
					"Mutually exclusive with `source_dir`. Updating this attribute uploads a new skill version " +
					"in place (skill ID is preserved).\n\nExample:\n```\n---\nname: my-skill\ndescription: What this skill does\n---\n\n# Instructions\n...\n```",
				Optional: true,
			},
			"source_dir": schema.StringAttribute{
				MarkdownDescription: "Path to a directory containing `SKILL.md` and any additional files " +
					"(scripts, references, etc.) that should be packaged into the skill. The directory's contents " +
					"are zipped under the skill's folder. Mutually exclusive with `content`. Updating this attribute " +
					"(or any file beneath it) uploads a new skill version in place (skill ID is preserved).",
				Optional: true,
			},
			"source_hash": schema.StringAttribute{
				MarkdownDescription: "SHA-256 of the packaged skill contents. Computed automatically; used to detect " +
					"changes when `source_dir` is in use (since file contents are not stored in the state otherwise).",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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

// ModifyPlan computes the source_hash from content or source_dir so plan
// shows drift when underlying files change. It also enforces that exactly
// one of content/source_dir is set, and invalidates server-side computed
// fields (latest_version, updated_at) when an in-place update is planned.
func (r *SkillResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return // delete
	}

	var plan SkillResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasContent := !plan.Content.IsNull() && !plan.Content.IsUnknown()
	hasSourceDir := !plan.SourceDir.IsNull() && !plan.SourceDir.IsUnknown()

	if hasContent == hasSourceDir {
		resp.Diagnostics.AddError(
			"Invalid skill configuration",
			"Exactly one of `content` or `source_dir` must be set.",
		)
		return
	}

	files, err := loadSkillFiles(plan)
	if err != nil {
		resp.Diagnostics.AddError("Skill files error", err.Error())
		return
	}

	newHash := hashFiles(files)
	plan.SourceHash = types.StringValue(newHash)

	// On update, if the hash is changing, mark the API-computed fields as
	// unknown so terraform doesn't expect them to stay at their state values.
	if !req.State.Raw.IsNull() {
		var state SkillResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if state.SourceHash.ValueString() != newHash {
			plan.LatestVersion = types.StringUnknown()
			plan.UpdatedAt = types.StringUnknown()
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *SkillResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SkillResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	files, err := loadSkillFiles(data)
	if err != nil {
		resp.Diagnostics.AddError("Skill files error", err.Error())
		return
	}

	skill, err := r.skills.CreateSkillFromFiles(
		ctx,
		data.DisplayTitle.ValueString(),
		data.SkillName.ValueString(),
		files,
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
	data.SourceHash = types.StringValue(hashFiles(files))

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
	// skill_name, content, source_dir, source_hash are not returned by GET — preserve from state

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SkillResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SkillResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	files, err := loadSkillFiles(plan)
	if err != nil {
		resp.Diagnostics.AddError("Skill files error", err.Error())
		return
	}

	version, err := r.skills.CreateSkillVersionFromFiles(
		ctx,
		state.Id.ValueString(),
		state.SkillName.ValueString(),
		files,
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upload new skill version: %s", err))
		return
	}

	// Refresh top-level metadata (latest_version, updated_at) from the API
	skill, _, err := r.skills.GetSkill(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to refresh skill after update: %s", err))
		return
	}

	plan.Id = state.Id
	plan.Source = types.StringValue(skill.Source)
	plan.LatestVersion = types.StringValue(version.Version)
	plan.CreatedAt = types.StringValue(skill.CreatedAt)
	plan.UpdatedAt = types.StringValue(skill.UpdatedAt)
	plan.SourceHash = types.StringValue(hashFiles(files))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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

// loadSkillFiles returns the file map for either content (single SKILL.md)
// or source_dir (everything in the directory recursively).
func loadSkillFiles(m SkillResourceModel) (map[string][]byte, error) {
	if !m.Content.IsNull() && !m.Content.IsUnknown() {
		return map[string][]byte{"SKILL.md": []byte(m.Content.ValueString())}, nil
	}

	dir := m.SourceDir.ValueString()
	files := map[string][]byte{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		// Use forward slashes inside the zip (zip spec)
		rel = strings.ReplaceAll(rel, string(os.PathSeparator), "/")
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[rel] = data
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking source_dir %q: %w", dir, err)
	}
	if _, ok := files["SKILL.md"]; !ok {
		return nil, fmt.Errorf("source_dir %q must contain SKILL.md", dir)
	}
	return files, nil
}

// hashFiles returns a SHA-256 over the sorted list of (filename, content).
// Used to detect drift when source_dir contents change.
func hashFiles(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		h.Write([]byte(n))
		h.Write([]byte{0})
		h.Write(files[n])
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
