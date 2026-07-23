package provider

import (
	"context"
	"encoding/json"
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

func NewFederationIssuerResource() resource.Resource {
	return &FederationIssuerResource{}
}

var _ resource.Resource = &FederationIssuerResource{}
var _ resource.ResourceWithImportState = &FederationIssuerResource{}

type FederationIssuerResource struct {
	baseResource
}

type FederationIssuerJwksModel struct {
	Type     types.String `tfsdk:"type"`
	Url      types.String `tfsdk:"url"`
	KeysJson types.String `tfsdk:"keys_json"`
}

type FederationIssuerModel struct {
	Id         types.String               `tfsdk:"id"`
	Name       types.String               `tfsdk:"name"`
	IssuerUrl  types.String               `tfsdk:"issuer_url"`
	Jwks       *FederationIssuerJwksModel `tfsdk:"jwks"`
	CreatedAt  types.String               `tfsdk:"created_at"`
	ArchivedAt types.String               `tfsdk:"archived_at"`
}

// toAPI builds the wire jwks payload from the model.
func (j *FederationIssuerJwksModel) toAPI() (apiclient.FederationIssuerJwks, error) {
	out := apiclient.FederationIssuerJwks{Type: j.Type.ValueString()}
	if !j.Url.IsNull() && !j.Url.IsUnknown() && j.Url.ValueString() != "" {
		out.Url = j.Url.ValueStringPointer()
	}
	if !j.KeysJson.IsNull() && !j.KeysJson.IsUnknown() && j.KeysJson.ValueString() != "" {
		var keys []map[string]interface{}
		if err := json.Unmarshal([]byte(j.KeysJson.ValueString()), &keys); err != nil {
			return out, fmt.Errorf("keys_json is not a valid JSON array of objects: %w", err)
		}
		out.Keys = &keys
	}
	return out, nil
}

func (m *FederationIssuerModel) Fill(fi apiclient.FederationIssuer) error {
	m.Id = types.StringValue(fi.Id)
	m.Name = types.StringValue(fi.Name)
	m.IssuerUrl = types.StringValue(fi.IssuerUrl)
	m.CreatedAt = types.StringPointerValue(fi.CreatedAt)
	m.ArchivedAt = types.StringPointerValue(fi.ArchivedAt)

	if fi.Jwks != nil {
		jm := &FederationIssuerJwksModel{
			Type: types.StringValue(fi.Jwks.Type),
			Url:  types.StringPointerValue(fi.Jwks.Url),
		}
		// Preserve the configured keys_json rather than reformatting the
		// API's echo, but fall back to the API's keys on import.
		if m.Jwks != nil && !m.Jwks.KeysJson.IsNull() {
			jm.KeysJson = m.Jwks.KeysJson
		} else if fi.Jwks.Keys != nil {
			b, err := json.Marshal(*fi.Jwks.Keys)
			if err != nil {
				return fmt.Errorf("failed to marshal jwks keys: %w", err)
			}
			jm.KeysJson = types.StringValue(string(b))
		} else {
			jm.KeysJson = types.StringNull()
		}
		m.Jwks = jm
	}
	return nil
}

func (r *FederationIssuerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_federation_issuer"
}

func (r *FederationIssuerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Workload Identity Federation issuer (`fdis_…`) — an OIDC identity provider " +
			"whose JWTs may assert workload identity for your organization.\n\n" +
			"Requires the provider's `oauth_token` (an `org:admin` OAuth bearer token).\n\n" +
			"~> **Experimental (beta).** The Workload Identity Federation endpoints are not exercised by " +
			"the provider's CI acceptance tests, which run with an Admin API key only (no `org:admin` OAuth " +
			"token). Treat this resource as beta and verify its behavior in your own organization.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the federation issuer (`fdis_…`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the issuer. Must match `^[a-z0-9-]+$`, 1–255 chars, unique within the organization.",
				Required:            true,
			},
			"issuer_url": schema.StringAttribute{
				MarkdownDescription: "The exact `iss` claim value in the provider's JWTs, e.g. `https://token.actions.githubusercontent.com`.",
				Required:            true,
			},
			"jwks": schema.SingleNestedAttribute{
				MarkdownDescription: "How Anthropic fetches the provider's JWT signing keys.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "One of `discovery` (fetch `/.well-known/openid-configuration` at the issuer URL), " +
							"`explicit_url` (use `url`), or `inline` (use `keys_json`).",
						Required: true,
					},
					"url": schema.StringAttribute{
						MarkdownDescription: "JWKS endpoint URL. Required when `type = explicit_url`.",
						Optional:            true,
					},
					"keys_json": schema.StringAttribute{
						MarkdownDescription: "A JSON array of JWK objects. Required when `type = inline` (for issuers not reachable from the public internet).",
						Optional:            true,
					},
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the issuer was created.",
				Computed:            true,
			},
			"archived_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the issuer was archived, or null if active.",
				Computed:            true,
			},
		},
	}
}

func (r *FederationIssuerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FederationIssuerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := r.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}

	jwks, err := data.Jwks.toAPI()
	if err != nil {
		resp.Diagnostics.AddError("Config Error", err.Error())
		return
	}

	httpResp, err := r.client.CreateFederationIssuerWithResponse(ctx, apiclient.CreateFederationIssuerJSONRequestBody{
		Name:      data.Name.ValueString(),
		IssuerUrl: data.IssuerUrl.ValueString(),
		Jwks:      jwks,
	}, editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create federation issuer, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create federation issuer, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to create federation issuer, got empty response body")
		return
	}

	if err := data.Fill(*httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FederationIssuerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FederationIssuerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := r.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}

	httpResp, err := r.client.GetFederationIssuerWithResponse(ctx, data.Id.ValueString(), editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read federation issuer, got error: %s", err))
		return
	}
	if httpResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read federation issuer, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to read federation issuer, got empty response body")
		return
	}

	if err := data.Fill(*httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FederationIssuerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FederationIssuerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := r.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}

	jwks, err := data.Jwks.toAPI()
	if err != nil {
		resp.Diagnostics.AddError("Config Error", err.Error())
		return
	}
	name := data.Name.ValueString()
	issuerUrl := data.IssuerUrl.ValueString()

	httpResp, err := r.client.UpdateFederationIssuerWithResponse(ctx, data.Id.ValueString(), apiclient.UpdateFederationIssuerJSONRequestBody{
		Name:      &name,
		IssuerUrl: &issuerUrl,
		Jwks:      &jwks,
	}, editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update federation issuer, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update federation issuer, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
	if httpResp.JSON200 == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to update federation issuer, got empty response body")
		return
	}

	if err := data.Fill(*httpResp.JSON200); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FederationIssuerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FederationIssuerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := r.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}

	httpResp, err := r.client.ArchiveFederationIssuerWithResponse(ctx, data.Id.ValueString(), editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive federation issuer, got error: %s", err))
		return
	}
	if httpResp.StatusCode() != http.StatusOK && httpResp.StatusCode() != http.StatusNotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive federation issuer, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body)))
		return
	}
}

func (r *FederationIssuerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
