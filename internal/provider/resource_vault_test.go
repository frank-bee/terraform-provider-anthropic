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
	// Vaults have no hard delete via this sweeper (mirroring deployments) — it
	// archives leftovers rather than assuming delete_on_destroy was set.
	resource.AddTestSweepers("anthropic_vault", &resource.Sweeper{
		Name: "anthropic_vault",
		F: func(r string) error {
			ctx := context.Background()

			params := &apiclient.ListVaultsParams{}

			for {
				httpResp, err := acctest.SharedClient.ListVaultsWithResponse(ctx, params, withManagedAgentsBeta)
				if err != nil {
					return fmt.Errorf("unable to list vaults: %s", err)
				}

				if httpResp.StatusCode() != http.StatusOK {
					return fmt.Errorf("unable to list vaults, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body))
				}

				if httpResp.JSON200 == nil {
					break
				}

				for _, v := range httpResp.JSON200.Data {
					if !strings.HasPrefix(v.DisplayName, "tf-") {
						continue
					}

					log.Printf("[INFO] Archiving vault %s", v.Id)

					_, err := acctest.SharedClient.ArchiveVaultWithResponse(ctx, v.Id, withManagedAgentsBeta)
					if err != nil {
						log.Printf("[ERROR] Unable to archive vault %s: %s", v.Id, err)
						continue
					}

					log.Printf("[INFO] Archived vault %s", v.Id)
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

// TestAccVaultResource_basic covers the happy path: create, import, update
// display_name.
func TestAccVaultResource_basic(t *testing.T) {
	rn := "anthropic_vault.test"
	name := acctest.RandomWithPrefix("tf-vault")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVaultResourceConfig_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("display_name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("created_at"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("metadata").AtMapKey("env"), knownvalue.StringExact("test")),
				},
			},
			{
				ResourceName:            rn,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_on_destroy"},
			},
			{
				Config: testAccVaultResourceConfig_basic(name + "-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("display_name"), knownvalue.StringExact(name+"-updated")),
				},
			},
		},
	})
}

func testAccVaultResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "anthropic_vault" "test" {
	display_name = %[1]q
	metadata = {
		env = "test"
	}
}
`, name)
}
