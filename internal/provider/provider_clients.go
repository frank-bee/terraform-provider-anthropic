package provider

import "github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"

// ProviderClients holds all API clients needed by resources and data sources.
type ProviderClients struct {
	API    *apiclient.ClientWithResponses
	Skills *apiclient.SkillsClient
}
