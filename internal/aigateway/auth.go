package aigateway

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

type authenticator struct {
	entries []ConfiguredWorkload
}

func newAuthenticator(entries []ConfiguredWorkload) *authenticator {
	copied := make([]ConfiguredWorkload, len(entries))
	copy(copied, entries)
	return &authenticator{entries: copied}
}

func (a *authenticator) authenticate(header string) (*Workload, error) {
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || len(parts[1]) < 8 || len(parts[1]) > 4096 {
		return nil, ErrUnauthorized
	}
	digest := sha256.Sum256([]byte(parts[1]))
	match := -1
	for index := range a.entries {
		equal := subtle.ConstantTimeCompare(digest[:], a.entries[index].KeyDigest[:])
		match = subtle.ConstantTimeSelect(equal, index, match)
	}
	if match < 0 {
		return nil, ErrUnauthorized
	}
	workload := a.entries[match].Workload
	workload.AllowedModels = cloneStringSet(workload.AllowedModels)
	return &workload, nil
}

func cloneStringSet(input map[string]struct{}) map[string]struct{} {
	if input == nil {
		return nil
	}
	output := make(map[string]struct{}, len(input))
	for value := range input {
		output[value] = struct{}{}
	}
	return output
}

func bearerDigestMatches(header string, expected *[sha256.Size]byte) bool {
	if expected == nil {
		return false
	}
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || len(parts[1]) < 8 || len(parts[1]) > 4096 {
		return false
	}
	digest := sha256.Sum256([]byte(parts[1]))
	return subtle.ConstantTimeCompare(digest[:], expected[:]) == 1
}

func authenticateRequest(request *http.Request, auth *authenticator) (*Workload, error) {
	return auth.authenticate(request.Header.Get("Authorization"))
}
