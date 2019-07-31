package parser

import (
	"fmt"

	"github.com/ccbrown/go-api/graphql/ast"
	"github.com/ccbrown/go-api/graphql/scanner"
	"github.com/ccbrown/go-api/graphql/token"
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

type parserToken struct {
	Token token.Token
	Value string
}

var eof = &parserToken{}

type parser struct {
	errors []*Error
	tokens []*parserToken
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
	ret := &ast.Document{}
	for p.peek() != eof {
		ret.Definitions = append(ret.Definitions, p.parseDefinition())
	}
	return ret
}

func (p *parser) parseDefinition() ast.Definition {
	if t := p.peek(); t.Token == token.NAME && t.Value == "fragment" {
		return p.parseFragmentDefinition()
	}
	return p.parseOperationDefinition()
}

func (p *parser) parseFragmentDefinition() *ast.FragmentDefinition {
	if t := p.peek(); t.Token != token.NAME || t.Value != "fragment" {
		panic(p.errorf(`expected "fragment"`))
	}
	p.consumeToken()

	return &ast.FragmentDefinition{
		Name:          p.parseName(),
		TypeCondition: p.parseTypeCondition(),
		Directives:    p.parseOptionalDirectives(),
		SelectionSet:  p.parseSelectionSet(),
	}
}

func (p *parser) parseOperationDefinition() *ast.OperationDefinition {
	if ss := p.parseOptionalSelectionSet(); ss != nil {
		return &ast.OperationDefinition{
			SelectionSet: ss,
		}
	}

	ret := &ast.OperationDefinition{}
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
	return ret
}

func (p *parser) parseOptionalSelectionSet() *ast.SelectionSet {
	if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != "{" {
		return nil
	}
	return p.parseSelectionSet()
}

func (p *parser) parseSelectionSet() *ast.SelectionSet {
	if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != "{" {
		panic(p.errorf("expected selection set"))
	}
	p.consumeToken()

	var selections []ast.Selection
	for {
		if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == "}" {
			p.consumeToken()
			break
		}
		if sel := p.parseSelection(); sel != nil {
			selections = append(selections, sel)
		} else {
			break
		}
	}
	return &ast.SelectionSet{
		Selections: selections,
	}
}

func (p *parser) parseField() *ast.Field {
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
	return ret
}

func (p *parser) parseTypeCondition() *ast.NamedType {
	if t := p.peek(); t.Token != token.NAME || t.Value != "on" {
		panic(p.errorf(`expected "on"`))
	}
	p.consumeToken()
	return p.parseNamedType()
}

func (p *parser) parseSelection() ast.Selection {
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
	return ret
}

func (p *parser) parseOptionalArguments() []*ast.Argument {
	if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != "(" {
		return nil
	}
	p.consumeToken()

	var ret []*ast.Argument
	for {
		if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == ")" {
			p.consumeToken()
			break
		}
		ret = append(ret, p.parseArgument())
	}
	return ret
}

func (p *parser) parseOptionalVariableDefinitions() []*ast.VariableDefinition {
	if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != "(" {
		return nil
	}
	p.consumeToken()

	var ret []*ast.VariableDefinition
	for {
		if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == ")" {
			p.consumeToken()
			break
		}
		ret = append(ret, p.parseVariableDefinition())
	}
	return ret
}

func (p *parser) parseVariableDefinition() *ast.VariableDefinition {
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
		ret.DefaultValue = p.parseValue()
	}
	return ret
}

func (p *parser) parseType() ast.Type {
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
	return ret
}

func (p *parser) parseArgument() *ast.Argument {
	name := p.parseName()
	if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != ":" {
		panic(p.errorf("expected colon"))
		return nil
	}
	p.consumeToken()
	value := p.parseValue()
	return &ast.Argument{
		Name:  name,
		Value: value,
	}
}

func (p *parser) parseOptionalDirectives() []*ast.Directive {
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
	return ret
}

func (p *parser) parseNamedType() *ast.NamedType {
	return &ast.NamedType{
		Name: p.parseName(),
	}
}

func (p *parser) parseName() *ast.Name {
	if t := p.peek(); t.Token == token.NAME {
		p.consumeToken()
		return &ast.Name{
			Name: t.Value,
		}
	}
	panic(p.errorf("expected name"))
}

func (p *parser) parseVariable() *ast.Variable {
	if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != "$" {
		panic(p.errorf("expected variable"))
	}
	p.consumeToken()
	return &ast.Variable{
		Name: p.parseName(),
	}
}

func (p *parser) parseValue() ast.Value {
	switch t := p.peek(); t.Token {
	case token.INT_VALUE:
		p.consumeToken()
		return &ast.IntValue{
			Value: t.Value,
		}
	case token.FLOAT_VALUE:
		p.consumeToken()
		return &ast.FloatValue{
			Value: t.Value,
		}
	case token.STRING_VALUE:
		p.consumeToken()
		return &ast.StringValue{
			Value: t.Value,
		}
	case token.NAME:
		p.consumeToken()
		switch v := t.Value; v {
		case "true", "false":
			return &ast.BooleanValue{
				Value: v == "true",
			}
		case "null":
			return &ast.NullValue{}
		default:
			return &ast.EnumValue{
				Value: v,
			}
		}
	case token.PUNCTUATOR:
		switch v := t.Value; v {
		case "$":
			return p.parseVariable()
		case "[":
			p.consumeToken()
			var values []ast.Value
			for {
				if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == "]" {
					p.consumeToken()
					break
				}
				values = append(values, p.parseValue())
			}
			return &ast.ListValue{
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
				value := p.parseValue()
				fields = append(fields, &ast.ObjectField{
					Name:  name,
					Value: value,
				})
			}
			return &ast.ObjectValue{
				Fields: fields,
			}
		}
	}
	panic(p.errorf("expected value"))
}
