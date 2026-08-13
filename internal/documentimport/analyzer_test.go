package documentimport

import "testing"

func TestClassifyExcludesEnforcementConsequencesFromObligationDenominator(t *testing.T) {
	statements := []string{
		"(2) An enforcement order made or sanction imposed under subsection (1) shall include —",
		"(4) The higher maximum amount shall be the greater of N10,000,000 and 2% of annual gross revenue.",
		"Default fee which is 50% of the filing fee applies when a controller misses the deadline.",
	}
	for _, statement := range statements {
		kind, _ := classify(statement)
		if obligation := ParseObligation(statement, kind); obligation.Eligible {
			t.Fatalf("enforcement consequence must not count as an organization obligation: %q (%s)", statement, kind)
		}
	}
}

func TestClassifyExcludesRegulatorActionsAndIncompleteListLeads(t *testing.T) {
	statements := []string{
		"The Commission shall take note of the Memorandum as a bona fide commitment.",
		"a) In the report accompanying the audit questions, emphasis should be on: i.",
		"(b) shall inform the data controller in writing of its decision.",
		"WHEREAS, filing the annual return is an obligation under the regulation;",
	}
	for _, statement := range statements {
		kind, _ := classify(statement)
		if obligation := ParseObligation(statement, kind); obligation.Eligible {
			t.Fatalf("context must remain visible without entering the denominator: %q (%s)", statement, kind)
		}
	}
}

func TestClassifyKeepsOrganizationDutiesActionable(t *testing.T) {
	statements := []string{
		"Data Controllers and Data Processors are to file CAR with the Commission.",
		"All designated DPOs are required to participate in induction training.",
	}
	for _, statement := range statements {
		kind, _ := classify(statement)
		if obligation := ParseObligation(statement, kind); !obligation.Eligible {
			t.Fatalf("organization duty must remain actionable: %q (%s)", statement, kind)
		}
	}
}
