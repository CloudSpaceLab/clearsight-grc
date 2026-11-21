package continuity

// DecisionAuthority identifies the current record and responsibility used for
// a decision lifecycle transition. It is a requirement, not an authorization
// result; the caller must resolve current authority at the decision boundary.
type DecisionAuthority struct {
	ObjectType     string
	ObjectID       string
	Responsibility string
	Materiality    int
}

// DecisionAuthorityRequirement shares lifecycle validation and exact record
// selection between command preparation and the future transaction boundary.
func DecisionAuthorityRequirement(aggregate MatterAggregate, decisionType string, target DecisionStatus) (DecisionAuthority, error) {
	if err := ValidateDecisionLifecycle(aggregate.Decisions, decisionType, target); err != nil {
		return DecisionAuthority{}, err
	}
	policy, err := DecisionLifecyclePolicy(target)
	if err != nil {
		return DecisionAuthority{}, err
	}
	result := DecisionAuthority{ObjectType: "MATTER", ObjectID: aggregate.Matter.ID, Responsibility: policy.Responsibility, Materiality: max(policy.Materiality, aggregate.Matter.Priority)}
	if current := CurrentDecisionForType(aggregate.Decisions, decisionType); current != nil {
		result.ObjectType, result.ObjectID = "DECISION", current.ID
	}
	return result, nil
}
