package apifu

import (
	"math"
	"strconv"
	"time"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/ccbrown/api-fu/graphql/ast"
)

func parseDateTime(v interface{}) interface{} {
	switch v := v.(type) {
	case []byte:
		t := time.Time{}
		if err := t.UnmarshalText(v); err == nil {
			return t
		}
		return nil
	case string:
		return parseDateTime([]byte(v))
	}
	return nil
}

// DateTimeType provides a DateTime implementation that serializing to and from RFC-3339 datetimes.
var DateTimeType = &graphql.ScalarType{
	Name:        "DateTime",
	Description: "DateTime represents an RFC-3339 datetime.",
	LiteralCoercion: func(v ast.Value) interface{} {
		switch v := v.(type) {
		case *ast.StringValue:
			return parseDateTime(v.Value)
		}
		return nil
	},
	VariableValueCoercion: parseDateTime,
	ResultCoercion: func(v interface{}) interface{} {
		switch v := v.(type) {
		case time.Time:
			if b, err := v.MarshalText(); err == nil {
				return string(b)
			}
		}
		return nil
	},
}

// NonZeroDateTime returns a field definition that resolves to the value of the field with the given
// name. If the field's value is the zero time, the field resolves to nil instead.
func NonZeroDateTime(fieldName string) *graphql.FieldDefinition {
	return &graphql.FieldDefinition{
		Type: DateTimeType,
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			if t := fieldValue(ctx.Object, fieldName).(time.Time); !t.IsZero() {
				return t, nil
			}
			return nil, nil
		},
	}
}

const (
	maxSafeInteger = 9007199254740991
	minSafeInteger = -9007199254740991
)

func coerceLongInt(v interface{}) interface{} {
	switch v := v.(type) {
	case bool:
		if v {
			return int64(1)
		}
		return int64(0)
	case int8:
		return int64(v)
	case uint8:
		return int64(v)
	case int16:
		return int64(v)
	case uint16:
		return int64(v)
	case int32:
		return int64(v)
	case uint32:
		return int64(v)
	case int64:
		if v >= minSafeInteger && v <= maxSafeInteger {
			return int64(v)
		}
	case uint64:
		if v <= maxSafeInteger {
			return int64(v)
		}
	case int:
		if v >= minSafeInteger && v <= maxSafeInteger {
			return int64(v)
		}
	case uint:
		if v <= maxSafeInteger {
			return int64(v)
		}
	case float32:
		return coerceLongInt(float64(v))
	case float64:
		if n := math.Trunc(v); n == v && n >= minSafeInteger && n <= maxSafeInteger {
			return int64(n)
		}
	}
	return nil
}

// LongIntType provides a scalar implementation for integers that may be larger than 32 bits, but
// can still be represented by JavaScript numbers.
var LongIntType = &graphql.ScalarType{
	Name:        "LongInt",
	Description: "LongInt represents a signed integer that may be longer than 32 bits, but still within JavaScript / IEEE-654's \"safe\" range.",
	LiteralCoercion: func(v ast.Value) interface{} {
		switch v := v.(type) {
		case *ast.IntValue:
			if n, err := strconv.ParseInt(v.Value, 10, 64); err == nil && n >= minSafeInteger && n <= maxSafeInteger {
				return n
			}
		}
		return nil
	},
	VariableValueCoercion: coerceLongInt,
	ResultCoercion:        coerceLongInt,
}
