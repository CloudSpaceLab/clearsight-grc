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

// PostgresPredicate returns source-side predicates for the same bounded
// tri-state condition. MatchSQL selects demonstrable matches; UnknownSQL selects
// rows that cannot be safely evaluated inside the T0 logical domain.
func (c *CompiledCondition) PostgresPredicate() (PostgresPredicate, error) {
	if c == nil {
		return PostgresPredicate{}, fmt.Errorf("compiled condition is required")
	}
	if err := validatePostgresCompatibility(c.condition); err != nil {
		return PostgresPredicate{}, err
	}
	builder := &postgresBuilder{fields: c.fields}
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

type postgresFieldExpression struct {
	raw        string
	comparable string
	valid      string
	unknown    string
}

type postgresBuilder struct {
	fields map[string]Field
	args   []any
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

	field, err := b.field(condition.Field)
	if err != nil {
		return postgresExpression{}, err
	}
	switch condition.Op {
	case OpExists:
		return postgresExpression{match: "(" + field.raw + " IS NOT NULL)", unknown: "FALSE"}, nil
	case OpMissing:
		return postgresExpression{match: "(" + field.raw + " IS NULL)", unknown: "FALSE"}, nil
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
			match:   fmt.Sprintf("(%s AND %s %s (%s))", field.valid, field.comparable, operator, strings.Join(placeholders, ", ")),
			unknown: field.unknown,
		}, nil
	case OpBetween:
		lower := b.add(condition.Values[0])
		upper := b.add(condition.Values[1])
		return postgresExpression{
			match:   fmt.Sprintf("(%s AND %s >= %s AND %s <= %s)", field.valid, field.comparable, lower, field.comparable, upper),
			unknown: field.unknown,
		}, nil
	case OpContains:
		value := b.add(condition.Value)
		return postgresExpression{
			match:   fmt.Sprintf("(%s AND strpos(%s::text, %s) > 0)", field.valid, field.raw, value),
			unknown: field.unknown,
		}, nil
	case OpEQ, OpNEQ, OpGT, OpGTE, OpLT, OpLTE:
		operator := postgresOperator(condition.Op)
		if operator == "" {
			return postgresExpression{}, fmt.Errorf("operator %s is not supported by PostgreSQL compiler", condition.Op)
		}
		if condition.OtherField != "" {
			right, err := b.field(condition.OtherField)
			if err != nil {
				return postgresExpression{}, err
			}
			return postgresExpression{
				match:   fmt.Sprintf("(%s AND %s AND %s %s %s)", field.valid, right.valid, field.comparable, operator, right.comparable),
				unknown: fmt.Sprintf("(%s OR %s)", field.unknown, right.unknown),
			}, nil
		}
		value := b.add(condition.Value)
		return postgresExpression{
			match:   fmt.Sprintf("(%s AND %s %s %s)", field.valid, field.comparable, operator, value),
			unknown: field.unknown,
		}, nil
	default:
		return postgresExpression{}, fmt.Errorf("operator %s is not supported by PostgreSQL compiler", condition.Op)
	}
}

func (b *postgresBuilder) field(name string) (postgresFieldExpression, error) {
	field, ok := b.fields[name]
	if !ok {
		return postgresFieldExpression{}, fmt.Errorf("PostgreSQL predicate references unknown field %q", name)
	}
	raw := quotePostgresIdentifier(name)
	result := postgresFieldExpression{raw: raw}
	switch field.Type {
	case TypeString:
		result.comparable = "(" + raw + "::text COLLATE \"C\")"
		result.valid = fmt.Sprintf("(%s IS NOT NULL AND octet_length(%s::text) <= %d)", raw, raw, hardMaxEvaluatedStringBytes)
	case TypeNumber:
		result.comparable = "(" + raw + "::double precision)"
		result.valid = fmt.Sprintf("(%s IS NOT NULL AND %s >= -%d AND %s <= %d)", raw, raw, maxExactFloatInteger, raw, maxExactFloatInteger)
	case TypeBool, TypeTime:
		result.comparable = raw
		result.valid = "(" + raw + " IS NOT NULL)"
	case TypeUnknown:
		result.comparable = raw
		result.valid = "(" + raw + " IS NOT NULL)"
	default:
		return postgresFieldExpression{}, fmt.Errorf("field %q has unsupported logical type %q", name, field.Type)
	}
	result.unknown = "(NOT " + result.valid + ")"
	return result, nil
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

func validatePostgresCompatibility(condition Condition) error {
	if condition.Value.Type == TypeTime && condition.Value.Time.Nanosecond()%1000 != 0 {
		return fmt.Errorf("PostgreSQL pushdown requires TIME literals at microsecond precision")
	}
	for _, value := range condition.Values {
		if value.Type == TypeTime && value.Time.Nanosecond()%1000 != 0 {
			return fmt.Errorf("PostgreSQL pushdown requires TIME literals at microsecond precision")
		}
	}
	for _, child := range condition.Children {
		if err := validatePostgresCompatibility(child); err != nil {
			return err
		}
	}
	return nil
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
