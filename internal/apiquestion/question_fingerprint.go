package apiquestion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

// questionFingerprint hashes the complete typed field and expected response
// type for a future transition manifest. It is currently unused. Its caller
// must load a validated, normalized form revision under the transition
// transaction. It does not cover unknown JSON properties, section-level
// conditions or scoring profiles and is not a substitute for validating and
// pinning the full form revision, source proof or approval.
func questionFingerprint(field formcontract.Field, resultType string) (string, error) {
	if !manifestID(field.ID) || len(field.ID) > 80 || !manifestID(field.SectionID) || len(field.SectionID) > 80 || strings.TrimSpace(field.Label) == "" || len(field.Label) > 200 {
		return "", ErrInvalid
	}
	if !CompatibleQuestionResult(string(field.Type), resultType) {
		return "", ErrInvalid
	}
	encoded, err := json.Marshal(struct {
		Schema     string             `json:"schema"`
		ResultType string             `json:"result_type"`
		Question   formcontract.Field `json:"question"`
	}{"form-api-question-definition-v1", resultType, field})
	if err != nil || len(encoded) > 128*1024 {
		return "", ErrInvalid
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
