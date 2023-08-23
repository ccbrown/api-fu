package ast

import "github.com/ccbrown/api-fu/sdl/token"

type Node interface {
	Position() token.Position
}

type Document struct {
	Definitions []Definition
}

func (*Document) Position() token.Position { return token.Position{1, 1} }

// InterfaceDefinition or ResourceDefinition
type Definition interface {
	Node
}

type Name struct {
	Name         string
	NamePosition token.Position
}

func (n *Name) Position() token.Position { return n.NamePosition }

type InterfaceDefinition struct {
	Name    *Name
	Extends []*Name

	Attributes    *Attributes
	Relationships *Relationships
}

func (n *InterfaceDefinition) Position() token.Position { return n.Name.Position() }

type ResourceDefinition struct {
	Name    *Name
	Extends []*Name

	Type          *StringValue
	Attributes    *Attributes
	Relationships *Relationships
}

func (n *ResourceDefinition) Position() token.Position { return n.Name.Position() }

type Attributes struct {
	Opening token.Position
	Closing token.Position
	Fields  []*Field
}

func (n *Attributes) Position() token.Position { return n.Opening }

type Relationships struct {
	Opening token.Position
	Closing token.Position
	Fields  []*Field
}

func (n *Relationships) Position() token.Position { return n.Opening }

type StringValue struct {
	// Value is the actual, unquoted value.
	Value string

	Literal token.Position
}

func (n *StringValue) Position() token.Position { return n.Literal }

type Field struct {
	Name *Name
	Type Type
}

func (n *Field) Position() token.Position { return n.Name.Position() }

// NamedType or RequiredType
type Type interface {
	Node
}

type RequiredType struct {
	Type Type
}

func (n *RequiredType) Position() token.Position { return n.Type.Position() }

type NamedType struct {
	Name *Name
}

func (n *NamedType) Position() token.Position { return n.Name.Position() }
