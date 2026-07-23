package acctest

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/jianyuan/go-utils/must"
)

var (
	TestApiKey     = os.Getenv("ANTHROPIC_API_KEY")
	TestUserId     = os.Getenv("ANTHROPIC_TEST_USER_ID")
	TestOAuthToken = os.Getenv("ANTHROPIC_OAUTH_TOKEN")

	SharedClient       *apiclient.ClientWithResponses
	SharedSkillsClient *apiclient.SkillsClient
	// SharedOAuthClient authenticates with the org:admin OAuth bearer token
	// instead of the Admin API key. The Workload Identity Federation endpoints
	// (service accounts, federation issuers, federation rules) reject x-api-key
	// and accept only a Bearer token, so their sweepers use this client.
	SharedOAuthClient *apiclient.ClientWithResponses
)

func init() {
	editor := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("anthropic-beta", "agent-api-2026-03-01")
		req.Header.Set("x-api-key", TestApiKey)
		return nil
	}

	SharedClient = must.Get(apiclient.NewClientWithResponses(
		"https://api.anthropic.com",
		apiclient.WithRequestEditorFn(editor),
	))

	SharedSkillsClient = apiclient.NewSkillsClient(
		SharedClient,
		"https://api.anthropic.com",
		http.DefaultClient,
		[]apiclient.RequestEditorFn{editor},
	)

	oauthEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Del("x-api-key")
		req.Header.Set("authorization", "Bearer "+TestOAuthToken)
		return nil
	}

	SharedOAuthClient = must.Get(apiclient.NewClientWithResponses(
		"https://api.anthropic.com",
		apiclient.WithRequestEditorFn(oauthEditor),
	))
}

func PreCheck(t *testing.T) {
	if TestApiKey == "" {
		t.Fatal("ANTHROPIC_API_KEY must be set for acceptance tests")
	}

	if TestUserId == "" {
		t.Fatal("ANTHROPIC_TEST_USER_ID must be set for acceptance tests")
	}
}

func PreCheckManagedAgents(t *testing.T) {
	if TestApiKey == "" {
		t.Fatal("ANTHROPIC_API_KEY must be set for acceptance tests")
	}
}

// PreCheckFederation gates the Workload Identity Federation acceptance tests
// (service accounts, federation issuers, federation rules). These need an
// org:admin OAuth bearer token, which the Admin API key cannot substitute for.
// When the token is absent the test is skipped rather than failed, so the
// normal acceptance run doesn't require org:admin credentials.
func PreCheckFederation(t *testing.T) {
	if TestOAuthToken == "" {
		t.Skip("ANTHROPIC_OAUTH_TOKEN not set; skipping Workload Identity Federation acceptance test")
	}
}
