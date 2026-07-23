package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewFederationRuleResource() resource.Resource {
	return &FederationRuleResource{}
}

var _ resource.Resource = &FederationRuleResource{}
var _ resource.ResourceWithImportState = &FederationRuleResource{}
var _ resource.ResourceWithValidateConfig = &FederationRuleResource{}

type FederationRuleResource struct {
	baseResource
}

// ValidateConfig surfaces the "workspace_id or applies_to_all_workspaces is
// required" rule at plan time instead of only at apply. A stock validator can't
// express it because applies_to_all_workspaces is a bool: false is a set value,
// not "unset", so ExactlyOneOf/AtLeastOneOf on nullness would accept the invalid
// (workspace_id null, applies_to_all_workspaces false) combination. The Create
// method keeps the same check as a backstop.
func (r *FederationRuleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data FederationRuleModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Skip when either value is unknown (e.g. workspace_id references another
	// resource not yet created) — it can't be validated until apply.
	if data.WorkspaceId.IsUnknown() || data.AppliesToAllWorkspaces.IsUnknown() {
		return
	}
	if data.WorkspaceId.IsNull() && !data.AppliesToAllWorkspaces.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("workspace_id"),
			"workspace_id or applies_to_all_workspaces is required",
			"Set `workspace_id` to enable the rule in one workspace, or `applies_to_all_workspaces = true`.",
		)
	}
}

type FederationRuleMatchModel struct {
	SubjectPrefix types.String `tfsdk:"subject_prefix"`
	Audience      types.String `tfsdk:"audience"`
	Claims        types.Map    `tfsdk:"claims"`
	Condition     types.String `tfsdk:"condition"`
}

type FederationRuleTargetModel struct {
	Type             types.String `tfsdk:"type"`
	ServiceAccountId types.String `tfsdk:"service_account_id"`
}

type FederationRuleModel struct {
	Id                     types.String               `tfsdk:"id"`
	Name                   types.String               `tfsdk:"name"`
	IssuerId               types.String               `tfsdk:"issuer_id"`
	Match                  *FederationRuleMatchModel  `tfsdk:"match"`
	Target                 *FederationRuleTargetModel `tfsdk:"target"`
	WorkspaceId            types.String               `tfsdk:"workspace_id"`
	AppliesToAllWorkspaces types.Bool                 `tfsdk:"applies_to_all_workspaces"`
	OauthScope             types.String               `tfsdk:"oauth_scope"`
	TokenLifetimeSeconds   types.Int64                `tfsdk:"token_lifetime_seconds"`
	CreatedAt              types.String               `tfsdk:"created_at"`
	ArchivedAt             types.String               `tfsdk:"archived_at"`
}

func (m *FederationRuleMatchModel) toAPI(ctx context.Context) (apiclient.FederationRuleMatch, error) {
	out := apiclient.FederationRuleMatch{}
	if !m.SubjectPrefix.IsNull() && !m.SubjectPrefix.IsUnknown() {
		out.SubjectPrefix = m.SubjectPrefix.ValueStringPointer()
	}
	if !m.Audience.IsNull() && !m.Audience.IsUnknown() {
		out.Audience = m.Audience.ValueStringPointer()
	}
	if !m.Condition.IsNull() && !m.Condition.IsUnknown() {
		out.Condition = m.Condition.ValueStringPointer()
	}
	if !m.Claims.IsNull() && !m.Claims.IsUnknown() {
		claims := map[string]string{}
		if diags := m.Claims.ElementsAs(ctx, &claims, false); diags.HasError() {
			return out, fmt.Errorf("invalid claims map")
		}
		out.Claims = &claims
	}
	return out, nil
}

func (m *FederationRuleModel) Fill(ctx context.Context, fr apiclient.FederationRule) error {
	m.Id = types.StringValue(fr.Id)
	m.Name = types.StringValue(fr.Name)
	m.IssuerId = types.StringValue(fr.IssuerId)
	m.CreatedAt = types.StringPointerValue(fr.CreatedAt)
	m.ArchivedAt = types.StringPointerValue(fr.ArchivedAt)
	m.OauthScope = types.StringPointerValue(fr.OauthScope)

	if fr.AppliesToAllWorkspaces != nil {
		m.AppliesToAllWorkspaces = types.BoolValue(*fr.AppliesToAllWorkspaces)
	} else {
		m.AppliesToAllWorkspaces = types.BoolNull()
	}

	if fr.TokenLifetimeSeconds != nil {
		m.TokenLifetimeSeconds = types.Int64Value(int64(*fr.TokenLifetimeSeconds))
	} else {
		m.TokenLifetimeSeconds = types.Int64Null()
	}

	if fr.Match != nil {
		mm := &FederationRuleMatchModel{
			SubjectPrefix: types.StringPointerValue(fr.Match.SubjectPrefix),
			Audience:      types.StringPointerValue(fr.Match.Audience),
			Condition:     types.StringPointerValue(fr.Match.Condition),
			Claims:        types.MapNull(types.StringType),
		}
		if fr.Match.Claims != nil && len(*fr.Match.Claims) > 0 {
			cv, diags := types.MapValueFrom(ctx, types.StringType, *fr.Match.Claims)
			if diags.HasError() {
				return fmt.Errorf("failed to build claims map")
			}
			mm.Claims = cv
		}
		m.Match = mm
	}

	if fr.Target != nil {
		m.Target = &FederationRuleTargetModel{
			Type:             types.StringValue(fr.Target.Type),
			ServiceAccountId: types.StringValue(fr.Target.ServiceAccountId),
		}
	}
	// WorkspaceId is a create-only input not echoed by the API; leave it as-is.
	return nil
}

func (r *FederationRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_federation_rule"
}

func (r *FederationRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Workload Identity Federation rule (`fdrl_…`) — binds an issuer's JWTs " +
			"(matched by subject/audience/claims/CEL) to a service account and OAuth scope.\n\n" +
			"Requires the provider's `oauth_token` (an `org:admin` OAuth bearer token). OAuth callers may only " +
			"manage rules scoped `workspace:developer` or `workspace:inference`.\n\n" +
			"~> **Experimental (beta).** The Workload Identity Federation endpoints are not exercised by " +
			"the provider's CI acceptance tests, which run with an Admin API key only (no `org:admin` OAuth " +
			"token). Treat this resource as beta and verify its behavior in your own organization.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the federation rule (`fdrl_…`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the rule. Must match `^[a-z0-9-]+$`, 1–255 chars, unique within the organization.",
				Required:            true,
			},
			"issuer_id": schema.StringAttribute{
				MarkdownDescription: "ID of the federation issuer (`fdis_…`) this rule applies to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"match": schema.SingleNestedAttribute{
				MarkdownDescription: "Conditions an incoming JWT must satisfy. At least one of `subject_prefix`, `claims`, or `condition` must be set; all configured matchers must pass.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"subject_prefix": schema.StringAttribute{
						MarkdownDescription: "Match on the `sub` claim. Exact match unless it ends in `*` (prefix match).",
						Optional:            true,
					},
					"audience": schema.StringAttribute{
						MarkdownDescription: "Exact `aud` claim to require.",
						Optional:            true,
					},
					"claims": schema.MapAttribute{
						MarkdownDescription: "Map of exact claim values the JWT must carry.",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"condition": schema.StringAttribute{
						MarkdownDescription: "A [CEL](https://cel.dev/) expression for complex match logic.",
						Optional:            true,
					},
				},
			},
			"target": schema.SingleNestedAttribute{
				MarkdownDescription: "The service account a matched JWT maps to.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "Target type. Defaults to `service_account`.",
						Optional:            true,
						Computed:            true,
					},
					"service_account_id": schema.StringAttribute{
						MarkdownDescription: "ID of the target service account (`svac_…`).",
						Required:            true,
					},
				},
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Workspace the rule is enabled in at creation. Either this or `applies_to_all_workspaces` is required. Changing forces replacement.",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"applies_to_all_workspaces": schema.BoolAttribute{
				MarkdownDescription: "Enable the rule in all workspaces instead of a single `workspace_id`. Changing forces replacement (the API has no in-place update for this field).",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"oauth_scope": schema.StringAttribute{
				MarkdownDescription: "OAuth scope granted on the minted token. Defaults to `workspace:developer`. OAuth callers may only set `workspace:developer` or `workspace:inference`.",
				Optional:            true,
				Computed:            true,
			},
			"token_lifetime_seconds": schema.Int64Attribute{
				MarkdownDescription: "Lifetime of the minted token in seconds (60–86400). Defaults to 3600.",
				Optional:            true,
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the rule was created.",
				Computed:            true,
			},
			"archived_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the rule was archived, or null if active.",
				Computed:            true,
			},
		},
	}
}

func (r *FederationRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FederationRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := r.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}

	appliesToAll := data.AppliesToAllWorkspaces.ValueBool()
	if data.WorkspaceId.IsNull() && !appliesToAll {
		resp.Diagnostics.AddError(
			"workspace_id or applies_to_all_workspaces is required",
			"Set `workspace_id` to enable the rule in one workspace, or `applies_to_all_workspaces = true`.",
		)
		return
	}

	match, err := data.Match.toAPI(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Config Error", err.Error())
		return
	}

	body := apiclient.CreateFederationRuleJSONRequestBody{
		Name:     data.Name.ValueString(),
		IssuerId: data.IssuerId.ValueString(),
		Match:    match,
		Target: apiclient.FederationRuleTarget{
			Type:             targetTypeOrDefault(data.Target),
			ServiceAccountId: data.Target.ServiceAccountId.ValueString(),
		},
	}
	if !data.WorkspaceId.IsNull() {
		body.WorkspaceId = data.WorkspaceId.ValueStringPointer()
	}
	if !data.AppliesToAllWorkspaces.IsNull() && !data.AppliesToAllWorkspaces.IsUnknown() {
		body.AppliesToAllWorkspaces = data.AppliesToAllWorkspaces.ValueBoolPointer()
	}
	if !data.OauthScope.IsNull() && !data.OauthScope.IsUnknown() {
		body.OauthScope = data.OauthScope.ValueStringPointer()
	}
	if !data.TokenLifetimeSeconds.IsNull() && !data.TokenLifetimeSeconds.IsUnknown() {
		v := int(data.TokenLifetimeSeconds.ValueInt64())
		body.TokenLifetimeSeconds = &v
	}

	httpResp, err := r.client.CreateFederationRuleWithResponse(ctx, body, editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create federation rule, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create federation rule, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to create federation rule, got empty response body")
		return
	}

	if err := data.Fill(ctx, *httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FederationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FederationRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := r.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}

	httpResp, err := r.client.GetFederationRuleWithResponse(ctx, data.Id.ValueString(), editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read federation rule, got error: %s", err))
		return
	}
	if httpResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read federation rule, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to read federation rule, got empty response body")
		return
	}

	if err := data.Fill(ctx, *httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FederationRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FederationRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := r.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}

	match, err := data.Match.toAPI(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Config Error", err.Error())
		return
	}
	name := data.Name.ValueString()
	body := apiclient.UpdateFederationRuleJSONRequestBody{
		Name:  &name,
		Match: &match,
		Target: &apiclient.FederationRuleTarget{
			Type:             targetTypeOrDefault(data.Target),
			ServiceAccountId: data.Target.ServiceAccountId.ValueString(),
		},
	}
	if !data.OauthScope.IsNull() && !data.OauthScope.IsUnknown() {
		body.OauthScope = data.OauthScope.ValueStringPointer()
	}
	if !data.TokenLifetimeSeconds.IsNull() && !data.TokenLifetimeSeconds.IsUnknown() {
		v := int(data.TokenLifetimeSeconds.ValueInt64())
		body.TokenLifetimeSeconds = &v
	}

	httpResp, err := r.client.UpdateFederationRuleWithResponse(ctx, data.Id.ValueString(), body, editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update federation rule, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update federation rule, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to update federation rule, got empty response body")
		return
	}

	if err := data.Fill(ctx, *httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FederationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FederationRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := r.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}

	httpResp, err := r.client.ArchiveFederationRuleWithResponse(ctx, data.Id.ValueString(), editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive federation rule, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK && httpResp.StatusCode() != http.StatusNotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive federation rule, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
}

func (r *FederationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// targetTypeOrDefault returns the configured target type or the "service_account"
// default when the caller left it unset.
func targetTypeOrDefault(t *FederationRuleTargetModel) string {
	if t != nil && !t.Type.IsNull() && !t.Type.IsUnknown() && t.Type.ValueString() != "" {
		return t.Type.ValueString()
	}
	return "service_account"
}
