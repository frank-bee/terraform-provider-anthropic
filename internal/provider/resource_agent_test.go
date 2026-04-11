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
	resource.AddTestSweepers("anthropic_agent", &resource.Sweeper{
		Name: "anthropic_agent",
		F: func(r string) error {
			ctx := context.Background()

			params := &apiclient.ListAgentsParams{}

			for {
				httpResp, err := acctest.SharedClient.ListAgentsWithResponse(ctx, params)
				if err != nil {
					return fmt.Errorf("unable to list agents: %s", err)
				}

				if httpResp.StatusCode() != http.StatusOK {
					return fmt.Errorf("unable to list agents, got status code %d: %s", httpResp.StatusCode(), string(httpResp.Body))
				}

				if httpResp.JSON200 == nil {
					break
				}

				for _, agent := range httpResp.JSON200.Data {
					if !strings.HasPrefix(agent.Name, "tf-") {
						continue
					}

					log.Printf("[INFO] Destroying agent %s", agent.Id)

					_, err := acctest.SharedClient.DeleteAgentWithResponse(ctx, agent.Id)
					if err != nil {
						log.Printf("[ERROR] Unable to delete agent %s: %s", agent.Id, err)
						continue
					}

					log.Printf("[INFO] Deleted agent %s", agent.Id)
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

func TestAccAgentResource_basic(t *testing.T) {
	rn := "anthropic_agent.test"
	agentName := acctest.RandomWithPrefix("tf-agent")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_basic(agentName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(agentName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("model"), knownvalue.StringExact("claude-sonnet-4-5")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("version"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAgentResourceConfig_basic(agentName + "-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(agentName+"-updated")),
				},
			},
		},
	})
}

func TestAccAgentResource_withTools(t *testing.T) {
	rn := "anthropic_agent.test"
	agentName := acctest.RandomWithPrefix("tf-agent")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_withTools(agentName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(agentName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("system"), knownvalue.StringExact("You are a helpful assistant.")),
				},
			},
		},
	})
}

func TestAccAgentResource_withSystem(t *testing.T) {
	rn := "anthropic_agent.test"
	agentName := acctest.RandomWithPrefix("tf-agent")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentResourceConfig_withSystem(agentName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("system"), knownvalue.StringExact("You are a DevOps assistant.")),
				},
			},
		},
	})
}

func testAccAgentResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "anthropic_agent" "test" {
	name  = %[1]q
	model = "claude-sonnet-4-5"
}
`, name)
}

func testAccAgentResourceConfig_withTools(name string) string {
	return fmt.Sprintf(`
resource "anthropic_agent" "test" {
	name   = %[1]q
	model  = "claude-sonnet-4-5"
	system = "You are a helpful assistant."

	tools {
		type = "agent_toolset_20251212"
	}
}
`, name)
}

func testAccAgentResourceConfig_withSystem(name string) string {
	return fmt.Sprintf(`
resource "anthropic_agent" "test" {
	name   = %[1]q
	model  = "claude-sonnet-4-5"
	system = "You are a DevOps assistant."
}
`, name)
}
