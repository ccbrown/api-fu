package schema

import (
	"testing"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/stretchr/testify/assert"
)

func TestCoerceInt(t *testing.T) {
	for _, tc := range []struct {
		Value    interface{}
		Expected int
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
		assert.Equal(t, tc.Expected, coerceInt(tc.Value))
	}

	assert.Nil(t, coerceInt("foo"))
}

func TestCoerceFloat(t *testing.T) {
	for _, tc := range []struct {
		Value    interface{}
		Expected float64
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
		assert.Equal(t, tc.Expected, coerceFloat(tc.Value))
	}

	assert.Nil(t, coerceFloat("foo"))
}

func TestFloatType(t *testing.T) {
	assert.Equal(t, 1.0, FloatType.LiteralCoercion(&ast.IntValue{
		Value: "1",
	}))

	assert.Equal(t, 1.0, FloatType.LiteralCoercion(&ast.FloatValue{
		Value: "1.0",
	}))
}

func TestIDType(t *testing.T) {
	assert.Equal(t, 1, IDType.LiteralCoercion(&ast.IntValue{
		Value: "1",
	}))

	assert.Equal(t, "1", IDType.LiteralCoercion(&ast.StringValue{
		Value: "1",
	}))

	for _, tc := range []struct {
		Value    interface{}
		Expected interface{}
	}{
		{Value: 1, Expected: 1},
		{Value: 1.0, Expected: 1},
		{Value: "1", Expected: "1"},
	} {
		assert.Equal(t, tc.Expected, IDType.VariableValueCoercion(tc.Value))
	}

	assert.Nil(t, IDType.VariableValueCoercion([]int{}))

	for _, tc := range []struct {
		Value    interface{}
		Expected string
	}{
		{Value: int8(1), Expected: "1"},
		{Value: uint8(1), Expected: "1"},
		{Value: int16(1), Expected: "1"},
		{Value: uint16(1), Expected: "1"},
		{Value: int32(1), Expected: "1"},
		{Value: uint32(1), Expected: "1"},
		{Value: int64(1), Expected: "1"},
		{Value: uint64(1), Expected: "1"},
		{Value: int(1), Expected: "1"},
		{Value: uint(1), Expected: "1"},
		{Value: "1", Expected: "1"},
	} {
		assert.Equal(t, tc.Expected, IDType.ResultCoercion(tc.Value))
	}

	assert.Nil(t, IDType.ResultCoercion([]int{}))
}
