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
				httpResp, err := acctest.SharedClient.ListEnvironmentsWithResponse(ctx, params, withEnvironmentsBeta)
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

					_, err := acctest.SharedClient.DeleteEnvironmentWithResponse(ctx, env.Id, withEnvironmentsBeta)
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

// TestAccEnvironmentResource_basic covers the happy path: create, import,
// update name. Exercises only the minimal required attributes.
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

// TestAccEnvironmentResource_descriptionAndMetadata covers the top-level
// description + metadata attributes.
func TestAccEnvironmentResource_descriptionAndMetadata(t *testing.T) {
	rn := "anthropic_environment.test"
	envName := acctest.RandomWithPrefix("tf-env")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfig_descAndMeta(envName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("description"),
						knownvalue.StringExact("created-by-acceptance-test")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("metadata").AtMapKey("owner"),
						knownvalue.StringExact("tf-test")),
				},
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccEnvironmentResource_limited covers networking.type = "limited"
// with allow_package_managers + allowed_hosts.
func TestAccEnvironmentResource_limited(t *testing.T) {
	rn := "anthropic_environment.test"
	envName := acctest.RandomWithPrefix("tf-env")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfig_limited(envName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("networking_type"),
						knownvalue.StringExact("limited")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("allow_package_managers"),
						knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("allowed_hosts"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("api.github.com"),
							knownvalue.StringExact("github.com"),
						})),
				},
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccEnvironmentResource_withAptPackages covers the packages per-manager
// lists. Only apt is exercised against the live API to keep the test cheap —
// the Fill() code path for all managers is covered by unit tests.
func TestAccEnvironmentResource_withAptPackages(t *testing.T) {
	rn := "anthropic_environment.test"
	envName := acctest.RandomWithPrefix("tf-env")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfig_withAptPackages(envName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("apt_packages"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("gh"),
							knownvalue.StringExact("jq"),
						})),
				},
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
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

func testAccEnvironmentResourceConfig_descAndMeta(name string) string {
	return fmt.Sprintf(`
resource "anthropic_environment" "test" {
	name            = %[1]q
	networking_type = "unrestricted"
	description     = "created-by-acceptance-test"
	metadata = {
		owner = "tf-test"
	}
}
`, name)
}

func testAccEnvironmentResourceConfig_limited(name string) string {
	return fmt.Sprintf(`
resource "anthropic_environment" "test" {
	name                    = %[1]q
	networking_type         = "limited"
	allow_package_managers  = true
	allowed_hosts           = ["api.github.com", "github.com"]
	apt_packages            = ["jq"]
}
`, name)
}

func testAccEnvironmentResourceConfig_withAptPackages(name string) string {
	return fmt.Sprintf(`
resource "anthropic_environment" "test" {
	name            = %[1]q
	networking_type = "unrestricted"
	apt_packages    = ["gh", "jq"]
}
`, name)
}
