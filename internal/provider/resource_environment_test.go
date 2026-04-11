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
	resource.AddTestSweepers("anthropic_environment", &resource.Sweeper{
		Name: "anthropic_environment",
		F: func(r string) error {
			ctx := context.Background()

			params := &apiclient.ListEnvironmentsParams{}

			for {
				httpResp, err := acctest.SharedClient.ListEnvironmentsWithResponse(ctx, params)
				if err != nil {
					return fmt.Errorf("unable to list environments: %s", err)
				}

				if httpResp.StatusCode() != http.StatusOK {
					return fmt.Errorf("unable to list environments, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body))
				}

				if httpResp.JSON200 == nil {
					break
				}

				for _, env := range httpResp.JSON200.Data {
					if !strings.HasPrefix(env.Name, "tf-") {
						continue
					}

					log.Printf("[INFO] Destroying environment %s", env.Id)

					_, err := acctest.SharedClient.DeleteEnvironmentWithResponse(ctx, env.Id)
					if err != nil {
						log.Printf("[ERROR] Unable to delete environment %s: %s", env.Id, err)
						continue
					}

					log.Printf("[INFO] Deleted environment %s", env.Id)
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

func TestAccEnvironmentResource_basic(t *testing.T) {
	rn := "anthropic_environment.test"
	envName := acctest.RandomWithPrefix("tf-env")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfig_basic(envName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(envName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("config_type"), knownvalue.StringExact("cloud")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("networking_type"), knownvalue.StringExact("unrestricted")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("created_at"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEnvironmentResourceConfig_basic(envName + "-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(envName+"-updated")),
				},
			},
		},
	})
}

func TestAccEnvironmentResource_restricted(t *testing.T) {
	rn := "anthropic_environment.test"
	envName := acctest.RandomWithPrefix("tf-env")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfig_restricted(envName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("networking_type"), knownvalue.StringExact("restricted")),
				},
			},
		},
	})
}

func TestAccEnvironmentResource_withPackages(t *testing.T) {
	rn := "anthropic_environment.test"
	envName := acctest.RandomWithPrefix("tf-env")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfig_withPackages(envName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(envName)),
				},
			},
		},
	})
}

func testAccEnvironmentResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "anthropic_environment" "test" {
	name            = %[1]q
	networking_type = "unrestricted"
}
`, name)
}

func testAccEnvironmentResourceConfig_restricted(name string) string {
	return fmt.Sprintf(`
resource "anthropic_environment" "test" {
	name            = %[1]q
	networking_type = "restricted"
}
`, name)
}

func testAccEnvironmentResourceConfig_withPackages(name string) string {
	return fmt.Sprintf(`
resource "anthropic_environment" "test" {
	name            = %[1]q
	networking_type = "unrestricted"
	packages = {
		"python" = "3.12"
	}
}
`, name)
}
