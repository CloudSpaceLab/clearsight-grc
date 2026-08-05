package continuity

import (
	"encoding/json"
	"strings"
)

const (
	MatterAccessPublic     = "PUBLIC"
	MatterAccessInternal   = "INTERNAL"
	MatterAccessRestricted = "RESTRICTED"
)

type MatterAccessPolicy struct {
	Access              string   `json:"access"`
	AllowedPrincipalIDs []string `json:"allowed_principal_ids"`
}

// ParseMatterAccessPolicy converts Matter scope access metadata into a strict,
// fail-closed policy. Existing Matters without an explicit access field remain
// internal to their verified tenant. Once an access field is present it must be
// one of the supported values, and RESTRICTED records require a non-empty
// principal allow-list.
func ParseMatterAccessPolicy(raw json.RawMessage) (MatterAccessPolicy, bool) {
	if len(raw) == 0 || !json.Valid(raw) {
		return MatterAccessPolicy{}, false
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return MatterAccessPolicy{}, false
	}

	accessRaw, hasAccess := fields["access"]
	if !hasAccess {
		return MatterAccessPolicy{Access: MatterAccessInternal}, true
	}

	var access string
	if err := json.Unmarshal(accessRaw, &access); err != nil {
		return MatterAccessPolicy{}, false
	}
	access = strings.ToUpper(strings.TrimSpace(access))
	switch access {
	case MatterAccessPublic, MatterAccessInternal:
		return MatterAccessPolicy{Access: access}, true
	case MatterAccessRestricted:
		var values []string
		if rawValues, ok := fields["allowed_principal_ids"]; !ok || json.Unmarshal(rawValues, &values) != nil {
			return MatterAccessPolicy{}, false
		}
		seen := make(map[string]struct{}, len(values))
		allowed := make([]string, 0, len(values))
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if _, duplicate := seen[value]; duplicate {
				continue
			}
			seen[value] = struct{}{}
			allowed = append(allowed, value)
		}
		if len(allowed) == 0 {
			return MatterAccessPolicy{}, false
		}
		return MatterAccessPolicy{Access: access, AllowedPrincipalIDs: allowed}, true
	default:
		return MatterAccessPolicy{}, false
	}
}

func MatterVisibleTo(matter Matter, principalID string) bool {
	policy, valid := ParseMatterAccessPolicy(matter.Scope)
	if !valid {
		return false
	}
	if policy.Access != MatterAccessRestricted {
		return true
	}
	principalID = strings.TrimSpace(principalID)
	if principalID == "" {
		return false
	}
	for _, allowed := range policy.AllowedPrincipalIDs {
		if allowed == principalID {
			return true
		}
	}
	return false
}
