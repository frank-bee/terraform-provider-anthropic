# Create a custom skill with inline SKILL.md content
resource "anthropic_skill" "code_reviewer" {
  display_title = "Code Reviewer"
  skill_name    = "code-reviewer"
  content       = <<-EOT
---
name: code-reviewer
description: Reviews code for quality issues, security vulnerabilities, and best practices
---

# Code Review Skill

## Instructions
When asked to review code:
1. Check for security vulnerabilities (SQL injection, XSS, etc.)
2. Verify error handling is comprehensive
3. Look for performance issues
4. Ensure code follows project conventions
5. Suggest improvements with examples

## Output Format
Provide findings as a numbered list with severity (HIGH/MEDIUM/LOW).
EOT
}

# Use the skill with an agent
resource "anthropic_agent" "reviewer" {
  name  = "code-review-agent"
  model = "claude-sonnet-4-5"

  skills {
    skill_id = anthropic_skill.code_reviewer.id
    type     = "custom"
    version  = "latest"
  }
}
