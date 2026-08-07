package continuity

import "context"

// ProgramSourceStateRepository provides the current evidence-source denominator
// used by the Program-state projection. It is deliberately a narrow read
// contract so source health remains owned by the evidence domain rather than a
// second state engine inside continuity.
type ProgramSourceStateRepository interface {
	CurrentProgramSourceState(context.Context, string, string) (ProgramSourceState, error)
}
