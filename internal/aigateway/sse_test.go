package aigateway

import (
	"errors"
	"strings"
	"testing"
)

func TestSSEReaderRejectsOversizedLineBeforeEventAllocation(t *testing.T) {
	reader := newSSEReader(strings.NewReader("data: "+strings.Repeat("x", 4096)+"\n\n"), 1024)
	if _, err := reader.next(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("oversized SSE error=%v", err)
	}
}
