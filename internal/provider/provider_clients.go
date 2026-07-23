package provider

import "github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"

// ProviderClients holds all API clients needed by resources and data sources.
type ProviderClients struct {
	API    *apiclient.ClientWithResponses
	Skills *apiclient.SkillsClient
	// OAuthToken is an org:admin OAuth bearer token. Required only by the
	// Workload Identity Federation resources (service accounts, federation
	// issuers, federation rules), which the Admin API key cannot access.
	// Empty when the provider was configured without one.
	OAuthToken string
}
