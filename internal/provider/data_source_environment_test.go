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

func TestAccEnvironmentDataSource(t *testing.T) {
	rn := "data.anthropic_environment.test"
	envName := acctest.RandomWithPrefix("tf-env")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentDataSourceConfig(envName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(envName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("config_type"), knownvalue.StringExact("cloud")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("networking_type"), knownvalue.StringExact("unrestricted")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("created_at"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func testAccEnvironmentDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "anthropic_environment" "test" {
	name            = %[1]q
	networking_type = "unrestricted"
}

data "anthropic_environment" "test" {
	id = anthropic_environment.test.id
}
`, name)
}
