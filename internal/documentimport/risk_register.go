package documentimport

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type RiskRegister struct {
	Groups []RiskRegisterGroup `json:"groups"`
	Rows   []RiskRegisterRow   `json:"rows"`
}
type RiskRegisterGroup struct {
	ID             string `json:"id"`
	Vendor         string `json:"vendor"`
	Service        string `json:"service"`
	AssessmentDate string `json:"assessment_date"`
}
type RiskRegisterRow struct {
	ID               string            `json:"id"`
	GroupID          string            `json:"group_id"`
	Finding          string            `json:"finding"`
	Recommendation   string            `json:"recommendation"`
	Responsibility   string            `json:"responsibility"`
	RecordedStatus   string            `json:"recorded_status"`
	SuggestedDueDate string            `json:"suggested_due_date"`
	Original         map[string]string `json:"original"`
	Inherited        []string          `json:"inherited"`
	Anchor           SourceAnchor      `json:"anchor"`
}

// ParseRiskRegister proposes continuation groups. The caller must review these
// suggestions before materialization; original cells and anchors are unchanged.
func ParseRiskRegister(d Document) (RiskRegister, error) {
	p := RiskRegister{Groups: []RiskRegisterGroup{}, Rows: []RiskRegisterRow{}}
	if d.ExtractionStatus != ExtractionExtracted || d.ContentTruncated || d.SectionsOmitted > 0 || d.SectionsTotal != len(d.Sections) || len(d.Degradations) > 0 {
		return p, errors.New("Complete source rows are required. Upload the complete register again.")
	}
	if d.Tabular != nil && (d.Tabular.FatalError != "" || d.Tabular.RowsRejected > 0) {
		return p, errors.New("Some register rows could not be read. Correct the file and upload it again.")
	}
	sheets := map[string][]ExtractedElement{}
	order := []string{}
	for _, e := range d.Elements {
		if e.Kind == ElementTable && e.Anchor.Sheet != "" {
			if _, ok := sheets[e.Anchor.Sheet]; !ok {
				order = append(order, e.Anchor.Sheet)
			}
			sheets[e.Anchor.Sheet] = append(sheets[e.Anchor.Sheet], e)
		}
	}
	// Every extracted source row must retain its structured cells. Missing
	// elements must not silently reduce the migration population.
	locations := map[string]bool{}
	for sheet, rows := range sheets {
		for _, row := range rows {
			locations[fmt.Sprintf("%s:%d:%d", sheet, row.Anchor.RowStart, row.Anchor.RowEnd)] = true
		}
	}
	for _, section := range d.Sections {
		if section.Sheet != "" && section.RowStart > 0 && !locations[fmt.Sprintf("%s:%d:%d", section.Sheet, section.RowStart, section.RowEnd)] {
			return p, errors.New("A register row is missing its source cells. Upload the complete file again.")
		}
	}
	for _, sheet := range order {
		rows := sheets[sheet]
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Anchor.RowStart < rows[j].Anchor.RowStart })
		columns := map[string]int{}
		headers := []string{}
		group := ""
		owner := ""
		serial, provider, service, assessmentDate := "", "", "", ""
		lastRow := 0
		for _, e := range rows {
			if len(e.Values) != 1 || e.Anchor.RowStart < 1 || e.Anchor.RowStart != e.Anchor.RowEnd || e.Anchor.RowStart == lastRow {
				return p, errors.New("Register rows have missing or conflicting source locations. Upload the complete file again.")
			}
			lastRow = e.Anchor.RowStart
			values := e.Values[0]
			candidate := map[string]int{}
			duplicate := false
			for i, h := range values {
				key := spreadsheetHeader(h)
				if _, ok := candidate[key]; ok && key != "" {
					duplicate = true
				}
				candidate[key] = i
			}
			_, hasFinding := candidate["findings"]
			_, hasVendor := candidate["service provider"]
			_, hasService := candidate["services offered"]
			_, hasRecommendation := candidate["recommendations"]
			if hasFinding && hasVendor && hasService && hasRecommendation {
				if duplicate {
					return p, errors.New("Give each register column a distinct heading.")
				}
				columns = candidate
				headers = values
				group = ""
				owner = ""
				serial, provider, service, assessmentDate = "", "", "", ""
				continue
			}
			if len(headers) == 0 {
				continue
			}
			cell := func(key string) string { return findingCell(values, columns, key) }
			changedContext := (cell("service provider") != "" && cell("service provider") != provider) || (cell("services offered") != "" && cell("services offered") != service) || (cell("date of assessment") != "" && cell("date of assessment") != assessmentDate)
			if serial != "" && cell("s n") == serial && changedContext {
				return p, findingRowError(e.Anchor, "has conflicting vendor, service or date for the same assessment number")
			}
			if group == "" || (cell("s n") != "" && cell("s n") != serial) || changedContext {
				if cell("service provider") == "" || cell("services offered") == "" {
					return p, findingRowError(e.Anchor, "needs its service provider and service")
				}
				group = stableFormProposalID("register-group", d.SHA256, sheet, fmt.Sprint(e.Anchor.RowStart))
				owner = cell("responsibility")
				serial, provider, service, assessmentDate = cell("s n"), cell("service provider"), cell("services offered"), cell("date of assessment")
				p.Groups = append(p.Groups, RiskRegisterGroup{ID: group, Vendor: cell("service provider"), Service: cell("services offered"), AssessmentDate: cell("date of assessment")})
			}
			if cell("findings") == "" {
				continue
			}
			if group == "" {
				return p, findingRowError(e.Anchor, "needs a vendor and service before its findings")
			}
			if len(p.Rows) >= 100 {
				return p, errors.New("Import up to 100 findings at a time. Split the register into complete assessments.")
			}
			original := map[string]string{}
			for i, h := range headers {
				v := ""
				if i < len(values) {
					v = values[i]
				}
				original[strings.TrimSpace(h)] = v
			}
			inherited := []string{}
			responsibility := cell("responsibility")
			if cell("service provider") == "" {
				inherited = append(inherited, "Vendor and service")
			}
			if responsibility == "" && owner != "" {
				responsibility = owner
				inherited = append(inherited, "Responsibility")
			}
			p.Rows = append(p.Rows, RiskRegisterRow{ID: stableFormProposalID("register-row", d.SHA256, sheet, fmt.Sprint(e.Anchor.RowStart)), GroupID: group, Finding: cell("findings"), Recommendation: cell("recommendations"), Responsibility: responsibility, RecordedStatus: cell("status"), SuggestedDueDate: registerDate(cell("timeline")), Original: original, Inherited: inherited, Anchor: e.Anchor})
		}
	}
	if len(p.Rows) == 0 {
		return p, errors.New("No vendor findings were found. Include Service provider, Services offered, Findings and Recommendations columns.")
	}
	return p, nil
}

var ordinalDay = regexp.MustCompile(`(?i)(\d+)(st|nd|rd|th)\b`)

func registerDate(value string) string {
	value = strings.Join(strings.Fields(ordinalDay.ReplaceAllString(strings.TrimSpace(value), "${1}")), " ")
	for _, layout := range []string{"2006-01-02", "2 January 2006", "January 2 2006", "2 Jan 2006", "January 2, 2006"} {
		if date, err := time.Parse(layout, value); err == nil {
			return date.Format("2006-01-02")
		}
	}
	return ""
}
