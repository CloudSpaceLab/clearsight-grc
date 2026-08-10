package assurance

import (
	"fmt"
	"strings"
)

type PostgresPredicate struct {
	MatchSQL   string `json:"match_sql"`
	UnknownSQL string `json:"unknown_sql"`
	Args       []any  `json:"-"`
}

// PostgresPredicate returns source-side predicates for the same tri-state
// condition. MatchSQL selects demonstrable matches; UnknownSQL selects rows that
// cannot be safely evaluated because a required field is null. Schema/type
// mismatch is handled before execution by schema fingerprint/condition validation.
func (c *CompiledCondition) PostgresPredicate() (PostgresPredicate, error) {
	if c == nil {
		return PostgresPredicate{}, fmt.Errorf("compiled condition is required")
	}
	builder := &postgresBuilder{}
	expression, err := builder.compile(c.condition)
	if err != nil {
		return PostgresPredicate{}, err
	}
	return PostgresPredicate{MatchSQL: expression.match, UnknownSQL: expression.unknown, Args: append([]any(nil), builder.args...)}, nil
}

type postgresExpression struct {
	match   string
	unknown string
}

type postgresBuilder struct {
	args []any
}

func (b *postgresBuilder) compile(condition Condition) (postgresExpression, error) {
	switch condition.Op {
	case OpAnd:
		children := make([]postgresExpression, 0, len(condition.Children))
		for _, child := range condition.Children {
			compiled, err := b.compile(child)
			if err != nil {
				return postgresExpression{}, err
			}
			children = append(children, compiled)
		}
		matchParts := make([]string, len(children))
		nonClearParts := make([]string, len(children))
		unknownParts := make([]string, len(children))
		for index, child := range children {
			matchParts[index] = child.match
			nonClearParts[index] = "(" + child.match + " OR " + child.unknown + ")"
			unknownParts[index] = child.unknown
		}
		return postgresExpression{
			match:   "(" + strings.Join(matchParts, " AND ") + ")",
			unknown: "((" + strings.Join(nonClearParts, " AND ") + ") AND (" + strings.Join(unknownParts, " OR ") + "))",
		}, nil
	case OpOr:
		children := make([]postgresExpression, 0, len(condition.Children))
		for _, child := range condition.Children {
			compiled, err := b.compile(child)
			if err != nil {
				return postgresExpression{}, err
			}
			children = append(children, compiled)
		}
		matchParts := make([]string, len(children))
		unknownParts := make([]string, len(children))
		for index, child := range children {
			matchParts[index] = child.match
			unknownParts[index] = child.unknown
		}
		match := "(" + strings.Join(matchParts, " OR ") + ")"
		return postgresExpression{match: match, unknown: "((NOT " + match + ") AND (" + strings.Join(unknownParts, " OR ") + "))"}, nil
	case OpNot:
		child, err := b.compile(condition.Children[0])
		if err != nil {
			return postgresExpression{}, err
		}
		return postgresExpression{match: "(NOT (" + child.match + " OR " + child.unknown + "))", unknown: child.unknown}, nil
	}

	field := quotePostgresIdentifier(condition.Field)
	switch condition.Op {
	case OpExists:
		return postgresExpression{match: "(" + field + " IS NOT NULL)", unknown: "FALSE"}, nil
	case OpMissing:
		return postgresExpression{match: "(" + field + " IS NULL)", unknown: "FALSE"}, nil
	case OpIn, OpNotIn:
		placeholders := make([]string, 0, len(condition.Values))
		for _, value := range condition.Values {
			placeholders = append(placeholders, b.add(value))
		}
		operator := "IN"
		if condition.Op == OpNotIn {
			operator = "NOT IN"
		}
		return postgresExpression{
			match:   fmt.Sprintf("(%s IS NOT NULL AND %s %s (%s))", field, field, operator, strings.Join(placeholders, ", ")),
			unknown: "(" + field + " IS NULL)",
		}, nil
	case OpBetween:
		lower := b.add(condition.Values[0])
		upper := b.add(condition.Values[1])
		return postgresExpression{
			match:   fmt.Sprintf("(%s IS NOT NULL AND %s >= %s AND %s <= %s)", field, field, lower, field, upper),
			unknown: "(" + field + " IS NULL)",
		}, nil
	case OpContains:
		value := b.add(condition.Value)
		return postgresExpression{
			match:   fmt.Sprintf("(%s IS NOT NULL AND POSITION(%s IN %s) > 0)", field, value, field),
			unknown: "(" + field + " IS NULL)",
		}, nil
	case OpEQ, OpNEQ, OpGT, OpGTE, OpLT, OpLTE:
		operator := postgresOperator(condition.Op)
		if operator == "" {
			return postgresExpression{}, fmt.Errorf("operator %s is not supported by PostgreSQL compiler", condition.Op)
		}
		if condition.OtherField != "" {
			right := quotePostgresIdentifier(condition.OtherField)
			return postgresExpression{
				match:   fmt.Sprintf("(%s IS NOT NULL AND %s IS NOT NULL AND %s %s %s)", field, right, field, operator, right),
				unknown: fmt.Sprintf("(%s IS NULL OR %s IS NULL)", field, right),
			}, nil
		}
		value := b.add(condition.Value)
		return postgresExpression{
			match:   fmt.Sprintf("(%s IS NOT NULL AND %s %s %s)", field, field, operator, value),
			unknown: "(" + field + " IS NULL)",
		}, nil
	default:
		return postgresExpression{}, fmt.Errorf("operator %s is not supported by PostgreSQL compiler", condition.Op)
	}
}

func (b *postgresBuilder) add(value Literal) string {
	var argument any
	switch value.Type {
	case TypeString:
		argument = value.String
	case TypeNumber:
		argument = value.Number
	case TypeBool:
		argument = value.Bool
	case TypeTime:
		argument = value.Time.UTC()
	default:
		argument = nil
	}
	b.args = append(b.args, argument)
	return fmt.Sprintf("$%d", len(b.args))
}

func postgresOperator(value Operator) string {
	switch value {
	case OpEQ:
		return "="
	case OpNEQ:
		return "<>"
	case OpGT:
		return ">"
	case OpGTE:
		return ">="
	case OpLT:
		return "<"
	case OpLTE:
		return "<="
	default:
		return ""
	}
}

func quotePostgresIdentifier(value string) string {
	return "\"" + strings.ReplaceAll(value, "\"", "\"\"") + "\""
}
