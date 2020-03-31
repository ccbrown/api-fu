package apifu

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ccbrown/api-fu/graphql/ast"
)

func TestDateTimeType(t *testing.T) {
	assert.Equal(t, time.Date(2019, time.December, 1, 1, 23, 45, 600000000, time.UTC), DateTimeType.LiteralCoercion(&ast.StringValue{
		Value: "2019-12-01T01:23:45.6Z",
	}))
}

func TestLongIntType(t *testing.T) {
	assert.Equal(t, int64(9007199254740991), LongIntType.LiteralCoercion(&ast.IntValue{
		Value: "9007199254740991",
	}))

	assert.Nil(t, LongIntType.LiteralCoercion(&ast.IntValue{
		Value: "9007199254740992",
	}))

	assert.Equal(t, int64(-9007199254740991), LongIntType.LiteralCoercion(&ast.IntValue{
		Value: "-9007199254740991",
	}))
}

func TestCoerceLongInt(t *testing.T) {
	for _, tc := range []struct {
		Value    interface{}
		Expected int64
	}{
		{Value: true, Expected: 1},
		{Value: false, Expected: 0},
		{Value: int8(1), Expected: 1},
		{Value: uint8(1), Expected: 1},
		{Value: int16(1), Expected: 1},
		{Value: uint16(1), Expected: 1},
		{Value: int32(1), Expected: 1},
		{Value: uint32(1), Expected: 1},
		{Value: int64(1), Expected: 1},
		{Value: uint64(1), Expected: 1},
		{Value: int(1), Expected: 1},
		{Value: uint(1), Expected: 1},
		{Value: float32(1.0), Expected: 1},
		{Value: float64(1.0), Expected: 1},
	} {
		assert.Equal(t, tc.Expected, coerceLongInt(tc.Value))
	}

	assert.Nil(t, coerceLongInt("foo"))
}
