package ast

import "github.com/ccbrown/api-fu/graphql/token"

type Node interface {
	Position() token.Position
}

type Document struct {
	Definitions []Definition
}

func (*Document) Position() token.Position { return token.Position{1, 1} }

// OperationDefinition or FragmentDefinition
type Definition interface {
	Node
}

type OperationDefinition struct {
	OperationType       *OperationType
	Name                *Name
	VariableDefinitions []*VariableDefinition
	Directives          []*Directive
	SelectionSet        *SelectionSet
}

func (n *OperationDefinition) Position() token.Position {
	if n.OperationType != nil {
		return n.OperationType.Position()
	}
	return n.SelectionSet.Position()
}

type OperationType struct {
	Value         string
	ValuePosition token.Position
}

func (n *OperationType) Position() token.Position { return n.ValuePosition }

type FragmentDefinition struct {
	Fragment      token.Position
	Name          *Name
	TypeCondition *NamedType
	Directives    []*Directive
	SelectionSet  *SelectionSet
}

func (n *FragmentDefinition) Position() token.Position { return n.Fragment }

type VariableDefinition struct {
	Variable     *Variable
	Type         Type
	DefaultValue Value
}

func (n *VariableDefinition) Position() token.Position { return n.Variable.Position() }

// NamedType, ListType, or NonNullType
type Type interface {
	Node
}

type ListType struct {
	Type    Type
	Opening token.Position
	Closing token.Position
}

func (n *ListType) Position() token.Position { return n.Opening }

type NonNullType struct {
	Type Type
}

func (n *NonNullType) Position() token.Position { return n.Type.Position() }

type Directive struct {
	Name      *Name
	Arguments []*Argument
	At        token.Position
}

func (n *Directive) Position() token.Position { return n.At }

type SelectionSet struct {
	Selections []Selection
	Opening    token.Position
	Closing    token.Position
}

func (n *SelectionSet) Position() token.Position { return n.Opening }

// Field, FragmentSpread, or InlineFragment
type Selection interface {
	Node
	SelectionDirectives() []*Directive
}

type Field struct {
	Alias        *Name
	Name         *Name
	Arguments    []*Argument
	Directives   []*Directive
	SelectionSet *SelectionSet
}

func (n *Field) Position() token.Position {
	if n.Alias != nil {
		return n.Alias.Position()
	}
	return n.Name.Position()
}

func (s *Field) SelectionDirectives() []*Directive { return s.Directives }

type FragmentSpread struct {
	FragmentName *Name
	Directives   []*Directive
	Ellipsis     token.Position
}

func (n *FragmentSpread) Position() token.Position          { return n.Ellipsis }
func (s *FragmentSpread) SelectionDirectives() []*Directive { return s.Directives }

type InlineFragment struct {
	TypeCondition *NamedType
	Directives    []*Directive
	SelectionSet  *SelectionSet
	Ellipsis      token.Position
}

func (n *InlineFragment) Position() token.Position          { return n.Ellipsis }
func (s *InlineFragment) SelectionDirectives() []*Directive { return s.Directives }

type Argument struct {
	Name  *Name
	Value Value
}

func (n *Argument) Position() token.Position { return n.Name.Position() }

type Name struct {
	Name         string
	NamePosition token.Position
}

func (n *Name) Position() token.Position { return n.NamePosition }

type NamedType struct {
	Name *Name
}

func (n *NamedType) Position() token.Position { return n.Name.Position() }

// Variable, IntValue, FloatValue, StringValue, BooleanValue, NullValue, EnumValue, ListValue, or
// ObjectValue
type Value interface {
	Node
	IsValue() bool
}

type Variable struct {
	Name   *Name
	Dollar token.Position
}

func (*Variable) IsValue() bool              { return true }
func (n *Variable) Position() token.Position { return n.Dollar }

type BooleanValue struct {
	Value   bool
	Literal token.Position
}

func (*BooleanValue) IsValue() bool              { return true }
func (n *BooleanValue) Position() token.Position { return n.Literal }

type FloatValue struct {
	Value   string
	Literal token.Position
}

func (*FloatValue) IsValue() bool              { return true }
func (n *FloatValue) Position() token.Position { return n.Literal }

type IntValue struct {
	Value   string
	Literal token.Position
}

func (*IntValue) IsValue() bool              { return true }
func (n *IntValue) Position() token.Position { return n.Literal }

type StringValue struct {
	// Value is the actual, unquoted value.
	Value string

	Literal token.Position
}

func (*StringValue) IsValue() bool              { return true }
func (n *StringValue) Position() token.Position { return n.Literal }

type EnumValue struct {
	Value   string
	Literal token.Position
}

func (*EnumValue) IsValue() bool              { return true }
func (n *EnumValue) Position() token.Position { return n.Literal }

type NullValue struct {
	Literal token.Position
}

func (*NullValue) IsValue() bool              { return true }
func (n *NullValue) Position() token.Position { return n.Literal }

func IsNullValue(v Value) bool {
	_, ok := v.(*NullValue)
	return ok
}

type ListValue struct {
	Values  []Value
	Opening token.Position
	Closing token.Position
}

func (*ListValue) IsValue() bool              { return true }
func (n *ListValue) Position() token.Position { return n.Opening }

type ObjectValue struct {
	Fields  []*ObjectField
	Opening token.Position
	Closing token.Position
}

func (*ObjectValue) IsValue() bool              { return true }
func (n *ObjectValue) Position() token.Position { return n.Opening }

type ObjectField struct {
	Name  *Name
	Value Value
}

func (n *ObjectField) Position() token.Position { return n.Name.Position() }
