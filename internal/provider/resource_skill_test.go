package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/frank-bee/terraform-provider-anthropic/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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

// Verifies that updating `content` issues a new skill version in place
// (skill ID is preserved, no destroy/recreate).
func TestAccSkillResource_contentUpdateInPlace(t *testing.T) {
	rn := "anthropic_skill.test"
	skillName := acctest.RandomWithPrefix("tf-skill")
	displayTitle := acctest.RandomWithPrefix("TF Skill")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSkillResourceConfig(skillName, displayTitle, "Original description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("latest_version"), knownvalue.NotNull()),
				},
			},
			{
				Config: testAccSkillResourceConfig(skillName, displayTitle, "Updated description"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(rn, plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

// Verifies that source_dir packages multiple files into the skill and that
// editing any file triggers an in-place update.
func TestAccSkillResource_sourceDir(t *testing.T) {
	rn := "anthropic_skill.test"
	skillName := acctest.RandomWithPrefix("tf-skill")
	displayTitle := acctest.RandomWithPrefix("TF Skill SourceDir")

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "SKILL.md"), fmt.Sprintf(
		"---\nname: %s\ndescription: source_dir test\n---\n\n# Hi\n", skillName))
	mustWrite(t, filepath.Join(dir, "scripts", "helper.sh"), "#!/bin/sh\necho v1\n")
	mustWrite(t, filepath.Join(dir, "references", "api.md"), "# API\nv1\n")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckManagedAgents(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSkillResourceConfigSourceDir(skillName, displayTitle, dir),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("source_hash"), knownvalue.NotNull()),
				},
			},
			{
				PreConfig: func() {
					mustWrite(t, filepath.Join(dir, "scripts", "helper.sh"), "#!/bin/sh\necho v2\n")
				},
				Config: testAccSkillResourceConfigSourceDir(skillName, displayTitle, dir),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(rn, plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

func testAccSkillResourceConfigSourceDir(skillName, displayTitle, dir string) string {
	return fmt.Sprintf(`
resource "anthropic_skill" "test" {
	display_title = %[2]q
	skill_name    = %[1]q
	source_dir    = %[3]q
}
`, skillName, displayTitle, dir)
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
