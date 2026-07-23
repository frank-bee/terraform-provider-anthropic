package provider

import (
	"fmt"
	"testing"

	"github.com/frank-bee/terraform-provider-anthropic/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccAgentDataSource(t *testing.T) {
	rn := "data.anthropic_agent.test"
	agentName := acctest.RandomWithPrefix("tf-agent")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentDataSourceConfig(agentName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(agentName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("model"), knownvalue.StringExact("claude-sonnet-4-5")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("version"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func testAccAgentDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "anthropic_agent" "test" {
	name  = %[1]q
	model = "claude-sonnet-4-5"
}

data "anthropic_agent" "test" {
	id = anthropic_agent.test.id
}
`, name)
}
