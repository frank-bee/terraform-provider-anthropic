package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// withEnvironmentsBeta overrides the anthropic-beta header for /v1/environments
// requests. The provider-level editor sets agent-api-2026-03-01, but the
// environments endpoint lives behind managed-agents-2026-04-01.
func withEnvironmentsBeta(_ context.Context, req *http.Request) error {
	req.Header.Set("anthropic-beta", "managed-agents-2026-04-01")
	return nil
}

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
	pkgListAttr := func(manager string) schema.Attribute {
		return schema.ListAttribute{
			MarkdownDescription: fmt.Sprintf("%s packages to install in the environment.", manager),
			Optional:            true,
			ElementType:         types.StringType,
		}
	}

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
			"description": schema.StringAttribute{
				MarkdownDescription: "Free-form description of the Environment.",
				Optional:            true,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Free-form string metadata attached to the Environment.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"config_type": schema.StringAttribute{
				MarkdownDescription: "Configuration type. Currently only `cloud` is supported.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("cloud"),
			},
			"networking_type": schema.StringAttribute{
				MarkdownDescription: "Networking type. One of `unrestricted` or `limited`.",
				Required:            true,
			},
			"allow_mcp_servers": schema.BoolAttribute{
				MarkdownDescription: "Only meaningful when `networking_type = \"limited\"`. Allow the agent session to talk to MCP servers.",
				Optional:            true,
				Computed:            true,
			},
			"allow_package_managers": schema.BoolAttribute{
				MarkdownDescription: "Only meaningful when `networking_type = \"limited\"`. Allow package managers (apt, pip, npm, ...) to reach their upstream registries during init.",
				Optional:            true,
				Computed:            true,
			},
			"allowed_hosts": schema.ListAttribute{
				MarkdownDescription: "Only meaningful when `networking_type = \"limited\"`. Additional hostnames the agent session is allowed to reach.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"apt_packages":   pkgListAttr("apt"),
			"pip_packages":   pkgListAttr("pip"),
			"npm_packages":   pkgListAttr("npm"),
			"cargo_packages": pkgListAttr("cargo"),
			"gem_packages":   pkgListAttr("gem"),
			"go_packages":    pkgListAttr("go"),
			"init_script": schema.StringAttribute{
				MarkdownDescription: "Read-only. Shell script executed when each session boots. Currently cannot be set via API — use the Anthropic dashboard.",
				Computed:            true,
			},
			"environment": schema.MapAttribute{
				MarkdownDescription: "Read-only. Environment variables baked into every session. Currently cannot be set via API — use the Anthropic dashboard. Treated as sensitive.",
				Computed:            true,
				ElementType:         types.StringType,
				Sensitive:           true,
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

// buildConfig turns the plan data into an EnvironmentConfig for Create/Update.
// Returns ok=false if the plan couldn't be decoded — the caller has already
// appended to resp.Diagnostics.
func (r *EnvironmentResource) buildConfig(ctx context.Context, data *EnvironmentModel, diags *diag.Diagnostics) apiclient.EnvironmentConfig {
	config := apiclient.EnvironmentConfig{
		Type: data.ConfigType.ValueString(),
		Networking: apiclient.EnvironmentNetworking{
			Type: data.NetworkingType.ValueString(),
		},
	}

	// networking: limited-only fields. The API rejects them for unrestricted,
	// so only send them when the user set them explicitly.
	if !data.AllowMcpServers.IsNull() && !data.AllowMcpServers.IsUnknown() {
		b := data.AllowMcpServers.ValueBool()
		config.Networking.AllowMcpServers = &b
	}
	if !data.AllowPackageManagers.IsNull() && !data.AllowPackageManagers.IsUnknown() {
		b := data.AllowPackageManagers.ValueBool()
		config.Networking.AllowPackageManagers = &b
	}
	if !data.AllowedHosts.IsNull() && !data.AllowedHosts.IsUnknown() {
		var hosts []string
		diags.Append(data.AllowedHosts.ElementsAs(ctx, &hosts, false)...)
		if diags.HasError() {
			return config
		}
		config.Networking.AllowedHosts = &hosts
	}

	packages := apiclient.EnvironmentPackages{Type: "packages"}
	hasPackages := false

	for _, p := range []struct {
		field *types.List
		dst   **[]string
	}{
		{&data.AptPackages, &packages.Apt},
		{&data.PipPackages, &packages.Pip},
		{&data.NpmPackages, &packages.Npm},
		{&data.CargoPackages, &packages.Cargo},
		{&data.GemPackages, &packages.Gem},
		{&data.GoPackages, &packages.Go},
	} {
		if p.field.IsNull() || p.field.IsUnknown() {
			continue
		}
		var vals []string
		diags.Append(p.field.ElementsAs(ctx, &vals, false)...)
		if diags.HasError() {
			return config
		}
		*p.dst = &vals
		hasPackages = true
	}

	if hasPackages {
		config.Packages = &packages
	}

	return config
}

// buildTopLevelFields pulls description + metadata from the plan model.
func buildTopLevelFields(ctx context.Context, data *EnvironmentModel, diags *diag.Diagnostics) (*string, *map[string]string) {
	var desc *string
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		s := data.Description.ValueString()
		desc = &s
	}

	var meta *map[string]string
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		m := make(map[string]string)
		diags.Append(data.Metadata.ElementsAs(ctx, &m, false)...)
		if diags.HasError() {
			return desc, nil
		}
		meta = &m
	}

	return desc, meta
}

func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data EnvironmentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config := r.buildConfig(ctx, &data, &resp.Diagnostics)
	desc, meta := buildTopLevelFields(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	body := apiclient.CreateEnvironmentJSONRequestBody{
		Name:        data.Name.ValueString(),
		Config:      config,
		Description: desc,
		Metadata:    meta,
	}

	httpResp, err := r.client.CreateEnvironmentWithResponse(ctx, body, withEnvironmentsBeta)
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

	httpResp, err := r.client.GetEnvironmentWithResponse(ctx, data.Id.ValueString(), withEnvironmentsBeta)
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

	config := r.buildConfig(ctx, &data, &resp.Diagnostics)
	desc, meta := buildTopLevelFields(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	body := apiclient.UpdateEnvironmentJSONRequestBody{
		Name:        &name,
		Config:      &config,
		Description: desc,
		Metadata:    meta,
	}

	httpResp, err := r.client.UpdateEnvironmentWithResponse(ctx, data.Id.ValueString(), body, withEnvironmentsBeta)
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

	httpResp, err := r.client.DeleteEnvironmentWithResponse(ctx, data.Id.ValueString(), withEnvironmentsBeta)
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
