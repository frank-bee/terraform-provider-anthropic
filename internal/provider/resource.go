package provider

import (
	"context"
	"fmt"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type baseResource struct {
	client     *apiclient.ClientWithResponses
	skills     *apiclient.SkillsClient
	oauthToken string
}

func (r *baseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = clients.API
	r.skills = clients.Skills
	r.oauthToken = clients.OAuthToken
}

// oauthEditor returns a request editor that authenticates with the provider's
// org:admin OAuth bearer token, for the Workload Identity Federation endpoints
// that reject the Admin API key. It adds an error to diags and returns ok=false
// when no oauth_token was configured.
func (r *baseResource) oauthEditor(diags *diag.Diagnostics) (apiclient.RequestEditorFn, bool) {
	if r.oauthToken == "" {
		diags.AddError(
			"oauth_token is required",
			"This resource uses Anthropic's Workload Identity Federation endpoints, which require an "+
				"`org:admin` OAuth bearer token. Set `oauth_token` on the provider (or the "+
				"ANTHROPIC_OAUTH_TOKEN environment variable).",
		)
		return nil, false
	}
	return withOAuthBearer(r.oauthToken), true
}
