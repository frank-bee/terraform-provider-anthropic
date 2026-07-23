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

// ---- anthropic_service_account ----

func TestAccServiceAccountDataSource(t *testing.T) {
	rn := "data.anthropic_service_account.test"
	saName := acctest.RandomWithPrefix("tf-service-account")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckFederation(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountDataSourceConfig(saName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(saName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("organization_role"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("created_at"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("archived_at"), knownvalue.Null()),
				},
			},
		},
	})
}

func testAccServiceAccountDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "anthropic_service_account" "test" {
	name = %[1]q
}

data "anthropic_service_account" "test" {
	id = anthropic_service_account.test.id
}
`, name)
}

// ---- anthropic_federation_issuer ----

func TestAccFederationIssuerDataSource(t *testing.T) {
	rn := "data.anthropic_federation_issuer.test"
	fiName := acctest.RandomWithPrefix("tf-federation-issuer")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckFederation(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFederationIssuerDataSourceConfig(fiName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(fiName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("issuer_url"), knownvalue.StringExact("https://token.actions.githubusercontent.com")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("jwks").AtMapKey("type"), knownvalue.StringExact("inline")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("created_at"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("archived_at"), knownvalue.Null()),
				},
			},
		},
	})
}

func testAccFederationIssuerDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "anthropic_federation_issuer" "test" {
	name       = %[1]q
	issuer_url = "https://token.actions.githubusercontent.com"
	jwks = {
		type      = "inline"
		keys_json = jsonencode([{
			kty = "RSA"
			kid = "example-key-1"
			use = "sig"
			alg = "RS256"
			n   = "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw"
			e   = "AQAB"
		}])
	}
}

data "anthropic_federation_issuer" "test" {
	id = anthropic_federation_issuer.test.id
}
`, name)
}

// ---- anthropic_federation_rule ----

func TestAccFederationRuleDataSource(t *testing.T) {
	rn := "data.anthropic_federation_rule.test"
	fiName := acctest.RandomWithPrefix("tf-federation-issuer")
	saName := acctest.RandomWithPrefix("tf-service-account")
	frName := acctest.RandomWithPrefix("tf-federation-rule")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckFederation(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFederationRuleDataSourceConfig(fiName, saName, frName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("name"), knownvalue.StringExact(frName)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("issuer_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("match").AtMapKey("subject_prefix"), knownvalue.StringExact("repo:example-org/example-repo:*")),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("target").AtMapKey("service_account_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("applies_to_all_workspaces"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("oauth_scope"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("token_lifetime_seconds"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("created_at"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(rn, tfjsonpath.New("archived_at"), knownvalue.Null()),
				},
			},
		},
	})
}

func testAccFederationRuleDataSourceConfig(issuerName, serviceAccountName, ruleName string) string {
	return fmt.Sprintf(`
resource "anthropic_federation_issuer" "test" {
	name       = %[1]q
	issuer_url = "https://token.actions.githubusercontent.com"
	jwks = {
		type      = "inline"
		keys_json = jsonencode([{
			kty = "RSA"
			kid = "example-key-1"
			use = "sig"
			alg = "RS256"
			n   = "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw"
			e   = "AQAB"
		}])
	}
}

resource "anthropic_service_account" "test" {
	name = %[2]q
}

resource "anthropic_federation_rule" "test" {
	name      = %[3]q
	issuer_id = anthropic_federation_issuer.test.id
	match = {
		subject_prefix = "repo:example-org/example-repo:*"
	}
	target = {
		service_account_id = anthropic_service_account.test.id
	}
	applies_to_all_workspaces = true
}

data "anthropic_federation_rule" "test" {
	id = anthropic_federation_rule.test.id
}
`, issuerName, serviceAccountName, ruleName)
}
