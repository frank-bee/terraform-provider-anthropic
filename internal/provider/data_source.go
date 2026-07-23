package provider

import (
	"context"
	"fmt"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type baseDataSource struct {
	client     *apiclient.ClientWithResponses
	skills     *apiclient.SkillsClient
	oauthToken string
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
	d.oauthToken = clients.OAuthToken
}

// oauthEditor mirrors baseResource.oauthEditor for data sources reading the
// Workload Identity Federation endpoints.
func (d *baseDataSource) oauthEditor(diags *diag.Diagnostics) (apiclient.RequestEditorFn, bool) {
	if d.oauthToken == "" {
		diags.AddError(
			"oauth_token is required",
			"This data source uses Anthropic's Workload Identity Federation endpoints, which require an "+
				"`org:admin` OAuth bearer token. Set `oauth_token` on the provider (or the "+
				"ANTHROPIC_OAUTH_TOKEN environment variable).",
		)
		return nil, false
	}
	return withOAuthBearer(d.oauthToken), true
}
