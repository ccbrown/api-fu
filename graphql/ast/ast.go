package ast

type Document struct {
	Definitions []interface{}
}

type OperationType string

const (
	OperationTypeQuery        OperationType = "query"
	OperationTypeMutation     OperationType = "mutation"
	OperationTypeSubscription OperationType = "subscription"
)

type OperationDefinition struct {
	OperationType       *OperationType
	Name                *Name
	VariableDefinitions *VariableDefinitions
	Directives          *Directives
	SelectionSet        *SelectionSet
}

type VariableDefinitions struct {
}

type Directives struct {
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
	Directives   *Directives
	SelectionSet *SelectionSet
}

type Argument struct {
	Name  *Name
	Value Value
}

type Name struct {
	Name string
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
