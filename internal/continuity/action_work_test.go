package continuity

import (
	"reflect"
	"testing"
)

func TestAllowedActionTargetsUsesCanonicalLifecycle(t *testing.T) {
	tests := []struct {
		from ActionStatus
		want []ActionStatus
	}{
		{ActionPlanned, []ActionStatus{ActionInProgress, ActionBlocked, ActionCancelled}},
		{ActionInProgress, []ActionStatus{ActionImplemented, ActionBlocked, ActionCancelled}},
		{ActionBlocked, []ActionStatus{ActionInProgress, ActionCancelled}},
		{ActionImplemented, []ActionStatus{}},
		{ActionCancelled, []ActionStatus{}},
	}
	for _, test := range tests {
		if got := AllowedActionTargets(test.from); !reflect.DeepEqual(got, test.want) {
			t.Fatalf("AllowedActionTargets(%s)=%v want %v", test.from, got, test.want)
		}
	}
}
