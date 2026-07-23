package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewVaultResource() resource.Resource {
	return &VaultResource{}
}

var _ resource.Resource = &VaultResource{}
var _ resource.ResourceWithImportState = &VaultResource{}

type VaultResource struct {
	baseResource
}

// VaultModel is the Terraform representation of an Anthropic Vault
// (`vlt_…`) — a container for stored credentials Deployments can reference
// via `vault_ids`.
type VaultModel struct {
	Id              types.String `tfsdk:"id"`
	DisplayName     types.String `tfsdk:"display_name"`
	Metadata        types.Map    `tfsdk:"metadata"`
	DeleteOnDestroy types.Bool   `tfsdk:"delete_on_destroy"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
	ArchivedAt      types.String `tfsdk:"archived_at"`
}

// Fill populates the API-backed fields of the model from the API's Vault
// representation. DeleteOnDestroy is provider-only (never sent to or
// returned by the API) and is intentionally left untouched here — Create
// picks it up from the plan (with its default applied), and Read/Update
// preserve whatever value is already in state.
func (m *VaultModel) Fill(v apiclient.Vault) {
	m.Id = types.StringValue(v.Id)
	m.DisplayName = types.StringValue(v.DisplayName)

	if v.Metadata != nil && len(*v.Metadata) > 0 {
		elems := make(map[string]attr.Value, len(*v.Metadata))
		for k, val := range *v.Metadata {
			elems[k] = types.StringValue(val)
		}
		m.Metadata = types.MapValueMust(types.StringType, elems)
	} else {
		m.Metadata = types.MapNull(types.StringType)
	}

	m.CreatedAt = types.StringPointerValue(v.CreatedAt)
	m.UpdatedAt = types.StringPointerValue(v.UpdatedAt)
	m.ArchivedAt = types.StringPointerValue(v.ArchivedAt)
}

func (r *VaultResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault"
}

func (r *VaultResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Anthropic Vault (`vlt_…`) — a container for stored credentials that " +
			"Deployments can reference via `vault_ids`.\n\n" +
			"~> **Experimental (beta).** This resource's wire format was derived from live-API probing rather " +
			"than published documentation and may change. Treat it as beta.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the Vault (`vlt_…`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the Vault.",
				Required:            true,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Free-form string metadata attached to the Vault.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"delete_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Whether `terraform destroy` hard-deletes the Vault instead of archiving " +
					"it. This is a provider-only setting — it is never sent to or read from the API. Defaults " +
					"to `false` (archive).",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime string indicating when the Vault was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime string indicating when the Vault was last updated.",
				Computed:            true,
			},
			"archived_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime string indicating when the Vault was archived, if ever.",
				Computed:            true,
			},
		},
	}
}

func (r *VaultResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VaultModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := apiclient.CreateVaultJSONRequestBody{
		DisplayName: data.DisplayName.ValueString(),
	}
	body.Metadata = mapFromTFMap(ctx, data.Metadata, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteOnDestroy := data.DeleteOnDestroy

	httpResp, err := r.client.CreateVaultWithResponse(ctx, body, withManagedAgentsBeta)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create vault, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create vault, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}

	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to create vault, got empty response body")
		return
	}

	data.Fill(*httpResp.JSON200)
	data.DeleteOnDestroy = deleteOnDestroy

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VaultResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VaultModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.GetVaultWithResponse(ctx, data.Id.ValueString(), withManagedAgentsBeta)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read vault, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read vault, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}

	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to read vault, got empty response body")
		return
	}

	data.Fill(*httpResp.JSON200)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VaultResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VaultModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := apiclient.UpdateVaultJSONRequestBody{
		DisplayName: data.DisplayName.ValueStringPointer(),
	}
	body.Metadata = mapFromTFMap(ctx, data.Metadata, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.UpdateVaultWithResponse(ctx, data.Id.ValueString(), body, withManagedAgentsBeta)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update vault, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update vault, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}

	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to update vault, got empty response body")
		return
	}

	deleteOnDestroy := data.DeleteOnDestroy
	data.Fill(*httpResp.JSON200)
	data.DeleteOnDestroy = deleteOnDestroy

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete hard-deletes the Vault when `delete_on_destroy = true`, otherwise
// archives it (the default). Both 200 and 404 (already gone) are treated as
// success.
func (r *VaultResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VaultModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.DeleteOnDestroy.ValueBool() {
		httpResp, err := r.client.DeleteVaultWithResponse(ctx, data.Id.ValueString(), withManagedAgentsBeta)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete vault, got error: %s", err))
			return
		}
		if httpResp.StatusCode() != http.StatusOK && httpResp.StatusCode() != http.StatusNotFound {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete vault, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
			return
		}
		return
	}

	httpResp, err := r.client.ArchiveVaultWithResponse(ctx, data.Id.ValueString(), withManagedAgentsBeta)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive vault, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK && httpResp.StatusCode() != http.StatusNotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive vault, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
}

// ImportState imports by ID. delete_on_destroy is provider-only and not
// derivable from the API, so it takes on its schema default (false, i.e.
// archive) after import.
func (r *VaultResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
