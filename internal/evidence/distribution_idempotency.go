package evidence

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type distributionCreationReceipt struct{ ID, Checksum string }

func distributionCreationChecksum(input CreateDistributionInput) string {
	raw, _ := json.Marshal(input)
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}
func distributionCreationKey(input CreateDistributionInput) string {
	return input.TenantID + "\x00" + input.LegalEntityID + "\x00" + input.IdempotencyKey
}
