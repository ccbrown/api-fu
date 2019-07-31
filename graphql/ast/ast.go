package ast

type Document struct {
	Definitions []Definition
}

// OperationDefinition or FragmentDefinition
type Definition interface{}

type OperationType string

const (
	OperationTypeQuery        OperationType = "query"
	OperationTypeMutation     OperationType = "mutation"
	OperationTypeSubscription OperationType = "subscription"
)

func (t OperationType) IsValid() bool {
	switch t {
	case OperationTypeQuery, OperationTypeMutation, OperationTypeSubscription:
		return true
	default:
		return false
	}
}

type OperationDefinition struct {
	OperationType       *OperationType
	Name                *Name
	VariableDefinitions []*VariableDefinition
	Directives          []*Directive
	SelectionSet        *SelectionSet
}

type FragmentDefinition struct {
	Name          *Name
	TypeCondition *NamedType
	Directives    []*Directive
	SelectionSet  *SelectionSet
}

type VariableDefinition struct {
	Variable     *Variable
	Type         Type
	DefaultValue Value
}

// NamedType, ListType, or NonNullType
type Type interface{}

type ListType struct {
	Type Type
}

type NonNullType struct {
	Type Type
}

type Directive struct {
	Name      *Name
	Arguments []*Argument
}

type SelectionSet struct {
	Selections []Selection
}

// Field, FragmentSpread, or InlineFragment
type Selection interface{}

type Field struct {
	Alias        *Name
	Name         *Name
	Arguments    []*Argument
	Directives   []*Directive
	SelectionSet *SelectionSet
}

type FragmentSpread struct {
	FragmentName *Name
	Directives   []*Directive
}

type InlineFragment struct {
	TypeCondition *NamedType
	Directives    []*Directive
	SelectionSet  *SelectionSet
}

type Argument struct {
	Name  *Name
	Value Value
}

type Name struct {
	Name string
}

type NamedType struct {
	Name *Name
}

// Variable, IntValue, FloatValue, StringValue, BooleanValue, NullValue, EnumValue, ListValue, or
// ObjectValue
type Value interface{}

type Variable struct {
	Name *Name
}

type BooleanValue struct {
	Value bool
}

type FloatValue struct {
	Value string
}

type IntValue struct {
	Value string
}

type StringValue struct {
	Value string
}

type EnumValue struct {
	Value string
}

type NullValue struct{}

type ListValue struct {
	Values []Value
}

type ObjectValue struct {
	Fields []*ObjectField
}

type ObjectField struct {
	Name  *Name
	Value Value
}
