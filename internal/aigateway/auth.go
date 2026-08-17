package aigateway

import (
	"context"
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

func (a *authenticator) Authenticate(ctx context.Context, header string) (*Workload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return a.authenticate(header)
}

func (a *authenticator) Ready() bool { return a != nil && len(a.entries) > 0 }

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
	workload := cloneWorkload(a.entries[match].Workload)
	workload.Policy = staticPolicy(workload)
	return &workload, nil
}

func cloneWorkload(input Workload) Workload {
	input.AllowedModels = cloneStringSet(input.AllowedModels)
	input.VerifiedMetadata = cloneStringMap(input.VerifiedMetadata)
	input.Policy.Definition.Bindings = append([]BindingRequirement(nil), input.Policy.Definition.Bindings...)
	input.Policy.Definition.Rules = append([]PolicyRule(nil), input.Policy.Definition.Rules...)
	return input
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
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

func authenticateRequest(request *http.Request, provider WorkloadProvider) (*Workload, error) {
	if provider == nil {
		return nil, ErrPolicyUnavailable
	}
	return provider.Authenticate(request.Context(), request.Header.Get("Authorization"))
}
