package reporting

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func renderMatterBoardBriefPDF(run ReportRun, row ReportRow) ([]byte, error) {
	if run.Dataset != DatasetMatterBoardBrief || run.Format != FormatPDF || run.ScopeKind != ScopeMatter || row.ID != run.ScopeRef {
		return nil, ErrInvalid
	}
	lines := []string{
		"ClearSight Board Brief",
		"",
		fmt.Sprintf("Issue: %s — %s", boardString(row.Values["reference"]), boardString(row.Values["title"])),
		fmt.Sprintf("Status: %s    Version: %s    Impact: %s", boardString(row.Values["status"]), boardValue(row.Values["version"]), boardValue(row.Values["priority"])),
		fmt.Sprintf("Applies to: %s", boardFallback(row.Values["organization_scope"], "Not recorded")),
		fmt.Sprintf("Affected service or process: %s", boardFallback(row.Values["affected_area"], "Not recorded")),
		fmt.Sprintf("Owner: %s", boardFallback(row.Values["owner_name"], "Not assigned")),
		fmt.Sprintf("Due: %s", boardDate(row.Values["due_at"])),
		"",
		"Position",
		boardFallback(row.Values["summary"], "No summary recorded."),
	}
	lines = appendBoardSection(lines, "Programs", row.Values["programs"], []string{"code", "name"})
	lines = appendBoardSection(lines, "Actions", row.Values["actions"], []string{"title", "status", "owner", "due_at"})
	lines = appendBoardSection(lines, "Decisions", row.Values["decisions"], []string{"type", "status", "selected_option", "decided_at"})
	lines = appendBoardSection(lines, "Outcome checks", row.Values["outcomes"], []string{"result", "observed_at", "rationale"})
	lines = appendBoardSection(lines, "Loss and recovery", row.Values["losses"], []string{"code", "title", "gross", "recovered", "net", "currency"})
	lines = appendBoardSection(lines, "Forms and responses", row.Values["forms"], []string{"title", "status", "deadline", "response_state", "concern"})
	lines = appendBoardSection(lines, "Vendor work", row.Values["vendor_work"], []string{"purpose", "state", "due_at"})
	lines = appendBoardSection(lines, "Connected source", row.Values["source_context"], []string{"type", "observed_at", "completeness", "records"})
	lines = append(lines, "", fmt.Sprintf("As of: %s", run.AsOf.UTC().Format(time.RFC3339)), fmt.Sprintf("Report receipt: %s / definition v%d", run.ID, run.DefinitionVersion))
	return renderSimplePDF(lines)
}

func appendBoardSection(lines []string, heading string, value any, fields []string) []string {
	lines = append(lines, "", heading)
	items, ok := value.([]any)
	if !ok {
		if one, single := value.(map[string]any); single {
			items = []any{one}
		}
	}
	if len(items) == 0 {
		return append(lines, "Not recorded or unavailable.")
	}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		parts := make([]string, 0, len(fields))
		for _, field := range fields {
			if text := boardValue(entry[field]); text != "" {
				parts = append(parts, humanBoardField(field)+": "+text)
			}
		}
		if len(parts) > 0 {
			lines = append(lines, "• "+strings.Join(parts, " · "))
		}
	}
	if lines[len(lines)-1] == heading {
		lines = append(lines, "Not recorded or unavailable.")
	}
	return lines
}

func humanBoardField(value string) string {
	value = strings.ReplaceAll(value, "_", " ")
	parts := strings.Fields(value)
	for index := range parts {
		parts[index] = strings.ToUpper(parts[index][:1]) + parts[index][1:]
	}
	return strings.Join(parts, " ")
}

func boardFallback(value any, fallback string) string {
	if text := boardValue(value); text != "" {
		return text
	}
	return fallback
}

func boardDate(value any) string {
	switch typed := value.(type) {
	case time.Time:
		return typed.UTC().Format("02 Jan 2006")
	case string:
		if parsed, err := time.Parse(time.RFC3339, typed); err == nil {
			return parsed.UTC().Format("02 Jan 2006")
		}
		return typed
	default:
		return boardValue(value)
	}
}

func boardString(value any) string { return strings.TrimSpace(boardValue(value)) }

func boardValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case time.Time:
		return typed.UTC().Format(time.RFC3339)
	case fmt.Stringer:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func renderSimplePDF(lines []string) ([]byte, error) {
	const maxLinesPerPage = 54
	rendered := make([]string, 0, len(lines)*2)
	for _, raw := range lines {
		wrapped := wrapPDFText(raw, 92)
		if len(wrapped) == 0 {
			rendered = append(rendered, "")
			continue
		}
		rendered = append(rendered, wrapped...)
	}
	if len(rendered) == 0 {
		rendered = []string{""}
	}
	pageCount := (len(rendered) + maxLinesPerPage - 1) / maxLinesPerPage
	pageObjectStart := 3
	contentObjectStart := pageObjectStart + pageCount
	fontObjectID := contentObjectStart + pageCount
	kids := make([]string, 0, pageCount)
	objects := make([]string, 0, 3+pageCount*2)
	objects = append(objects, "<< /Type /Catalog /Pages 2 0 R >>")
	for page := 0; page < pageCount; page++ {
		kids = append(kids, fmt.Sprintf("%d 0 R", pageObjectStart+page))
	}
	objects = append(objects, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), pageCount))

	streams := make([]string, 0, pageCount)
	for page := 0; page < pageCount; page++ {
		first := page * maxLinesPerPage
		last := first + maxLinesPerPage
		if last > len(rendered) {
			last = len(rendered)
		}
		content := &strings.Builder{}
		content.WriteString("BT\n/F1 10 Tf\n48 790 Td\n13 TL\n")
		for index, line := range rendered[first:last] {
			if index > 0 {
				content.WriteString("T*\n")
			}
			content.WriteString("(")
			content.WriteString(escapePDFText(line))
			content.WriteString(") Tj\n")
		}
		content.WriteString("ET\n")
		streams = append(streams, content.String())
		objects = append(objects, fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 %d 0 R >> >> /Contents %d 0 R >>",
			fontObjectID, contentObjectStart+page,
		))
	}
	for _, stream := range streams {
		objects = append(objects, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(stream), stream))
	}
	objects = append(objects, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")

	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}
	xref := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for index := 1; index <= len(objects); index++ {
		fmt.Fprintf(&output, "%010d 00000 n \n", offsets[index])
	}
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	if output.Len() > MaxReportRunBytes {
		return nil, &reportLimitError{Code: FailureByteLimitExceeded, Limit: "bytes"}
	}
	return output.Bytes(), nil
}

func wrapPDFText(value string, limit int) []string {
	value = strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' {
			return ' '
		}
		if r > 126 {
			switch r {
			case '•':
				return '-'
			case '—', '–':
				return '-'
			default:
				return '?'
			}
		}
		return r
	}, value))
	if value == "" {
		return nil
	}
	words := strings.Fields(value)
	lines := make([]string, 0, 1)
	var current strings.Builder
	for _, word := range words {
		if current.Len() > 0 && current.Len()+1+len(word) > limit {
			lines = append(lines, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(word)
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}

func escapePDFText(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)")
	return replacer.Replace(value)
}
