package ropa

// ReplayActivityEvents rebuilds the current processing activity from the
// complete event sequence. It deliberately starts from an empty aggregate and
// accepts only a contiguous version-1..N sequence whose snapshots pass the
// same operation and payload checks as the command boundary.
func ReplayActivityEvents(events []Event) (ProcessingActivity, error) {
	if len(events) == 0 {
		return ProcessingActivity{}, ErrInvalid
	}

	first, err := decodeActivity(events[0].Payload)
	if err != nil {
		return ProcessingActivity{}, err
	}
	if err := validateActivityWhitespace(first); err != nil {
		return ProcessingActivity{}, err
	}
	first = normalizeProcessingActivity(first)
	if first.Version != 1 || first.Status != StatusNew {
		return ProcessingActivity{}, ErrInvalid
	}
	if _, err := validateCreatedActivityEvent(events[0], first); err != nil {
		return ProcessingActivity{}, err
	}
	current := first

	for index := 1; index < len(events); index++ {
		if events[index].AggregateVersion != current.Version+1 {
			return ProcessingActivity{}, ErrInvalid
		}
		next, _, err := prepareActivityEvent(current, current.Version, events[index])
		if err != nil {
			return ProcessingActivity{}, err
		}
		current = next
	}
	return cloneProcessingActivity(current), nil
}
