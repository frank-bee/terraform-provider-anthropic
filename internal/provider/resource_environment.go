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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewEnvironmentResource() resource.Resource {
	return &EnvironmentResource{}
}

var _ resource.Resource = &EnvironmentResource{}
var _ resource.ResourceWithImportState = &EnvironmentResource{}

type EnvironmentResource struct {
	baseResource
}

func (r *EnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *EnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Anthropic Managed Agent Environment.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the Environment.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the Environment.",
				Required:            true,
			},
			"config_type": schema.StringAttribute{
				MarkdownDescription: "Configuration type. Currently only `cloud` is supported.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("cloud"),
			},
			"networking_type": schema.StringAttribute{
				MarkdownDescription: "Networking type. One of `unrestricted` or `restricted`.",
				Required:            true,
			},
			"packages": schema.MapAttribute{
				MarkdownDescription: "Package versions to install (e.g. `{\"python\" = \"3.12\", \"node\" = \"20\"}`).",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime string indicating when the Environment was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime string indicating when the Environment was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data EnvironmentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := apiclient.CreateEnvironmentJSONRequestBody{
		Name: data.Name.ValueString(),
		Config: apiclient.EnvironmentConfig{
			Type: data.ConfigType.ValueString(),
			Networking: apiclient.EnvironmentNetworking{
				Type: data.NetworkingType.ValueString(),
			},
		},
	}

	if !data.Packages.IsNull() && !data.Packages.IsUnknown() {
		packages := make(map[string]string)
		resp.Diagnostics.Append(data.Packages.ElementsAs(ctx, &packages, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body.Config.Packages = &packages
	}

	httpResp, err := r.client.CreateEnvironmentWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create environment, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create environment, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}

	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to create environment, got empty response body")
		return
	}

	if err := data.Fill(ctx, *httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fill data: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EnvironmentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.GetEnvironmentWithResponse(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
		return
	}

	if httpResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}

	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to read environment, got empty response body")
		return
	}

	if err := data.Fill(ctx, *httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fill data: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data EnvironmentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	config := apiclient.EnvironmentConfig{
		Type: data.ConfigType.ValueString(),
		Networking: apiclient.EnvironmentNetworking{
			Type: data.NetworkingType.ValueString(),
		},
	}

	if !data.Packages.IsNull() && !data.Packages.IsUnknown() {
		packages := make(map[string]string)
		resp.Diagnostics.Append(data.Packages.ElementsAs(ctx, &packages, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		config.Packages = &packages
	}

	body := apiclient.UpdateEnvironmentJSONRequestBody{
		Name:   &name,
		Config: &config,
	}

	httpResp, err := r.client.UpdateEnvironmentWithResponse(ctx, data.Id.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update environment, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update environment, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}

	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to update environment, got empty response body")
		return
	}

	if err := data.Fill(ctx, *httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fill data: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EnvironmentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.DeleteEnvironmentWithResponse(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete environment, got error: %s", err))
		return
	}

	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete environment, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
}

func (r *EnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
