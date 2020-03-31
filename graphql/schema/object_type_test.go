package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObjectType_IsSubTypeOf(t *testing.T) {
	iface := &InterfaceType{
		Fields: map[string]*FieldDefinition{
			"a": {
				Type: StringType,
			},
		},
	}

	obj := &ObjectType{
		Fields: map[string]*FieldDefinition{
			"a": {
				Type: StringType,
			},
		},
		ImplementedInterfaces: []*InterfaceType{iface},
	}

	union := &UnionType{
		MemberTypes: []*ObjectType{obj},
	}

	assert.True(t, obj.IsSubTypeOf(obj))
	assert.True(t, obj.IsSubTypeOf(union))
	assert.True(t, obj.IsSubTypeOf(iface))
	assert.False(t, obj.IsSubTypeOf(IntType))
}
