package documentimport

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ConversionObjectID is stable for one accepted handoff and canonical target.
// It makes materialization replay-safe without adding a receipt or mapping table.
func ConversionObjectID(handoffID string, target ConversionTarget) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(handoffID) + "\x00document-proposal-conversion\x00" + string(target)))
	raw := append([]byte(nil), digest[:16]...)
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}

func ConversionCode(handoffID string, target ConversionTarget) string {
	compact := strings.ReplaceAll(ConversionObjectID(handoffID, target), "-", "")
	if len(compact) > 16 {
		compact = compact[:16]
	}
	prefix := "IMP"
	if target == ConversionControlObjective {
		prefix = "IMPCTRL"
	}
	return prefix + "-" + strings.ToUpper(compact)
}
