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
	displayTitle := acctest.RandomWithPrefix("TF Skill")
	displayTitleUpdated := acctest.RandomWithPrefix("TF Skill Updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSkillResourceConfig(skillName, displayTitle, "A test skill created by Terraform"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("display_title"), knownvalue.StringExact(displayTitle)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("skill_name"), knownvalue.StringExact(skillName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("source"), knownvalue.StringExact("custom")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("latest_version"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("created_at"), knownvalue.NotNull()),
				},
			},
			{
				Config: testAccSkillResourceConfig(skillName, displayTitleUpdated, "An updated test skill"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("display_title"), knownvalue.StringExact(displayTitleUpdated)),
				},
			},
		},
	})
}

func testAccSkillResourceConfig(skillName, displayTitle, description string) string {
	return fmt.Sprintf(`
resource "anthropic_skill" "test" {
	display_title = %[2]q
	skill_name    = %[1]q
	content       = <<-EOT
---
name: %[1]s
description: %[3]s
---

# Test Skill

You are a test skill.
EOT
}
`, skillName, displayTitle, description)
}
