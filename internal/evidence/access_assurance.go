package evidence

import "strings"

func validAccessAssurance(value AccessAssurance) bool {
	return value == AssuranceLinkPossession || value == AssuranceEmailVerified
}

func validWorkspaceAssurance(value AccessAssurance) bool {
	return validAccessAssurance(value) || value == AssuranceInternalAuthenticated
}

// Exactly one origin is permitted. Existing external and carried records have
// no explicit origin discriminator; internal records must identify their actor.
func validWorkspaceEditOrigin(origin, principal, session, route, carried string, assurance AccessAssurance) bool {
	if carried != "" {
		return origin == "" && principal == "" && session == "" && route == "" && validWorkspaceAssurance(assurance)
	}
	if origin == "INTERNAL" {
		return strings.TrimSpace(principal) != "" && principal == strings.TrimSpace(principal) && session == "" && route == "" && assurance == AssuranceInternalAuthenticated
	}
	return origin == "" && principal == "" && session != "" && route != "" && validAccessAssurance(assurance)
}
