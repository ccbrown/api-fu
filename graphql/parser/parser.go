package parser

import (
	"fmt"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/scanner"
	"github.com/ccbrown/api-fu/graphql/token"
)

type Error struct {
	message string
}

func (err *Error) Error() string {
	return err.message
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
	Token token.Token
	Value string
}

var eof = &parserToken{}

type parser struct {
	errors    []*Error
	tokens    []*parserToken
	recursion int
}

func newParser(src []byte) *parser {
	var tokens []*parserToken
	s := scanner.New(src, 0)
	for s.Scan() {
		tokens = append(tokens, &parserToken{
			Token: s.Token(),
			Value: s.StringValue(),
		})
	}
	ret := &parser{
		errors: make([]*Error, len(s.Errors())),
		tokens: tokens,
	}
	for i, err := range s.Errors() {
		ret.errors[i] = &Error{
			message: err.Error(),
		}
	}
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
	if len(p.tokens) > 0 {
		return p.tokens[0]
	}
	return eof
}

func (p *parser) consumeToken() {
	if len(p.tokens) > 0 {
		p.tokens = p.tokens[1:]
	}
}

func (p *parser) errorf(message string, args ...interface{}) *Error {
	err := &Error{
		message: fmt.Sprintf(message, args...),
	}
	p.errors = append(p.errors, err)
	return err
}

func (p *parser) parseDocument() *ast.Document {
	p.enter()

	ret := &ast.Document{}
	for p.peek() != eof {
		ret.Definitions = append(ret.Definitions, p.parseDefinition())
	}

	p.exit()
	return ret
}

func (p *parser) parseDefinition() ast.Definition {
	p.enter()

	var ret ast.Definition
	if t := p.peek(); t.Token == token.NAME && t.Value == "fragment" {
		ret = p.parseFragmentDefinition()
	} else {
		ret = p.parseOperationDefinition()
	}

	p.exit()
	return ret
}

func (p *parser) parseFragmentDefinition() *ast.FragmentDefinition {
	p.enter()

	if t := p.peek(); t.Token != token.NAME || t.Value != "fragment" {
		panic(p.errorf(`expected "fragment"`))
	}
	p.consumeToken()

	ret := &ast.FragmentDefinition{
		Name:          p.parseName(),
		TypeCondition: p.parseTypeCondition(),
		Directives:    p.parseOptionalDirectives(),
		SelectionSet:  p.parseSelectionSet(),
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
		if t := p.peek(); t.Token != token.NAME || !ast.OperationType(t.Value).IsValid() {
			panic(p.errorf("expected operation type"))
		} else {
			ot := ast.OperationType(t.Value)
			ret.OperationType = &ot
			p.consumeToken()
		}

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
	p.consumeToken()

	ret := &ast.SelectionSet{}
	for {
		if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == "}" {
			if len(ret.Selections) == 0 {
				panic(p.errorf("expected selection"))
			}
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
	p.consumeToken()

	if t := p.peek(); t.Token == token.NAME && t.Value != "on" {
		return &ast.FragmentSpread{
			FragmentName: p.parseName(),
			Directives:   p.parseOptionalDirectives(),
		}
	}

	ret := &ast.InlineFragment{}
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
					panic(p.errorf("variable definition"))
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
		p.consumeToken()
		typ := p.parseType()
		if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != "]" {
			panic(p.errorf("expected ]"))
		}
		p.consumeToken()
		ret = &ast.ListType{
			Type: typ,
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
		p.consumeToken()
		ret = append(ret, &ast.Directive{
			Name:      p.parseName(),
			Arguments: p.parseOptionalArguments(),
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
	p.consumeToken()
	ret := &ast.Variable{
		Name: p.parseName(),
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
			Value: t.Value,
		}
	case token.FLOAT_VALUE:
		p.consumeToken()
		ret = &ast.FloatValue{
			Value: t.Value,
		}
	case token.STRING_VALUE:
		p.consumeToken()
		ret = &ast.StringValue{
			Value: t.Value,
		}
	case token.NAME:
		p.consumeToken()
		switch v := t.Value; v {
		case "true", "false":
			ret = &ast.BooleanValue{
				Value: v == "true",
			}
		case "null":
			ret = &ast.NullValue{}
		default:
			ret = &ast.EnumValue{
				Value: v,
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
			p.consumeToken()
			var values []ast.Value
			for {
				if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == "]" {
					p.consumeToken()
					break
				}
				values = append(values, p.parseValue(constant))
			}
			ret = &ast.ListValue{
				Values: values,
			}
		case "{":
			p.consumeToken()
			var fields []*ast.ObjectField
			for {
				t := p.peek()
				if t.Token == token.PUNCTUATOR && t.Value == "}" {
					p.consumeToken()
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
			ret = &ast.ObjectValue{
				Fields: fields,
			}
		}
	}

	if ret == nil {
		panic(p.errorf("expected value"))
	}

	p.exit()
	return ret
}
