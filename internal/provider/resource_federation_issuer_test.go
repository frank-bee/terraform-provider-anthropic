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
	// Federation issuers have no hard delete — the sweeper archives leftovers instead.
	resource.AddTestSweepers("anthropic_federation_issuer", &resource.Sweeper{
		Name: "anthropic_federation_issuer",
		F: func(r string) error {
			if acctest.TestOAuthToken == "" {
				log.Printf("[INFO] Skipping anthropic_federation_issuer sweeper: ANTHROPIC_OAUTH_TOKEN not set")
				return nil
			}

			ctx := context.Background()

			params := &apiclient.ListFederationIssuersParams{}

			for {
				httpResp, err := acctest.SharedOAuthClient.ListFederationIssuersWithResponse(ctx, params)
				if err != nil {
					return fmt.Errorf("unable to list federation issuers: %s", err)
				}

				if httpResp.StatusCode() != http.StatusOK {
					return fmt.Errorf("unable to list federation issuers, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body))
				}

				if httpResp.JSON200 == nil {
					break
				}

				for _, fi := range httpResp.JSON200.Data {
					if !strings.HasPrefix(fi.Name, "tf-") {
						continue
					}

					log.Printf("[INFO] Archiving federation issuer %s", fi.Id)

					_, err := acctest.SharedOAuthClient.ArchiveFederationIssuerWithResponse(ctx, fi.Id)
					if err != nil {
						log.Printf("[ERROR] Unable to archive federation issuer %s: %s", fi.Id, err)
						continue
					}

					log.Printf("[INFO] Archived federation issuer %s", fi.Id)
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

// TestAccFederationIssuerResource_basic covers the happy path: create with a
// static inline JWKS, import, update name.
func TestAccFederationIssuerResource_basic(t *testing.T) {
	rn := "anthropic_federation_issuer.test"
	fiName := acctest.RandomWithPrefix("tf-federation-issuer")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckFederation(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFederationIssuerResourceConfig_basic(fiName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(fiName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("issuer_url"), knownvalue.StringExact("https://token.actions.githubusercontent.com")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("created_at"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccFederationIssuerResourceConfig_basic(fiName + "-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(fiName+"-updated")),
				},
			},
		},
	})
}

func testAccFederationIssuerResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "anthropic_federation_issuer" "test" {
	name       = %[1]q
	issuer_url = "https://token.actions.githubusercontent.com"
	jwks = {
		type      = "inline"
		keys_json = jsonencode([{
			kty = "RSA"
			kid = "example-key-1"
			use = "sig"
			alg = "RS256"
			n   = "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw"
			e   = "AQAB"
		}])
	}
}
`, name)
}
