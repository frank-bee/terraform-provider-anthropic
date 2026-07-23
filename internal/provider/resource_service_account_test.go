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
	// Service accounts have no hard delete — the sweeper archives leftovers instead.
	resource.AddTestSweepers("anthropic_service_account", &resource.Sweeper{
		Name: "anthropic_service_account",
		F: func(r string) error {
			if acctest.TestOAuthToken == "" {
				log.Printf("[INFO] Skipping anthropic_service_account sweeper: ANTHROPIC_OAUTH_TOKEN not set")
				return nil
			}

			ctx := context.Background()

			params := &apiclient.ListServiceAccountsParams{}

			for {
				httpResp, err := acctest.SharedOAuthClient.ListServiceAccountsWithResponse(ctx, params)
				if err != nil {
					return fmt.Errorf("unable to list service accounts: %s", err)
				}

				if httpResp.StatusCode() != http.StatusOK {
					return fmt.Errorf("unable to list service accounts, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body))
				}

				if httpResp.JSON200 == nil {
					break
				}

				for _, sa := range httpResp.JSON200.Data {
					if !strings.HasPrefix(sa.Name, "tf-") {
						continue
					}

					log.Printf("[INFO] Archiving service account %s", sa.Id)

					_, err := acctest.SharedOAuthClient.ArchiveServiceAccountWithResponse(ctx, sa.Id)
					if err != nil {
						log.Printf("[ERROR] Unable to archive service account %s: %s", sa.Id, err)
						continue
					}

					log.Printf("[INFO] Archived service account %s", sa.Id)
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

// TestAccServiceAccountResource_basic covers the happy path: create, import, update name.
func TestAccServiceAccountResource_basic(t *testing.T) {
	rn := "anthropic_service_account.test"
	saName := acctest.RandomWithPrefix("tf-service-account")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckFederation(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountResourceConfig_basic(saName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(saName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("created_at"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccServiceAccountResourceConfig_basic(saName + "-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(saName+"-updated")),
				},
			},
		},
	})
}

func testAccServiceAccountResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "anthropic_service_account" "test" {
	name = %[1]q
}
`, name)
}
