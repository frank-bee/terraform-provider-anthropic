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

func NewMemoryStoreDataSource() datasource.DataSource {
	return &MemoryStoreDataSource{}
}

var _ datasource.DataSource = &MemoryStoreDataSource{}

type MemoryStoreDataSource struct {
	baseDataSource
}

// MemoryStoreDataSourceModel mirrors MemoryStoreModel minus the provider-only
// delete_on_destroy field, which the data source's schema doesn't expose.
type MemoryStoreDataSourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Metadata    types.Map    `tfsdk:"metadata"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	ArchivedAt  types.String `tfsdk:"archived_at"`
}

func (m *MemoryStoreDataSourceModel) Fill(ms apiclient.MemoryStore) {
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

func (d *MemoryStoreDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_memory_store"
}

func (d *MemoryStoreDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about a single Memory Store by ID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the Memory Store (`memstore_…`).",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the Memory Store.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the Memory Store.",
				Computed:            true,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Arbitrary key-value metadata attached to the Memory Store.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the Memory Store was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the Memory Store was last updated.",
				Computed:            true,
			},
			"archived_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 datetime when the Memory Store was archived, or null if active.",
				Computed:            true,
			},
		},
	}
}

func (d *MemoryStoreDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MemoryStoreDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.client.GetMemoryStoreWithResponse(ctx, data.Id.ValueString(), withManagedAgentsBeta)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read memory store, got error: %s", err))
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
