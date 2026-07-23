package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewVaultDataSource() datasource.DataSource {
	return &VaultDataSource{}
}

var _ datasource.DataSource = &VaultDataSource{}

type VaultDataSource struct {
	baseDataSource
}

// VaultDataSourceModel intentionally has no `delete_on_destroy` field — that
// attribute is provider-only and specific to the resource; the data source
// schema doesn't declare it, so the model must not either (terraform-plugin-
// framework requires the model's tfsdk fields to exactly match the schema).
type VaultDataSourceModel struct {
	Id          types.String `tfsdk:"id"`
	DisplayName types.String `tfsdk:"display_name"`
	Metadata    types.Map    `tfsdk:"metadata"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	ArchivedAt  types.String `tfsdk:"archived_at"`
}

func (m *VaultDataSourceModel) Fill(v apiclient.Vault) {
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

func (d *VaultDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault"
}

func (d *VaultDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about a single Vault by ID.\n\n" +
			"~> **Experimental (beta).** This resource's wire format was derived from live-API probing rather than published documentation and may change. Treat as beta.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the Vault (`vlt_…`).",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name of the Vault.",
				Computed:            true,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Arbitrary key-value metadata attached to the Vault.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the Vault was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the Vault was last updated.",
				Computed:            true,
			},
			"archived_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the Vault was archived, or null if active.",
				Computed:            true,
			},
		},
	}
}

func (d *VaultDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VaultDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.client.GetVaultWithResponse(ctx, data.Id.ValueString(), withManagedAgentsBeta)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read vault, got error: %s", err))
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
