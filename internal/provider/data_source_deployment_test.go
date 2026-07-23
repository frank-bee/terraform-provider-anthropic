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

func TestAccDeploymentDataSource(t *testing.T) {
	rn := "data.anthropic_deployment.test"
	depName := acctest.RandomWithPrefix("tf-deployment")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentDataSourceConfig(depName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(depName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("agent_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("agent_version"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("environment_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("status"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("created_at"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func testAccDeploymentDataSourceFixtures(name string) string {
	return fmt.Sprintf(`
resource "anthropic_agent" "test" {
	name  = %[1]q
	model = "claude-sonnet-4-5"
}

resource "anthropic_environment" "test" {
	name            = %[1]q
	networking_type = "unrestricted"
}
`, name)
}

func testAccDeploymentDataSourceConfig(name string) string {
	return testAccDeploymentDataSourceFixtures(name) + fmt.Sprintf(`
resource "anthropic_deployment" "test" {
	name           = %[1]q
	agent_id       = anthropic_agent.test.id
	environment_id = anthropic_environment.test.id

	initial_events = [jsonencode({
		type    = "user.message"
		content = [{ type = "text", text = "hello" }]
	})]
}

data "anthropic_deployment" "test" {
	id = anthropic_deployment.test.id
}
`, name)
}
