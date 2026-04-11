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

func TestAccSkillResource_basic(t *testing.T) {
	rn := "anthropic_skill.test"
	skillName := acctest.RandomWithPrefix("tf-skill")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSkillResourceConfig_basic(skillName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("display_title"), knownvalue.StringExact("TF Test Skill")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("skill_name"), knownvalue.StringExact(skillName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("source"), knownvalue.StringExact("custom")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("latest_version"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("created_at"), knownvalue.NotNull()),
				},
			},
			{
				Config: testAccSkillResourceConfig_updated(skillName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("display_title"), knownvalue.StringExact("TF Test Skill Updated")),
				},
			},
		},
	})
}

func testAccSkillResourceConfig_basic(skillName string) string {
	return fmt.Sprintf(`
resource "anthropic_skill" "test" {
	display_title = "TF Test Skill"
	skill_name    = %[1]q
	content       = <<-EOT
---
name: %[1]s
description: A test skill created by Terraform acceptance tests
---

# Test Skill

You are a test skill. When invoked, respond with "Hello from test skill."
EOT
}
`, skillName)
}

func testAccSkillResourceConfig_updated(skillName string) string {
	return fmt.Sprintf(`
resource "anthropic_skill" "test" {
	display_title = "TF Test Skill Updated"
	skill_name    = %[1]q
	content       = <<-EOT
---
name: %[1]s
description: An updated test skill
---

# Updated Test Skill

You are an updated test skill.
EOT
}
`, skillName)
}
