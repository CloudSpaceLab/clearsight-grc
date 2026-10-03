package main

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestRCSAImplementationAssessableUsesCurrentLifecycleWindow(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name  string
		value continuity.ControlImplementation
		want  bool
	}{
		{
			name: "implemented and current",
			value: continuity.ControlImplementation{
				Status: continuity.ImplementationImplemented, EffectiveFrom: past,
			},
			want: true,
		},
		{
			name: "planned and current",
			value: continuity.ControlImplementation{
				Status: continuity.ImplementationPlanned, EffectiveFrom: past,
			},
			want: true,
		},
		{
			name: "inactive",
			value: continuity.ControlImplementation{
				Status: continuity.ImplementationInactive, EffectiveFrom: past,
			},
		},
		{
			name: "retired",
			value: continuity.ControlImplementation{
				Status: continuity.ImplementationRetired, EffectiveFrom: past,
			},
		},
		{
			name: "not started",
			value: continuity.ControlImplementation{
				Status: continuity.ImplementationImplemented, EffectiveFrom: future,
			},
		},
		{
			name: "ended",
			value: continuity.ControlImplementation{
				Status: continuity.ImplementationImplemented, EffectiveFrom: past, EffectiveUntil: &past,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rcsaImplementationAssessable(tc.value, now); got != tc.want {
				t.Fatalf("assessable=%v want=%v value=%#v", got, tc.want, tc.value)
			}
		})
	}
}
