package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---- service_account ----

func NewServiceAccountDataSource() datasource.DataSource { return &ServiceAccountDataSource{} }

var _ datasource.DataSource = &ServiceAccountDataSource{}

type ServiceAccountDataSource struct{ baseDataSource }

func (d *ServiceAccountDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (d *ServiceAccountDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get a single service account (`svac_…`) by ID. Requires the provider's `oauth_token`.\n\n" +
			"~> **Experimental (beta).** The Workload Identity Federation endpoints are not exercised by the provider's CI acceptance tests. Treat as beta.",
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Required: true, MarkdownDescription: "ID of the service account (`svac_…`)."},
			"name":              schema.StringAttribute{Computed: true, MarkdownDescription: "Name of the service account."},
			"organization_role": schema.StringAttribute{Computed: true, MarkdownDescription: "Organization role of the service account."},
			"created_at":        schema.StringAttribute{Computed: true, MarkdownDescription: "RFC 3339 creation datetime."},
			"archived_at":       schema.StringAttribute{Computed: true, MarkdownDescription: "RFC 3339 archive datetime, or null if active."},
		},
	}
}

func (d *ServiceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServiceAccountModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := d.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}
	httpResp, err := d.client.GetServiceAccountWithResponse(ctx, data.Id.ValueString(), editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service account, got error: %s", err))
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

// ---- federation_issuer ----

func NewFederationIssuerDataSource() datasource.DataSource { return &FederationIssuerDataSource{} }

var _ datasource.DataSource = &FederationIssuerDataSource{}

type FederationIssuerDataSource struct{ baseDataSource }

func (d *FederationIssuerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_federation_issuer"
}

func (d *FederationIssuerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get a single federation issuer (`fdis_…`) by ID. Requires the provider's `oauth_token`.\n\n" +
			"~> **Experimental (beta).** The Workload Identity Federation endpoints are not exercised by the provider's CI acceptance tests. Treat as beta.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Required: true, MarkdownDescription: "ID of the federation issuer (`fdis_…`)."},
			"name":       schema.StringAttribute{Computed: true, MarkdownDescription: "Name of the issuer."},
			"issuer_url": schema.StringAttribute{Computed: true, MarkdownDescription: "The `iss` claim value the issuer's JWTs carry."},
			"jwks": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "How Anthropic fetches the provider's signing keys.",
				Attributes: map[string]schema.Attribute{
					"type":      schema.StringAttribute{Computed: true, MarkdownDescription: "`discovery`, `explicit_url`, or `inline`."},
					"url":       schema.StringAttribute{Computed: true, MarkdownDescription: "JWKS endpoint URL (explicit_url mode)."},
					"keys_json": schema.StringAttribute{Computed: true, MarkdownDescription: "JSON array of JWK objects (inline mode)."},
				},
			},
			"created_at":  schema.StringAttribute{Computed: true, MarkdownDescription: "RFC 3339 creation datetime."},
			"archived_at": schema.StringAttribute{Computed: true, MarkdownDescription: "RFC 3339 archive datetime, or null if active."},
		},
	}
}

func (d *FederationIssuerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FederationIssuerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := d.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}
	httpResp, err := d.client.GetFederationIssuerWithResponse(ctx, data.Id.ValueString(), editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read federation issuer, got error: %s", err))
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

// ---- federation_rule ----

func NewFederationRuleDataSource() datasource.DataSource { return &FederationRuleDataSource{} }

var _ datasource.DataSource = &FederationRuleDataSource{}

type FederationRuleDataSource struct{ baseDataSource }

func (d *FederationRuleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_federation_rule"
}

func (d *FederationRuleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get a single federation rule (`fdrl_…`) by ID. Requires the provider's `oauth_token`.\n\n" +
			"~> **Experimental (beta).** The Workload Identity Federation endpoints are not exercised by the provider's CI acceptance tests. Treat as beta.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Required: true, MarkdownDescription: "ID of the federation rule (`fdrl_…`)."},
			"name":      schema.StringAttribute{Computed: true, MarkdownDescription: "Name of the rule."},
			"issuer_id": schema.StringAttribute{Computed: true, MarkdownDescription: "ID of the issuer the rule applies to."},
			"match": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Conditions an incoming JWT must satisfy.",
				Attributes: map[string]schema.Attribute{
					"subject_prefix": schema.StringAttribute{Computed: true, MarkdownDescription: "Match on the `sub` claim."},
					"audience":       schema.StringAttribute{Computed: true, MarkdownDescription: "Required `aud` claim."},
					"claims":         schema.MapAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Exact claim values required."},
					"condition":      schema.StringAttribute{Computed: true, MarkdownDescription: "CEL match expression."},
				},
			},
			"target": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The service account a matched JWT maps to.",
				Attributes: map[string]schema.Attribute{
					"type":               schema.StringAttribute{Computed: true, MarkdownDescription: "Target type."},
					"service_account_id": schema.StringAttribute{Computed: true, MarkdownDescription: "Target service account ID (`svac_…`)."},
				},
			},
			"workspace_id":              schema.StringAttribute{Computed: true, MarkdownDescription: "Not returned by the API; always null on reads."},
			"applies_to_all_workspaces": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the rule applies to all workspaces."},
			"oauth_scope":               schema.StringAttribute{Computed: true, MarkdownDescription: "OAuth scope granted on minted tokens."},
			"token_lifetime_seconds":    schema.Int64Attribute{Computed: true, MarkdownDescription: "Minted token lifetime in seconds."},
			"created_at":                schema.StringAttribute{Computed: true, MarkdownDescription: "RFC 3339 creation datetime."},
			"archived_at":               schema.StringAttribute{Computed: true, MarkdownDescription: "RFC 3339 archive datetime, or null if active."},
		},
	}
}

func (d *FederationRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FederationRuleModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	editor, ok := d.oauthEditor(&resp.Diagnostics)
	if !ok {
		return
	}
	httpResp, err := d.client.GetFederationRuleWithResponse(ctx, data.Id.ValueString(), editor)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read federation rule, got error: %s", err))
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
