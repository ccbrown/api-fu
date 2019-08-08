package parser

import (
	"fmt"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/scanner"
	"github.com/ccbrown/api-fu/graphql/token"
)

type Location struct {
	Line   int
	Column int
}

type Error struct {
	Message  string
	Location Location
}

func (err *Error) Error() string {
	return err.Message
}

func ParseDocument(src []byte) (doc *ast.Document, errs []*Error) {
	p := newParser(src)
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(*Error); ok {
				errs = p.errors
			} else {
				panic(r)
			}
		}
	}()
	return p.parseDocument(), p.errors
}

func ParseValue(src []byte) (value ast.Value, errs []*Error) {
	p := newParser(src)
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(*Error); ok {
				errs = p.errors
			} else {
				panic(r)
			}
		}
	}()
	return p.parseValue(false), p.errors
}

type parserToken struct {
	Token    token.Token
	Value    string
	Position token.Position
}

type parser struct {
	errors        []*Error
	recursion     int
	scanner       *scanner.Scanner
	scannerErrors int
	eof           bool
	nextToken     *parserToken
}

func newParser(src []byte) *parser {
	ret := &parser{
		scanner: scanner.New(src, 0),
	}
	ret.consumeToken()
	return ret
}

const maxRecursion = 1000

func (p *parser) enter() {
	p.recursion++
	if p.recursion > maxRecursion {
		panic(p.errorf("maximum recursion depth exceeded"))
	}
}

func (p *parser) exit() {
	p.recursion--
}

func (p *parser) peek() *parserToken {
	return p.nextToken
}

func (p *parser) consumeToken() {
	if p.scanner.Scan() {
		p.nextToken = &parserToken{
			Token:    p.scanner.Token(),
			Value:    p.scanner.StringValue(),
			Position: p.scanner.Position(),
		}
	} else {
		p.eof = true
		p.nextToken = &parserToken{
			Token:    token.INVALID,
			Value:    "EOF",
			Position: p.scanner.Position(),
		}
	}
	for _, err := range p.scanner.Errors()[p.scannerErrors:] {
		p.errors = append(p.errors, &Error{
			Message: err.Message,
			Location: Location{
				Line:   err.Line,
				Column: err.Column,
			},
		})
		p.scannerErrors++
	}
}

func (p *parser) errorf(message string, args ...interface{}) *Error {
	err := &Error{
		Message: fmt.Sprintf(message, args...),
		Location: Location{
			Line:   p.peek().Position.Line,
			Column: p.peek().Position.Column,
		},
	}
	p.errors = append(p.errors, err)
	return err
}

func (p *parser) parseDocument() *ast.Document {
	p.enter()

	ret := &ast.Document{}
	for !p.eof {
		ret.Definitions = append(ret.Definitions, p.parseDefinition())
	}
	if len(ret.Definitions) == 0 {
		panic(p.errorf("expected definition"))
	}

	p.exit()
	return ret
}

func (p *parser) parseDefinition() ast.Definition {
	p.enter()

	var ret ast.Definition
	if def := p.parseOptionalFragmentDefinition(); def != nil {
		ret = def
	} else {
		ret = p.parseOperationDefinition()
	}

	p.exit()
	return ret
}

func (p *parser) parseOptionalFragmentDefinition() *ast.FragmentDefinition {
	p.enter()

	var ret *ast.FragmentDefinition
	if t := p.peek(); t.Token == token.NAME && t.Value == "fragment" {
		fragment := t.Position
		p.consumeToken()

		if t := p.peek(); t.Token != token.NAME || t.Value == "on" {
			panic(p.errorf(`expected fragment name`))
		}

		ret = &ast.FragmentDefinition{
			Fragment:      fragment,
			Name:          p.parseName(),
			TypeCondition: p.parseTypeCondition(),
			Directives:    p.parseOptionalDirectives(),
			SelectionSet:  p.parseSelectionSet(),
		}
	}

	p.exit()
	return ret
}

func (p *parser) parseOperationDefinition() *ast.OperationDefinition {
	p.enter()

	ret := &ast.OperationDefinition{}
	if ss := p.parseOptionalSelectionSet(); ss != nil {
		ret.SelectionSet = ss
	} else {
		ret.OperationType = p.parseOperationType()

		if t := p.peek(); t.Token == token.NAME {
			ret.Name = p.parseName()
		}

		ret.VariableDefinitions = p.parseOptionalVariableDefinitions()
		ret.Directives = p.parseOptionalDirectives()
		ret.SelectionSet = p.parseSelectionSet()
	}

	p.exit()
	return ret
}

func (p *parser) parseOperationType() *ast.OperationType {
	p.enter()

	var ret *ast.OperationType
	if t := p.peek(); t.Token != token.NAME || (t.Value != "query" && t.Value != "mutation" && t.Value != "subscription") {
		panic(p.errorf("expected operation type"))
	} else {
		ret = &ast.OperationType{
			Value:         t.Value,
			ValuePosition: t.Position,
		}
		p.consumeToken()
	}

	p.exit()
	return ret
}

func (p *parser) parseOptionalSelectionSet() *ast.SelectionSet {
	p.enter()

	var ret *ast.SelectionSet
	if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == "{" {
		ret = p.parseSelectionSet()
	}

	p.exit()
	return ret
}

func (p *parser) parseSelectionSet() *ast.SelectionSet {
	p.enter()

	if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != "{" {
		panic(p.errorf("expected selection set"))
	}
	ret := &ast.SelectionSet{
		Opening: p.peek().Position,
	}
	p.consumeToken()

	for {
		if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == "}" {
			if len(ret.Selections) == 0 {
				panic(p.errorf("expected selection"))
			}
			ret.Closing = t.Position
			p.consumeToken()
			break
		}
		ret.Selections = append(ret.Selections, p.parseSelection())
	}

	p.exit()
	return ret
}

func (p *parser) parseField() *ast.Field {
	p.enter()

	ret := &ast.Field{}
	ret.Name = p.parseName()
	if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == ":" {
		p.consumeToken()
		ret.Alias = ret.Name
		ret.Name = p.parseName()
	}
	ret.Arguments = p.parseOptionalArguments()
	ret.Directives = p.parseOptionalDirectives()
	ret.SelectionSet = p.parseOptionalSelectionSet()

	p.exit()
	return ret
}

func (p *parser) parseTypeCondition() *ast.NamedType {
	p.enter()

	if t := p.peek(); t.Token != token.NAME || t.Value != "on" {
		panic(p.errorf(`expected "on"`))
	}
	p.consumeToken()
	ret := p.parseNamedType()

	p.exit()
	return ret
}

func (p *parser) parseSelection() ast.Selection {
	p.enter()

	if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != "..." {
		return p.parseField()
	}
	ellipsis := p.peek().Position
	p.consumeToken()

	if t := p.peek(); t.Token == token.NAME && t.Value != "on" {
		return &ast.FragmentSpread{
			FragmentName: p.parseName(),
			Directives:   p.parseOptionalDirectives(),
			Ellipsis:     ellipsis,
		}
	}

	ret := &ast.InlineFragment{
		Ellipsis: ellipsis,
	}
	if t := p.peek(); t.Token == token.NAME {
		ret.TypeCondition = p.parseTypeCondition()
	}
	ret.Directives = p.parseOptionalDirectives()
	ret.SelectionSet = p.parseSelectionSet()

	p.exit()
	return ret
}

func (p *parser) parseOptionalArguments() []*ast.Argument {
	p.enter()

	var ret []*ast.Argument
	if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == "(" {
		p.consumeToken()

		for {
			if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == ")" {
				if len(ret) == 0 {
					panic(p.errorf("expected argument"))
				}
				p.consumeToken()
				break
			}
			ret = append(ret, p.parseArgument())
		}
	}

	p.exit()
	return ret
}

func (p *parser) parseOptionalVariableDefinitions() []*ast.VariableDefinition {
	p.enter()

	var ret []*ast.VariableDefinition
	if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == "(" {
		p.consumeToken()

		for {
			if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == ")" {
				if len(ret) == 0 {
					panic(p.errorf("expected variable definition"))
				}
				p.consumeToken()
				break
			}
			ret = append(ret, p.parseVariableDefinition())
		}
	}

	p.exit()
	return ret
}

func (p *parser) parseVariableDefinition() *ast.VariableDefinition {
	p.enter()

	variable := p.parseVariable()

	if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != ":" {
		panic(p.errorf("expected colon"))
	}
	p.consumeToken()

	typ := p.parseType()

	ret := &ast.VariableDefinition{
		Variable: variable,
		Type:     typ,
	}
	if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == "=" {
		p.consumeToken()
		ret.DefaultValue = p.parseValue(true)
	}

	p.exit()
	return ret
}

func (p *parser) parseType() ast.Type {
	p.enter()

	var ret ast.Type
	if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == "[" {
		opening := t.Position
		p.consumeToken()
		typ := p.parseType()
		if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != "]" {
			panic(p.errorf("expected ]"))
		}
		closing := p.peek().Position
		p.consumeToken()
		ret = &ast.ListType{
			Type:    typ,
			Opening: opening,
			Closing: closing,
		}
	} else {
		ret = p.parseNamedType()
	}
	if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == "!" {
		p.consumeToken()
		ret = &ast.NonNullType{
			Type: ret,
		}
	}

	p.exit()
	return ret
}

func (p *parser) parseArgument() *ast.Argument {
	p.enter()

	ret := &ast.Argument{}
	ret.Name = p.parseName()
	if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != ":" {
		panic(p.errorf("expected colon"))
	}
	p.consumeToken()
	ret.Value = p.parseValue(false)

	p.exit()
	return ret
}

func (p *parser) parseOptionalDirectives() []*ast.Directive {
	p.enter()

	var ret []*ast.Directive
	for {
		if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != "@" {
			break
		}
		at := p.peek().Position
		p.consumeToken()
		ret = append(ret, &ast.Directive{
			Name:      p.parseName(),
			Arguments: p.parseOptionalArguments(),
			At:        at,
		})
	}

	p.exit()
	return ret
}

func (p *parser) parseNamedType() *ast.NamedType {
	p.enter()

	ret := &ast.NamedType{
		Name: p.parseName(),
	}

	p.exit()
	return ret
}

func (p *parser) parseName() *ast.Name {
	p.enter()

	ret := &ast.Name{}
	if t := p.peek(); t.Token == token.NAME {
		ret.Name = t.Value
		ret.NamePosition = t.Position
		p.consumeToken()
	} else {
		panic(p.errorf("expected name"))
	}

	p.exit()
	return ret
}

func (p *parser) parseVariable() *ast.Variable {
	p.enter()

	if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != "$" {
		panic(p.errorf("expected variable"))
	}
	dollar := p.peek().Position
	p.consumeToken()
	ret := &ast.Variable{
		Name:   p.parseName(),
		Dollar: dollar,
	}

	p.exit()
	return ret
}

func (p *parser) parseValue(constant bool) ast.Value {
	p.enter()

	var ret ast.Value

	switch t := p.peek(); t.Token {
	case token.INT_VALUE:
		p.consumeToken()
		ret = &ast.IntValue{
			Value:   t.Value,
			Literal: t.Position,
		}
	case token.FLOAT_VALUE:
		p.consumeToken()
		ret = &ast.FloatValue{
			Value:   t.Value,
			Literal: t.Position,
		}
	case token.STRING_VALUE:
		p.consumeToken()
		ret = &ast.StringValue{
			Value:   t.Value,
			Literal: t.Position,
		}
	case token.NAME:
		p.consumeToken()
		switch v := t.Value; v {
		case "true", "false":
			ret = &ast.BooleanValue{
				Value:   v == "true",
				Literal: t.Position,
			}
		case "null":
			ret = &ast.NullValue{
				Literal: t.Position,
			}
		default:
			ret = &ast.EnumValue{
				Value:   v,
				Literal: t.Position,
			}
		}
	case token.PUNCTUATOR:
		switch v := t.Value; v {
		case "$":
			if constant {
				panic(p.errorf("expected constant value"))
			}
			ret = p.parseVariable()
		case "[":
			opening := t.Position
			p.consumeToken()
			var values []ast.Value
			for {
				if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == "]" {
					p.consumeToken()
					ret = &ast.ListValue{
						Values:  values,
						Opening: opening,
						Closing: t.Position,
					}
					break
				}
				values = append(values, p.parseValue(constant))
			}
		case "{":
			opening := t.Position
			p.consumeToken()
			var fields []*ast.ObjectField
			for {
				t := p.peek()
				if t.Token == token.PUNCTUATOR && t.Value == "}" {
					p.consumeToken()
					ret = &ast.ObjectValue{
						Fields:  fields,
						Opening: opening,
						Closing: t.Position,
					}
					break
				}
				name := p.parseName()
				if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != ":" {
					panic(p.errorf("expected colon"))
				}
				p.consumeToken()
				value := p.parseValue(constant)
				fields = append(fields, &ast.ObjectField{
					Name:  name,
					Value: value,
				})
			}
		}
	}

	if ret == nil {
		panic(p.errorf("expected value"))
	}

	p.exit()
	return ret
}
