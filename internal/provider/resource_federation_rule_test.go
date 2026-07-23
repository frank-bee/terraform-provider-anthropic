package provider

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/frank-bee/terraform-provider-anthropic/internal/acctest"
	"github.com/frank-bee/terraform-provider-anthropic/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func init() {
	// Federation rules have no hard delete — the sweeper archives leftovers instead.
	resource.AddTestSweepers("anthropic_federation_rule", &resource.Sweeper{
		Name: "anthropic_federation_rule",
		F: func(r string) error {
			if acctest.TestOAuthToken == "" {
				log.Printf("[INFO] Skipping anthropic_federation_rule sweeper: ANTHROPIC_OAUTH_TOKEN not set")
				return nil
			}

			ctx := context.Background()

			params := &apiclient.ListFederationRulesParams{}

			for {
				httpResp, err := acctest.SharedOAuthClient.ListFederationRulesWithResponse(ctx, params)
				if err != nil {
					return fmt.Errorf("unable to list federation rules: %s", err)
				}

				if httpResp.StatusCode() != http.StatusOK {
					return fmt.Errorf("unable to list federation rules, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body))
				}

				if httpResp.JSON200 == nil {
					break
				}

				for _, fr := range httpResp.JSON200.Data {
					if !strings.HasPrefix(fr.Name, "tf-") {
						continue
					}

					log.Printf("[INFO] Archiving federation rule %s", fr.Id)

					_, err := acctest.SharedOAuthClient.ArchiveFederationRuleWithResponse(ctx, fr.Id)
					if err != nil {
						log.Printf("[ERROR] Unable to archive federation rule %s: %s", fr.Id, err)
						continue
					}

					log.Printf("[INFO] Archived federation rule %s", fr.Id)
				}

				if httpResp.JSON200.NextPage == nil || *httpResp.JSON200.NextPage == "" {
					break
				}
				params.Page = httpResp.JSON200.NextPage
			}

			return nil
		},
	})
}

// TestAccFederationRuleResource_basic covers the happy path: create against a
// fixture issuer (inline JWKS) + service account, import, update name.
func TestAccFederationRuleResource_basic(t *testing.T) {
	rn := "anthropic_federation_rule.test"
	name := acctest.RandomWithPrefix("tf-federation-rule")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckFederation(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFederationRuleResourceConfig_basic(name, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("issuer_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("target").AtMapKey("service_account_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("created_at"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Only the rule's own name changes here — the issuer and
				// service account fixtures keep their original name so
				// issuer_id/target.service_account_id don't get replaced.
				Config: testAccFederationRuleResourceConfig_basic(name, name+"-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(name+"-updated")),
				},
			},
		},
	})
}

func testAccFederationRuleResourceFixtures(fixtureName string) string {
	return fmt.Sprintf(`
resource "anthropic_federation_issuer" "test" {
	name       = %[1]q
	issuer_url = "https://token.actions.githubusercontent.com"
	jwks = {
		type      = "inline"
		keys_json = jsonencode([{ kty = "RSA", kid = "example-key-1", use = "sig", alg = "RS256", n = "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw", e = "AQAB" }])
	}
}

resource "anthropic_service_account" "test" {
	name = %[1]q
}
`, fixtureName)
}

func testAccFederationRuleResourceConfig_basic(fixtureName, ruleName string) string {
	return testAccFederationRuleResourceFixtures(fixtureName) + fmt.Sprintf(`
resource "anthropic_federation_rule" "test" {
	name      = %[1]q
	issuer_id = anthropic_federation_issuer.test.id
	match = {
		subject_prefix = "repo:my-org/my-repo:*"
	}
	target = {
		service_account_id = anthropic_service_account.test.id
	}
	applies_to_all_workspaces = true
}
`, ruleName)
}
