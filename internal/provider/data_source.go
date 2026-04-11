package provider

import (
	"context"
	"fmt"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

type baseDataSource struct {
	client *apiclient.ClientWithResponses
	skills *apiclient.SkillsClient
}

func (d *baseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ProviderClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = clients.API
	d.skills = clients.Skills
}
