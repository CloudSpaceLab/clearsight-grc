package evidence

import "strings"

// CanonicalSubjectTypeRegistry is the closed set backed by exact authoritative
// subject-scope lookups. A type cannot become automation-eligible merely
// because an authority policy contains a wildcard route for its name.
type CanonicalSubjectTypeRegistry struct{}

func (CanonicalSubjectTypeRegistry) SupportsSubjectType(subjectType string) bool {
	switch strings.ToUpper(strings.TrimSpace(subjectType)) {
	case "PROGRAM", "MATTER", "VENDOR_RELATIONSHIP":
		return true
	default:
		return false
	}
}
