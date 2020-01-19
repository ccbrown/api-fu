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
