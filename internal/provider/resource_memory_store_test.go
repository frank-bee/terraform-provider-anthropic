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
	// Memory stores default to soft-delete (archive) on destroy, so leftovers
	// from a failed test run are archived rather than left active.
	resource.AddTestSweepers("anthropic_memory_store", &resource.Sweeper{
		Name: "anthropic_memory_store",
		F: func(r string) error {
			ctx := context.Background()

			params := &apiclient.ListMemoryStoresParams{}

			for {
				httpResp, err := acctest.SharedClient.ListMemoryStoresWithResponse(ctx, params, withManagedAgentsBeta)
				if err != nil {
					return fmt.Errorf("unable to list memory stores: %s", err)
				}

				if httpResp.StatusCode() != http.StatusOK {
					return fmt.Errorf("unable to list memory stores, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body))
				}

				if httpResp.JSON200 == nil {
					break
				}

				for _, ms := range httpResp.JSON200.Data {
					if !strings.HasPrefix(ms.Name, "tf-") {
						continue
					}

					log.Printf("[INFO] Archiving memory store %s", ms.Id)

					_, err := acctest.SharedClient.ArchiveMemoryStoreWithResponse(ctx, ms.Id, withManagedAgentsBeta)
					if err != nil {
						log.Printf("[ERROR] Unable to archive memory store %s: %s", ms.Id, err)
						continue
					}

					log.Printf("[INFO] Archived memory store %s", ms.Id)
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

// TestAccMemoryStoreResource_basic covers the happy path: create, import,
// update description.
func TestAccMemoryStoreResource_basic(t *testing.T) {
	rn := "anthropic_memory_store.test"
	msName := acctest.RandomWithPrefix("tf-memory")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMemoryStoreResourceConfig_basic(msName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(msName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("description"), knownvalue.StringExact("acc test")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("created_at"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:            rn,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_on_destroy"},
			},
			{
				Config: testAccMemoryStoreResourceConfig_basic(msName, "acc test updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("description"), knownvalue.StringExact("acc test updated")),
				},
			},
		},
	})
}

func testAccMemoryStoreResourceConfig_basic(name string, description ...string) string {
	desc := "acc test"
	if len(description) > 0 {
		desc = description[0]
	}
	return fmt.Sprintf(`
resource "anthropic_memory_store" "test" {
	name        = %[1]q
	description = %[2]q

	metadata = {
		env = "test"
	}
}
`, name, desc)
}
