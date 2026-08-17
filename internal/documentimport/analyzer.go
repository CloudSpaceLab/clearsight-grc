package documentimport

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
)

var enumeratedLine = regexp.MustCompile(`(?i)^(?:[-*•▪]\s+|\(?[a-zivx]+\)\s+|\d+(?:\.\d+)*[.)]\s+)`)

var regulatorAction = regexp.MustCompile(`(?i)^\s*(?:\(?[a-zivx]+\)?[.)]\s+)?(?:the\s+)?commission\s+(?:must|shall|should|is\s+(?:required\s+)?to)\b`)
var danglingListLead = regexp.MustCompile(`(?i):\s*[ivx]+[.]?$`)
var bareEnumeratedModality = regexp.MustCompile(`(?i)^\s*\(?[a-zivx]+\)?[.)]\s+(?:must|shall|should|is\s+(?:required\s+)?to|are\s+(?:required\s+)?to)\b`)

type AnalysisResult struct {
	Proposals []Proposal
	Total     int
	Omitted   int
}

func Analyze(sections []Section) []Proposal {
	return AnalyzeBounded(sections, DefaultExtractionPolicy().MaxProposals).Proposals
}

func AnalyzeBounded(sections []Section, maximum int) AnalysisResult {
	if maximum <= 0 {
		maximum = DefaultExtractionPolicy().MaxProposals
	}
	result := AnalysisResult{Proposals: make([]Proposal, 0, min(maximum, 64))}
	seen := map[string]struct{}{}
	for _, section := range sections {
		for _, match := range analysisStatements(section.Text) {
			statement := strings.TrimSpace(match)
			if len(statement) < 18 || len(statement) > 1200 {
				continue
			}
			kind, confidence := classify(statement)
			if kind == "" {
				continue
			}
			obligation := ParseObligation(statement, kind)
			key := kind + "\x00" + strings.ToLower(statement)
			if obligation.Eligible {
				key = "obligation\x00" + obligation.Fingerprint
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result.Total++
			if len(result.Proposals) >= maximum {
				result.Omitted++
				continue
			}
			result.Proposals = append(result.Proposals, Proposal{
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
				Status:     ProposalPending,
				Obligation: &obligation,
			})
		}
	}
	return result
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
	case strings.HasPrefix(strings.TrimSpace(strings.ToLower(statement)), "whereas"):
		return "AUTHORITY_REFERENCE", 0.84
	case regulatorAction.MatchString(statement):
		return "AUTHORITY_REFERENCE", 0.84
	case bareEnumeratedModality.MatchString(statement):
		return "AUTHORITY_REFERENCE", 0.72
	case contains(" enforcement order", " sanction ", " penalty ", " maximum amount", " gross revenue", " default fee"):
		return "RISK_SIGNAL", 0.84
	case danglingListLead.MatchString(strings.TrimSpace(statement)):
		return "AUTHORITY_REFERENCE", 0.65
	case contains(" must ", " shall ", " should ", " is required to ", " are required to ", " is to ", " are to ", " required by ", " obligation "):
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

// analysisStatements reconstructs logical prose before sentence analysis.
// Layout-preserving PDF extraction intentionally retains printed line wraps;
// those wraps are not semantic boundaries and must not become obligations.
func analysisStatements(value string) []string {
	lines := strings.Split(normalizeText(value), "\n")
	paragraphs := make([]string, 0, len(lines)/2+1)
	var current strings.Builder
	flush := func() {
		text := strings.TrimSpace(current.String())
		current.Reset()
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	}
	for _, raw := range lines {
		line := strings.Join(strings.Fields(raw), " ")
		if line == "" {
			flush()
			continue
		}
		if analysisHeading(line) {
			flush()
			continue
		}
		if enumeratedLine.MatchString(line) && current.Len() > 0 {
			flush()
		}
		if current.Len() > 0 {
			text := current.String()
			if strings.HasSuffix(text, "-") && len(line) > 0 && unicode.IsLower(rune(line[0])) {
				current.WriteString(line)
			} else {
				current.WriteByte(' ')
				current.WriteString(line)
			}
		} else {
			current.WriteString(line)
		}
		if strings.HasSuffix(line, ".") || strings.HasSuffix(line, "?") || strings.HasSuffix(line, "!") || strings.HasSuffix(line, ";") {
			flush()
		}
	}
	flush()

	statements := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		statements = append(statements, splitAnalysisSentences(paragraph)...)
	}
	return statements
}

func splitAnalysisSentences(paragraph string) []string {
	statements := make([]string, 0, 2)
	start := 0
	for index := 0; index < len(paragraph); index++ {
		character := paragraph[index]
		if character != '.' && character != '!' && character != '?' {
			continue
		}
		if character == '.' && index > 0 && index+1 < len(paragraph) && paragraph[index-1] >= '0' && paragraph[index-1] <= '9' && paragraph[index+1] >= '0' && paragraph[index+1] <= '9' {
			continue
		}
		statement := strings.TrimSpace(paragraph[start : index+1])
		if statement != "" {
			statements = append(statements, statement)
		}
		start = index + 1
	}
	if remainder := strings.TrimSpace(paragraph[start:]); remainder != "" {
		statements = append(statements, remainder)
	}
	return statements
}

func analysisHeading(value string) bool {
	value = strings.TrimSpace(value)
	value = strings.TrimSpace(enumeratedLine.ReplaceAllString(value, ""))
	if value == "" || len(value) > 160 || strings.ContainsAny(value, ".!?;") {
		return false
	}
	letters, upper := 0, 0
	for _, character := range value {
		if !unicode.IsLetter(character) {
			continue
		}
		letters++
		if unicode.IsUpper(character) {
			upper++
		}
	}
	return letters >= 4 && float64(upper)/float64(letters) >= .8
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
