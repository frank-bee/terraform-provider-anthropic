package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewServiceAccountResource() resource.Resource {
	return &ServiceAccountResource{}
}

var _ resource.Resource = &ServiceAccountResource{}
var _ resource.ResourceWithImportState = &ServiceAccountResource{}

type ServiceAccountResource struct {
	baseResource
}

type ServiceAccountModel struct {
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	OrganizationRole types.String `tfsdk:"organization_role"`
	CreatedAt        types.String `tfsdk:"created_at"`
	ArchivedAt       types.String `tfsdk:"archived_at"`
}

func (m *ServiceAccountModel) Fill(sa apiclient.ServiceAccount) {
	m.Id = types.StringValue(sa.Id)
	m.Name = types.StringValue(sa.Name)
	m.OrganizationRole = types.StringPointerValue(sa.OrganizationRole)
	m.CreatedAt = types.StringPointerValue(sa.CreatedAt)
	m.ArchivedAt = types.StringPointerValue(sa.ArchivedAt)
}

func (r *ServiceAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (r *ServiceAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Anthropic service account (`svac_…`) — the non-human identity that " +
			"[Workload Identity Federation](https://platform.claude.com/docs/en/manage-claude/workload-identity-federation) " +
			"tokens act as.\n\n" +
			"Requires the provider's `oauth_token` (an `org:admin` OAuth bearer token); the Admin API key " +
			"cannot access these endpoints.\n\n" +
			"~> **Experimental (beta).** The Workload Identity Federation endpoints are not exercised by " +
			"the provider's CI acceptance tests, which run with an Admin API key only (no `org:admin` OAuth " +
			"token). Treat this resource as beta and verify its behavior in your own organization.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the service account (`svac_…`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the service account. Must match `^[a-z0-9-]+$`, 1–255 chars, unique within the organization.",
				Required:            true,
			},
			"organization_role": schema.StringAttribute{
				MarkdownDescription: "Organization role of the service account (e.g. `developer`, or `admin` for a bootstrap `org:admin` rule target).",
				Optional:            true,
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the service account was created.",
				Computed:            true,
			},
			"archived_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the service account was archived, or null if active.",
				Computed:            true,
			},
		},
	}
}

func (r *ServiceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServiceAccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := r.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}

	body := apiclient.CreateServiceAccountJSONRequestBody{Name: data.Name.ValueString()}
	if !data.OrganizationRole.IsNull() && !data.OrganizationRole.IsUnknown() {
		body.OrganizationRole = data.OrganizationRole.ValueStringPointer()
	}

	httpResp, err := r.client.CreateServiceAccountWithResponse(ctx, body, editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service account, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service account, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to create service account, got empty response body")
		return
	}

	data.Fill(*httpResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServiceAccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := r.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}

	httpResp, err := r.client.GetServiceAccountWithResponse(ctx, data.Id.ValueString(), editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service account, got error: %s", err))
		return
	}
	if httpResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service account, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to read service account, got empty response body")
		return
	}

	data.Fill(*httpResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ServiceAccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := r.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}

	body := apiclient.UpdateServiceAccountJSONRequestBody{
		Name: data.Name.ValueStringPointer(),
	}
	if !data.OrganizationRole.IsNull() && !data.OrganizationRole.IsUnknown() {
		body.OrganizationRole = data.OrganizationRole.ValueStringPointer()
	}

	httpResp, err := r.client.UpdateServiceAccountWithResponse(ctx, data.Id.ValueString(), body, editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service account, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service account, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to update service account, got empty response body")
		return
	}

	data.Fill(*httpResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServiceAccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := r.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}

	httpResp, err := r.client.ArchiveServiceAccountWithResponse(ctx, data.Id.ValueString(), editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive service account, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK && httpResp.StatusCode() != http.StatusNotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive service account, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
}

func (r *ServiceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
