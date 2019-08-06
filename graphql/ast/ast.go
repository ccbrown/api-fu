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
type Selection interface {
	SelectionDirectives() []*Directive
}

type Field struct {
	Alias        *Name
	Name         *Name
	Arguments    []*Argument
	Directives   []*Directive
	SelectionSet *SelectionSet
}

func (s *Field) SelectionDirectives() []*Directive {
	return s.Directives
}

type FragmentSpread struct {
	FragmentName *Name
	Directives   []*Directive
}

func (s *FragmentSpread) SelectionDirectives() []*Directive {
	return s.Directives
}

type InlineFragment struct {
	TypeCondition *NamedType
	Directives    []*Directive
	SelectionSet  *SelectionSet
}

func (s *InlineFragment) SelectionDirectives() []*Directive {
	return s.Directives
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
type Value interface {
	IsValue() bool
}

type Variable struct {
	Name *Name
}

func (*Variable) IsValue() bool { return true }

type BooleanValue struct {
	Value bool
}

func (*BooleanValue) IsValue() bool { return true }

type FloatValue struct {
	Value string
}

func (*FloatValue) IsValue() bool { return true }

type IntValue struct {
	Value string
}

func (*IntValue) IsValue() bool { return true }

type StringValue struct {
	// Value is the actual, unquoted value.
	Value string
}

func (*StringValue) IsValue() bool { return true }

type EnumValue struct {
	Value string
}

func (*EnumValue) IsValue() bool { return true }

type NullValue struct{}

func (*NullValue) IsValue() bool { return true }

func IsNullValue(v Value) bool {
	_, ok := v.(*NullValue)
	return ok
}

type ListValue struct {
	Values []Value
}

func (*ListValue) IsValue() bool { return true }

type ObjectValue struct {
	Fields []*ObjectField
}

func (*ObjectValue) IsValue() bool { return true }

type ObjectField struct {
	Name  *Name
	Value Value
}
