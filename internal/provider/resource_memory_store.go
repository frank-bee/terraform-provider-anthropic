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

func NewMemoryStoreResource() resource.Resource {
	return &MemoryStoreResource{}
}

var _ resource.Resource = &MemoryStoreResource{}
var _ resource.ResourceWithImportState = &MemoryStoreResource{}

type MemoryStoreResource struct {
	baseResource
}

// MemoryStoreModel is the Terraform representation of an
// apiclient.MemoryStore. DeleteOnDestroy is provider-only — it's never sent
// to the API and never returned by it; Fill must not touch it.
type MemoryStoreModel struct {
	Id              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Metadata        types.Map    `tfsdk:"metadata"`
	DeleteOnDestroy types.Bool   `tfsdk:"delete_on_destroy"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
	ArchivedAt      types.String `tfsdk:"archived_at"`
}

func (m *MemoryStoreModel) Fill(ms apiclient.MemoryStore) {
	m.Id = types.StringValue(ms.Id)
	m.Name = types.StringValue(ms.Name)
	m.Description = types.StringPointerValue(ms.Description)

	if ms.Metadata != nil && len(*ms.Metadata) > 0 {
		elems := make(map[string]attr.Value, len(*ms.Metadata))
		for k, v := range *ms.Metadata {
			elems[k] = types.StringValue(v)
		}
		m.Metadata = types.MapValueMust(types.StringType, elems)
	} else {
		m.Metadata = types.MapNull(types.StringType)
	}

	m.CreatedAt = types.StringPointerValue(ms.CreatedAt)
	m.UpdatedAt = types.StringPointerValue(ms.UpdatedAt)
	m.ArchivedAt = types.StringPointerValue(ms.ArchivedAt)
}

func (r *MemoryStoreResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_memory_store"
}

func (r *MemoryStoreResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Anthropic Managed Agents Memory Store (`memstore_…`) — a persistent " +
			"key-value-ish resource that can be mounted into Deployment sessions for cross-session state.\n\n" +
			"By default `terraform destroy` archives the Memory Store (soft delete, recoverable via the API) " +
			"rather than hard-deleting it. Set `delete_on_destroy = true` to hard-delete instead.\n\n" +
			"~> **Experimental (beta).** This resource's wire format was derived from live-API probing rather " +
			"than published documentation and may change. Treat it as beta.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the Memory Store (`memstore_…`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the Memory Store.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of what the Memory Store holds.",
				Optional:            true,
				Computed:            true,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Free-form string metadata attached to the Memory Store.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"delete_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Provider-only setting (never sent to or read from the API). When `false` " +
					"(the default), `terraform destroy` archives the Memory Store instead of hard-deleting it. " +
					"Set to `true` to hard-delete on destroy.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime string indicating when the Memory Store was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime string indicating when the Memory Store was last updated.",
				Computed:            true,
			},
			"archived_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime string indicating when the Memory Store was archived, if ever.",
				Computed:            true,
			},
		},
	}
}

func (r *MemoryStoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MemoryStoreModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := apiclient.CreateMemoryStoreJSONRequestBody{Name: data.Name.ValueString()}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body.Description = ptrTo(data.Description.ValueString())
	}
	body.Metadata = mapFromTFMap(ctx, data.Metadata, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.CreateMemoryStoreWithResponse(ctx, body, withManagedAgentsBeta)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create memory store, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create memory store, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}

	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to create memory store, got empty response body")
		return
	}

	data.Fill(*httpResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MemoryStoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MemoryStoreModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.GetMemoryStoreWithResponse(ctx, data.Id.ValueString(), withManagedAgentsBeta)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read memory store, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read memory store, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}

	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to read memory store, got empty response body")
		return
	}

	data.Fill(*httpResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MemoryStoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MemoryStoreModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := apiclient.UpdateMemoryStoreJSONRequestBody{Name: ptrTo(data.Name.ValueString())}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body.Description = ptrTo(data.Description.ValueString())
	}
	body.Metadata = mapFromTFMap(ctx, data.Metadata, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.UpdateMemoryStoreWithResponse(ctx, data.Id.ValueString(), body, withManagedAgentsBeta)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update memory store, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update memory store, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}

	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to update memory store, got empty response body")
		return
	}

	data.Fill(*httpResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete archives the Memory Store by default (soft delete). Set
// delete_on_destroy = true in config to hard-delete instead. Either a
// successful archive/delete or a 404 (already gone) is treated as success.
func (r *MemoryStoreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MemoryStoreModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.DeleteOnDestroy.IsNull() && data.DeleteOnDestroy.ValueBool() {
		httpResp, err := r.client.DeleteMemoryStoreWithResponse(ctx, data.Id.ValueString(), withManagedAgentsBeta)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete memory store, got error: %s", err))
			return
		}
		if httpResp.StatusCode() != http.StatusOK && httpResp.StatusCode() != http.StatusNotFound {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete memory store, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
			return
		}
		return
	}

	httpResp, err := r.client.ArchiveMemoryStoreWithResponse(ctx, data.Id.ValueString(), withManagedAgentsBeta)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive memory store, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK && httpResp.StatusCode() != http.StatusNotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive memory store, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
}

func (r *MemoryStoreResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
