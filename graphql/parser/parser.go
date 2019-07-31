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

type parserToken struct {
	Token token.Token
	Value string
}

var eof = &parserToken{}

type Parser struct {
	errors []*Error
	tokens []*parserToken
}

func New(src []byte) *Parser {
	var tokens []*parserToken
	s := scanner.New(src, 0)
	for s.Scan() {
		tokens = append(tokens, &parserToken{
			Token: s.Token(),
			Value: s.StringValue(),
		})
	}
	ret := &Parser{
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

func (p *Parser) peek() *parserToken {
	if len(p.tokens) > 0 {
		return p.tokens[0]
	}
	return eof
}

func (p *Parser) consumeToken() {
	if len(p.tokens) > 0 {
		p.tokens = p.tokens[1:]
	}
}

func (p *Parser) Errors() []*Error {
	return p.errors
}

func (p *Parser) errorf(message string, args ...interface{}) {
	p.errors = append(p.errors, &Error{
		message: fmt.Sprintf(message, args...),
	})
}

func (p *Parser) ParseArgument() *ast.Argument {
	name := p.ParseName()
	if name == nil {
		return nil
	}
	if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != ":" {
		p.errorf("expected colon")
		return nil
	}
	p.consumeToken()
	value := p.ParseValue()
	if value == nil {
		return nil
	}
	return &ast.Argument{
		Name:  name,
		Value: value,
	}
}

func (p *Parser) ParseName() *ast.Name {
	if t := p.peek(); t.Token == token.NAME {
		p.consumeToken()
		return &ast.Name{
			Name: t.Value,
		}
	}
	p.errorf("expected name")
	return nil
}

func (p *Parser) ParseValue() ast.Value {
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
			return &ast.Variable{
				Name: p.ParseName(),
			}
		case "[":
			p.consumeToken()
			var values []ast.Value
			for {
				if t := p.peek(); t.Token == token.PUNCTUATOR && t.Value == "]" {
					p.consumeToken()
					break
				}
				value := p.ParseValue()
				if value == nil {
					break
				}
				values = append(values, value)
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
				name := p.ParseName()
				if name == nil {
					break
				}
				if t := p.peek(); t.Token != token.PUNCTUATOR || t.Value != ":" {
					p.errorf("expected colon")
					break
				}
				p.consumeToken()
				value := p.ParseValue()
				if value == nil {
					break
				}
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
	p.errorf("expected value")
	return nil
}
