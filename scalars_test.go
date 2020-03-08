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
	assert.Equal(t, 9007199254740991, LongIntType.LiteralCoercion(&ast.IntValue{
		Value: "9007199254740991",
	}))

	assert.Nil(t, LongIntType.LiteralCoercion(&ast.IntValue{
		Value: "9007199254740992",
	}))

	assert.Equal(t, -9007199254740991, LongIntType.LiteralCoercion(&ast.IntValue{
		Value: "-9007199254740991",
	}))
}
