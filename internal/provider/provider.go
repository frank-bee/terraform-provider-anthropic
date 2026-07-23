package provider

import (
	"context"
	"net/http"
	"os"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure AnthropicProvider satisfies various provider interfaces.
var _ provider.Provider = &AnthropicProvider{}
var _ provider.ProviderWithFunctions = &AnthropicProvider{}

// AnthropicProvider defines the provider implementation.
type AnthropicProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// AnthropicProviderModel describes the provider data model.
type AnthropicProviderModel struct {
	BaseUrl    types.String `tfsdk:"base_url"`
	ApiKey     types.String `tfsdk:"api_key"`
	OAuthToken types.String `tfsdk:"oauth_token"`
}

func (p *AnthropicProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "anthropic"
	resp.Version = p.version
}

func (p *AnthropicProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Anthropic provider manages Anthropic resources including workspaces, organization members, and Managed Agents (agents, environments, deployments).",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "API endpoint for the Anthropic service. Defaults to `https://api.anthropic.com`. It can be sourced from the `ANTHROPIC_BASE_URL` environment variable.",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "The Admin API key for authentication. Get this from the [Anthropic console](https://console.anthropic.com/settings/admin-keys). It can be sourced from the `ANTHROPIC_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"oauth_token": schema.StringAttribute{
				MarkdownDescription: "An `org:admin` OAuth bearer token. Required only by the Workload Identity Federation resources " +
					"(`anthropic_service_account`, `anthropic_federation_issuer`, `anthropic_federation_rule`), which the " +
					"Admin API key cannot access. Obtain it with `ant auth login --scope org:admin` then " +
					"`ant auth print-credentials --access-token`. It can be sourced from the `ANTHROPIC_OAUTH_TOKEN` environment variable.",
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

func (p *AnthropicProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data AnthropicProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var baseUrl string
	if !data.BaseUrl.IsNull() {
		baseUrl = data.BaseUrl.ValueString()
	} else if v := os.Getenv("ANTHROPIC_BASE_URL"); v != "" {
		baseUrl = v
	} else {
		baseUrl = "https://api.anthropic.com"
	}

	var apiKey string
	if !data.ApiKey.IsNull() {
		apiKey = data.ApiKey.ValueString()
	} else if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		apiKey = v
	}

	var oauthToken string
	if !data.OAuthToken.IsNull() {
		oauthToken = data.OAuthToken.ValueString()
	} else if v := os.Getenv("ANTHROPIC_OAUTH_TOKEN"); v != "" {
		oauthToken = v
	}

	if baseUrl == "" {
		resp.Diagnostics.AddError("base_url is required", "base_url is required")
		return
	}

	if apiKey == "" && oauthToken == "" {
		resp.Diagnostics.AddError(
			"api_key or oauth_token is required",
			"Set `api_key` (or ANTHROPIC_API_KEY) for standard resources, and/or `oauth_token` "+
				"(or ANTHROPIC_OAUTH_TOKEN) for the Workload Identity Federation resources.",
		)
		return
	}

	retryClient := retryablehttp.NewClient()
	retryClient.ErrorHandler = retryablehttp.PassthroughErrorHandler
	retryClient.Logger = nil
	retryClient.RetryMax = 10

	editors := []apiclient.RequestEditorFn{
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("anthropic-version", "2023-06-01")
			req.Header.Set("anthropic-beta", "agent-api-2026-03-01")
			// Only set x-api-key when an Admin API key is configured. In an
			// oauth_token-only configuration apiKey is empty; sending
			// x-api-key: "" makes every non-WIF request 401 instead of the
			// per-call OAuth editor taking over cleanly.
			if apiKey != "" {
				req.Header.Set("x-api-key", apiKey)
			}
			return nil
		},
	}

	stdClient := retryClient.StandardClient()

	client, err := apiclient.NewClientWithResponses(
		baseUrl,
		apiclient.WithHTTPClient(stdClient),
		apiclient.WithRequestEditorFn(editors[0]),
	)
	if err != nil {
		resp.Diagnostics.AddError("failed to create API client", err.Error())
		return
	}

	skillsClient := apiclient.NewSkillsClient(client, baseUrl, stdClient, editors)

	clients := &ProviderClients{
		API:        client,
		Skills:     skillsClient,
		OAuthToken: oauthToken,
	}

	resp.DataSourceData = clients
	resp.ResourceData = clients
}

func (p *AnthropicProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAgentResource,
		NewDeploymentResource,
		NewEnvironmentResource,
		NewFederationIssuerResource,
		NewFederationRuleResource,
		NewMemoryStoreResource,
		NewOrganizationInviteResource,
		NewServiceAccountResource,
		NewSkillResource,
		NewVaultResource,
		NewWorkspaceMemberResource,
		NewWorkspaceResource,
	}
}

func (p *AnthropicProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAgentDataSource,
		NewAgentsDataSource,
		NewDeploymentDataSource,
		NewEnvironmentDataSource,
		NewEnvironmentsDataSource,
		NewFederationIssuerDataSource,
		NewFederationRuleDataSource,
		NewMemoryStoreDataSource,
		NewOrganizationInvitesDataSource,
		NewServiceAccountDataSource,
		NewSkillDataSource,
		NewSkillsDataSource,
		NewVaultDataSource,
		NewUserDataSource,
		NewUsersDataSource,
		NewWorkspaceDataSource,
		NewWorkspaceMemberDataSource,
		NewWorkspaceMembersDataSource,
		NewWorkspacesDataSource,
	}
}

func (p *AnthropicProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AnthropicProvider{
			version: version,
		}
	}
}
