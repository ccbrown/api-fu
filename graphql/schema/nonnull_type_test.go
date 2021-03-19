package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNonNullType_IsSubTypeOf(t *testing.T) {
	iface := &InterfaceType{}
	obj := &ObjectType{
		ImplementedInterfaces: []*InterfaceType{iface},
	}
	assert.True(t, NewNonNullType(obj).IsSubTypeOf(NewNonNullType(iface)))
}
