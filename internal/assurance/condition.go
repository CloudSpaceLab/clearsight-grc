package assurance

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type Operator string

const (
	OpAnd      Operator = "AND"
	OpOr       Operator = "OR"
	OpNot      Operator = "NOT"
	OpExists   Operator = "EXISTS"
	OpMissing  Operator = "MISSING"
	OpEQ       Operator = "EQ"
	OpNEQ      Operator = "NEQ"
	OpIn       Operator = "IN"
	OpNotIn    Operator = "NOT_IN"
	OpGT       Operator = "GT"
	OpGTE      Operator = "GTE"
	OpLT       Operator = "LT"
	OpLTE      Operator = "LTE"
	OpBetween  Operator = "BETWEEN"
	OpContains Operator = "CONTAINS"
)

type Literal struct {
	Type   LogicalType `json:"type"`
	String string      `json:"string,omitempty"`
	Number float64     `json:"number,omitempty"`
	Bool   bool        `json:"bool,omitempty"`
	Time   time.Time   `json:"time,omitempty"`
}

func StringLiteral(value string) Literal  { return Literal{Type: TypeString, String: value} }
func NumberLiteral(value float64) Literal { return Literal{Type: TypeNumber, Number: value} }
func BoolLiteral(value bool) Literal      { return Literal{Type: TypeBool, Bool: value} }
func TimeLiteral(value time.Time) Literal { return Literal{Type: TypeTime, Time: value.UTC()} }

type Condition struct {
	Op         Operator    `json:"op"`
	Field      string      `json:"field,omitempty"`
	OtherField string      `json:"other_field,omitempty"`
	Value      Literal     `json:"value,omitempty"`
	Values     []Literal   `json:"values,omitempty"`
	Children   []Condition `json:"children,omitempty"`
}

const (
	hardMaxConditionDepth        = 16
	hardMaxConditionNodes        = 256
	hardMaxConditionInValues     = 256
	hardMaxConditionStringBytes  = 4096
	hardMaxConditionTotalValues  = 1024
	hardMaxConditionLiteralBytes = 64 << 10
)

type ConditionLimits struct {
	MaxDepth       int
	MaxNodes       int
	MaxInValues    int
	MaxStringBytes int
}

func DefaultConditionLimits() ConditionLimits {
	return ConditionLimits{MaxDepth: 8, MaxNodes: 64, MaxInValues: 64, MaxStringBytes: 512}
}

type Result string

const (
	ResultMatch   Result = "MATCH"
	ResultClear   Result = "CLEAR"
	ResultUnknown Result = "UNKNOWN"
)

type CompiledCondition struct {
	condition    Condition
	fields       map[string]Field
	dependencies []string
}

func CompileCondition(schema Schema, condition Condition, limits ConditionLimits) (*CompiledCondition, error) {
	if err := schema.Validate(); err != nil {
		return nil, err
	}
	limits = normalizedConditionLimits(limits)
	fields := make(map[string]Field, len(schema.Fields))
	for _, field := range schema.Fields {
		fields[field.Name] = field
	}

	state := validationState{fields: fields, limits: limits}
	if err := state.validate(condition, 1); err != nil {
		return nil, err
	}
	dependencies := make([]string, 0, len(state.dependencies))
	for field := range state.dependencies {
		dependencies = append(dependencies, field)
	}
	sort.Strings(dependencies)
	return &CompiledCondition{
		condition:    cloneCondition(condition),
		fields:       fields,
		dependencies: dependencies,
	}, nil
}

func (c *CompiledCondition) Dependencies() []string {
	if c == nil {
		return nil
	}
	return append([]string(nil), c.dependencies...)
}

func (c *CompiledCondition) Evaluate(row map[string]any) Result {
	if c == nil {
		return ResultUnknown
	}
	return c.evaluate(c.condition, row)
}

func (c *CompiledCondition) evaluate(condition Condition, row map[string]any) Result {
	switch condition.Op {
	case OpAnd:
		seenUnknown := false
		for _, child := range condition.Children {
			switch c.evaluate(child, row) {
			case ResultClear:
				return ResultClear
			case ResultUnknown:
				seenUnknown = true
			}
		}
		if seenUnknown {
			return ResultUnknown
		}
		return ResultMatch
	case OpOr:
		seenUnknown := false
		for _, child := range condition.Children {
			switch c.evaluate(child, row) {
			case ResultMatch:
				return ResultMatch
			case ResultUnknown:
				seenUnknown = true
			}
		}
		if seenUnknown {
			return ResultUnknown
		}
		return ResultClear
	case OpNot:
		result := c.evaluate(condition.Children[0], row)
		switch result {
		case ResultMatch:
			return ResultClear
		case ResultClear:
			return ResultMatch
		default:
			return ResultUnknown
		}
	case OpExists, OpMissing:
		value, exists := row[condition.Field]
		present := exists && value != nil
		if condition.Op == OpExists {
			if present {
				return ResultMatch
			}
			return ResultClear
		}
		if present {
			return ResultClear
		}
		return ResultMatch
	}

	field := c.fields[condition.Field]
	leftRaw, exists := row[condition.Field]
	if !exists || leftRaw == nil {
		return ResultUnknown
	}
	left, ok := coerceValue(field.Type, leftRaw)
	if !ok {
		return ResultUnknown
	}

	if condition.Op == OpIn || condition.Op == OpNotIn {
		matched := false
		for _, literal := range condition.Values {
			if compareTyped(OpEQ, left, valueFromLiteral(literal)) {
				matched = true
				break
			}
		}
		if condition.Op == OpNotIn {
			matched = !matched
		}
		return boolResult(matched)
	}
	if condition.Op == OpBetween {
		lower := valueFromLiteral(condition.Values[0])
		upper := valueFromLiteral(condition.Values[1])
		return boolResult(compareTyped(OpGTE, left, lower) && compareTyped(OpLTE, left, upper))
	}
	if condition.Op == OpContains {
		return boolResult(strings.Contains(left.stringValue, condition.Value.String))
	}

	var right typedValue
	if condition.OtherField != "" {
		rightRaw, rightExists := row[condition.OtherField]
		if !rightExists || rightRaw == nil {
			return ResultUnknown
		}
		rightField := c.fields[condition.OtherField]
		var rightOK bool
		right, rightOK = coerceValue(rightField.Type, rightRaw)
		if !rightOK {
			return ResultUnknown
		}
	} else {
		right = valueFromLiteral(condition.Value)
	}
	return boolResult(compareTyped(condition.Op, left, right))
}

type validationState struct {
	fields            map[string]Field
	limits            ConditionLimits
	nodes             int
	dependencies      map[string]struct{}
	totalValues       int
	totalLiteralBytes int
}

func (s *validationState) validate(condition Condition, depth int) error {
	if s.dependencies == nil {
		s.dependencies = make(map[string]struct{})
	}
	s.nodes++
	if s.nodes > s.limits.MaxNodes {
		return fmt.Errorf("condition exceeds %d nodes", s.limits.MaxNodes)
	}
	if depth > s.limits.MaxDepth {
		return fmt.Errorf("condition exceeds depth %d", s.limits.MaxDepth)
	}

	switch condition.Op {
	case OpAnd, OpOr:
		if len(condition.Children) < 2 {
			return fmt.Errorf("%s requires at least two children", condition.Op)
		}
		if condition.Field != "" || condition.OtherField != "" || condition.Value.Type != "" || len(condition.Values) != 0 {
			return fmt.Errorf("%s cannot contain scalar operands", condition.Op)
		}
		for _, child := range condition.Children {
			if err := s.validate(child, depth+1); err != nil {
				return err
			}
		}
		return nil
	case OpNot:
		if len(condition.Children) != 1 {
			return fmt.Errorf("NOT requires exactly one child")
		}
		if condition.Field != "" || condition.OtherField != "" || condition.Value.Type != "" || len(condition.Values) != 0 {
			return fmt.Errorf("NOT cannot contain scalar operands")
		}
		return s.validate(condition.Children[0], depth+1)
	}

	if len(condition.Children) != 0 {
		return fmt.Errorf("%s cannot contain child conditions", condition.Op)
	}
	field, ok := s.fields[condition.Field]
	if condition.Field == "" || !ok {
		return fmt.Errorf("condition references unknown field %q", condition.Field)
	}
	s.dependencies[condition.Field] = struct{}{}

	switch condition.Op {
	case OpExists, OpMissing:
		if condition.OtherField != "" || condition.Value.Type != "" || len(condition.Values) != 0 {
			return fmt.Errorf("%s accepts only a field", condition.Op)
		}
		return nil
	case OpIn, OpNotIn:
		if condition.OtherField != "" || condition.Value.Type != "" {
			return fmt.Errorf("%s accepts a field and values", condition.Op)
		}
		if len(condition.Values) == 0 || len(condition.Values) > s.limits.MaxInValues {
			return fmt.Errorf("%s requires 1-%d values", condition.Op, s.limits.MaxInValues)
		}
		for _, value := range condition.Values {
			if err := s.validateLiteral(field.Type, value); err != nil {
				return fmt.Errorf("%s field %q: %w", condition.Op, field.Name, err)
			}
		}
		return nil
	case OpBetween:
		if condition.OtherField != "" || condition.Value.Type != "" || len(condition.Values) != 2 {
			return fmt.Errorf("BETWEEN requires exactly two literal values")
		}
		if field.Type != TypeNumber && field.Type != TypeTime {
			return fmt.Errorf("BETWEEN requires NUMBER or TIME field")
		}
		for _, value := range condition.Values {
			if err := s.validateLiteral(field.Type, value); err != nil {
				return err
			}
		}
		if compareTyped(OpGT, valueFromLiteral(condition.Values[0]), valueFromLiteral(condition.Values[1])) {
			return fmt.Errorf("BETWEEN lower value must not exceed upper value")
		}
		return nil
	case OpContains:
		if field.Type != TypeString {
			return fmt.Errorf("CONTAINS requires STRING field")
		}
		if condition.OtherField != "" || len(condition.Values) != 0 {
			return fmt.Errorf("CONTAINS requires one string literal")
		}
		if err := s.validateLiteral(TypeString, condition.Value); err != nil {
			return err
		}
		if condition.Value.String == "" {
			return fmt.Errorf("CONTAINS requires a non-empty string literal")
		}
		return nil
	case OpEQ, OpNEQ, OpGT, OpGTE, OpLT, OpLTE:
		if len(condition.Values) != 0 {
			return fmt.Errorf("%s does not accept values list", condition.Op)
		}
		if condition.OtherField != "" {
			if condition.Value.Type != "" {
				return fmt.Errorf("%s must use either value or other_field, not both", condition.Op)
			}
			right, exists := s.fields[condition.OtherField]
			if !exists {
				return fmt.Errorf("condition references unknown other_field %q", condition.OtherField)
			}
			if right.Type != field.Type {
				return fmt.Errorf("field comparison requires matching logical types")
			}
			s.dependencies[condition.OtherField] = struct{}{}
		} else if err := s.validateLiteral(field.Type, condition.Value); err != nil {
			return err
		}
		if orderedOperator(condition.Op) && field.Type != TypeNumber && field.Type != TypeTime {
			return fmt.Errorf("%s requires NUMBER or TIME field", condition.Op)
		}
		return nil
	default:
		return fmt.Errorf("unsupported condition operator %q", condition.Op)
	}
}

func (s *validationState) validateLiteral(expected LogicalType, value Literal) error {
	if err := validateLiteral(expected, value, s.limits); err != nil {
		return err
	}
	s.totalValues++
	if s.totalValues > hardMaxConditionTotalValues {
		return fmt.Errorf("condition exceeds %d total literal values", hardMaxConditionTotalValues)
	}
	s.totalLiteralBytes += literalBudgetBytes(value)
	if s.totalLiteralBytes > hardMaxConditionLiteralBytes {
		return fmt.Errorf("condition literal payload exceeds %d bytes", hardMaxConditionLiteralBytes)
	}
	return nil
}

func literalBudgetBytes(value Literal) int {
	switch value.Type {
	case TypeString:
		return len(value.String)
	case TypeNumber, TypeTime:
		return 16
	case TypeBool:
		return 1
	default:
		return 0
	}
}

func normalizedConditionLimits(value ConditionLimits) ConditionLimits {
	defaults := DefaultConditionLimits()
	if value.MaxDepth <= 0 {
		value.MaxDepth = defaults.MaxDepth
	}
	if value.MaxNodes <= 0 {
		value.MaxNodes = defaults.MaxNodes
	}
	if value.MaxInValues <= 0 {
		value.MaxInValues = defaults.MaxInValues
	}
	if value.MaxStringBytes <= 0 {
		value.MaxStringBytes = defaults.MaxStringBytes
	}
	value.MaxDepth = minPositive(value.MaxDepth, hardMaxConditionDepth)
	value.MaxNodes = minPositive(value.MaxNodes, hardMaxConditionNodes)
	value.MaxInValues = minPositive(value.MaxInValues, hardMaxConditionInValues)
	value.MaxStringBytes = minPositive(value.MaxStringBytes, hardMaxConditionStringBytes)
	return value
}

func validateLiteral(expected LogicalType, value Literal, limits ConditionLimits) error {
	if value.Type != expected {
		return fmt.Errorf("literal type %q does not match field type %q", value.Type, expected)
	}
	switch value.Type {
	case TypeString:
		if len(value.String) > limits.MaxStringBytes {
			return fmt.Errorf("string literal exceeds %d bytes", limits.MaxStringBytes)
		}
	case TypeNumber:
		if math.IsNaN(value.Number) || math.IsInf(value.Number, 0) {
			return fmt.Errorf("number literal must be finite")
		}
	case TypeBool:
	case TypeTime:
		if value.Time.IsZero() {
			return fmt.Errorf("time literal is required")
		}
	default:
		return fmt.Errorf("unsupported literal type %q", value.Type)
	}
	return nil
}

func orderedOperator(op Operator) bool {
	switch op {
	case OpGT, OpGTE, OpLT, OpLTE:
		return true
	default:
		return false
	}
}

func boolResult(value bool) Result {
	if value {
		return ResultMatch
	}
	return ResultClear
}

func cloneCondition(value Condition) Condition {
	clone := value
	clone.Values = append([]Literal(nil), value.Values...)
	clone.Children = make([]Condition, len(value.Children))
	for index := range value.Children {
		clone.Children[index] = cloneCondition(value.Children[index])
	}
	return clone
}

type typedValue struct {
	kind        LogicalType
	stringValue string
	numberValue float64
	boolValue   bool
	timeValue   time.Time
}

func valueFromLiteral(value Literal) typedValue {
	return typedValue{kind: value.Type, stringValue: value.String, numberValue: value.Number, boolValue: value.Bool, timeValue: value.Time.UTC()}
}

func compareTyped(op Operator, left, right typedValue) bool {
	if left.kind != right.kind {
		return false
	}
	switch left.kind {
	case TypeString:
		switch op {
		case OpEQ:
			return left.stringValue == right.stringValue
		case OpNEQ:
			return left.stringValue != right.stringValue
		}
	case TypeNumber:
		return compareOrdered(op, left.numberValue, right.numberValue)
	case TypeBool:
		switch op {
		case OpEQ:
			return left.boolValue == right.boolValue
		case OpNEQ:
			return left.boolValue != right.boolValue
		}
	case TypeTime:
		switch op {
		case OpEQ:
			return left.timeValue.Equal(right.timeValue)
		case OpNEQ:
			return !left.timeValue.Equal(right.timeValue)
		case OpGT:
			return left.timeValue.After(right.timeValue)
		case OpGTE:
			return left.timeValue.After(right.timeValue) || left.timeValue.Equal(right.timeValue)
		case OpLT:
			return left.timeValue.Before(right.timeValue)
		case OpLTE:
			return left.timeValue.Before(right.timeValue) || left.timeValue.Equal(right.timeValue)
		}
	}
	return false
}

func compareOrdered(op Operator, left, right float64) bool {
	switch op {
	case OpEQ:
		return left == right
	case OpNEQ:
		return left != right
	case OpGT:
		return left > right
	case OpGTE:
		return left >= right
	case OpLT:
		return left < right
	case OpLTE:
		return left <= right
	default:
		return false
	}
}
