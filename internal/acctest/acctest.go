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
	TestApiKey = os.Getenv("ANTHROPIC_API_KEY")
	TestUserId = os.Getenv("ANTHROPIC_TEST_USER_ID")

	SharedClient       *apiclient.ClientWithResponses
	SharedSkillsClient *apiclient.SkillsClient
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
