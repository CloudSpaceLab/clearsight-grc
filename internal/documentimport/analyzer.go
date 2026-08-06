package documentimport

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
)

var sentenceBoundary = regexp.MustCompile(`(?m)([^.!?\n]+[.!?]?)`)

func Analyze(sections []Section) []Proposal {
	proposals := make([]Proposal, 0)
	seen := map[string]struct{}{}
	for _, section := range sections {
		for _, match := range sentenceBoundary.FindAllString(section.Text, -1) {
			statement := strings.TrimSpace(match)
			if len(statement) < 18 || len(statement) > 1200 {
				continue
			}
			kind, confidence := classify(statement)
			if kind == "" {
				continue
			}
			key := kind + "\x00" + strings.ToLower(statement)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			proposals = append(proposals, Proposal{
				ID:         stableProposalID(section.ID, key),
				Kind:       kind,
				Title:      proposalTitle(kind),
				Statement:  statement,
				Confidence: confidence,
				Anchor: Anchor{
					SectionID: section.ID,
					Quote:     statement,
					Page:      section.Page,
					Sheet:     section.Sheet,
					RowStart:  section.RowStart,
					RowEnd:    section.RowEnd,
				},
				Status: ProposalPending,
			})
			if len(proposals) >= 500 {
				return proposals
			}
		}
	}
	return proposals
}

func classify(statement string) (string, float64) {
	value := " " + strings.ToLower(statement) + " "
	contains := func(words ...string) bool {
		for _, word := range words {
			if strings.Contains(value, word) {
				return true
			}
		}
		return false
	}
	switch {
	case contains(" must ", " shall ", " is required to ", " are required to ", " required by ", " obligation "):
		return "REQUIREMENT_CANDIDATE", 0.86
	case contains(" within ", " no later than ", " deadline ", " business days", " calendar days", " annually", " quarterly", " monthly"):
		return "DEADLINE_CANDIDATE", 0.80
	case contains(" act ", " regulation ", " circular ", " directive ", " guideline ", " standard ", " section ", " article "):
		return "AUTHORITY_REFERENCE", 0.76
	case contains(" implement ", " maintain ", " monitor ", " review ", " approve ", " verify ", " retain ", " reconcile "):
		return "CONTROL_EXPECTATION", 0.73
	case contains(" breach ", " violation ", " failure ", " penalty ", " sanction ", " non-compliance", " risk ", " exposure "):
		return "RISK_SIGNAL", 0.70
	default:
		return "", 0
	}
}

func proposalTitle(kind string) string {
	switch kind {
	case "REQUIREMENT_CANDIDATE":
		return "Possible requirement"
	case "DEADLINE_CANDIDATE":
		return "Possible deadline"
	case "AUTHORITY_REFERENCE":
		return "Possible authority reference"
	case "CONTROL_EXPECTATION":
		return "Possible control expectation"
	case "RISK_SIGNAL":
		return "Possible risk or consequence"
	default:
		return titleCase(kind)
	}
}

func stableProposalID(sectionID, key string) string {
	digest := sha256.Sum256([]byte(sectionID + "\x00" + key))
	raw := digest[:16]
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}

func titleCase(value string) string {
	value = strings.ReplaceAll(strings.ToLower(value), "_", " ")
	characters := []rune(value)
	upper := true
	for index, character := range characters {
		if upper && unicode.IsLetter(character) {
			characters[index] = unicode.ToUpper(character)
			upper = false
		}
		if unicode.IsSpace(character) {
			upper = true
		}
	}
	return string(characters)
}
