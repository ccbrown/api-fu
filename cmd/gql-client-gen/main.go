package main

import (
	"encoding/json"
	"fmt"
	goast "go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
	"github.com/ccbrown/api-fu/graphql/schema/introspection"
)

type generateState struct {
	output      string
	schema      *schema.Schema
	wrapper     string
	outputEnums map[string]struct{}
}

func (s *generateState) generateType(t schema.Type, selections []ast.Selection, nonNull bool) string {
	if t, ok := t.(*schema.NonNullType); ok {
		return s.generateType(t.Type, selections, true)
	}

	ret := "interface{}"

	switch t := t.(type) {
	case *schema.ScalarType:
		switch t {
		case schema.BooleanType:
			ret = "bool"
		case schema.IntType:
			ret = "int"
		case schema.FloatType:
			ret = "float64"
		case schema.StringType:
			ret = "string"
		case schema.IDType:
			ret = "string"
		default:
			ret = t.Name
		}

		if !nonNull {
			ret = "*" + ret
		}
	case *schema.ListType:
		ret = "[]" + s.generateType(t.Type, selections, false)
	case *schema.EnumType:
		if _, ok := s.outputEnums[t.Name]; !ok {
			s.output += "type " + t.Name + " string\n\nconst (\n"
			for k := range t.Values {
				parts := strings.Split(k, "_")
				for i, part := range parts {
					parts[i] = strings.Title(strings.ToLower(part))
				}
				s.output += t.Name + strings.Join(parts, "") + " " + t.Name + " = \"" + k + "\"\n"
			}
			s.output += ")\n\n"
			s.outputEnums[t.Name] = struct{}{}
		}

		ret = t.Name

		if !nonNull {
			ret = "*" + ret
		}
	case *schema.ObjectType, *schema.InterfaceType:
		fields := map[string]string{}
		for _, sel := range selections {
			switch sel := sel.(type) {
			case *ast.FragmentSpread:
				name := sel.FragmentName.Name
				fields[name] = name + "Fragment"
			case *ast.InlineFragment:
				cond := s.schema.NamedTypes()[sel.TypeCondition.Name.Name]
				fields[cond.TypeName()] = s.generateType(cond, sel.SelectionSet.Selections, false)
			case *ast.Field:
				var selections []ast.Selection
				if sel.SelectionSet != nil {
					selections = sel.SelectionSet.Selections
				}
				k := sel.Name.Name
				if sel.Alias != nil {
					k = sel.Alias.Name
				}
				k = strings.Title(k)
				switch t := t.(type) {
				case *schema.ObjectType:
					fields[k] = s.generateType(t.Fields[sel.Name.Name].Type, selections, false)
				case *schema.InterfaceType:
					fields[k] = s.generateType(t.Fields[sel.Name.Name].Type, selections, false)
				}
			}
		}

		parts := make([]string, 0, len(fields))
		for k, v := range fields {
			parts = append(parts, k+" "+v+"\n")
		}
		ret = "struct {\n" + strings.Join(parts, "") + "}"

		if !nonNull {
			ret = "*" + ret
		}
	}

	return ret
}

func (s *generateState) processQuery(q string) []error {
	var ret []error
	doc, errs := graphql.ParseAndValidate(q, s.schema)
	if len(errs) > 0 {
		for _, err := range errs {
			ret = append(ret, err)
		}
		return ret
	}

	for _, op := range doc.Definitions {
		switch op := op.(type) {
		case *ast.OperationDefinition:
			t := s.schema.QueryType()
			if op.OperationType != nil {
				switch op.OperationType.Value {
				case "mutation":
					t = s.schema.MutationType()
				case "subscription":
					continue
				}
			}
			if op.Name != nil {
				s.output += "type " + op.Name.Name + "Data " + s.generateType(t, op.SelectionSet.Selections, true) + "\n\n"
			}
		case *ast.FragmentDefinition:
			if op.Name != nil {
				s.output += "type " + op.Name.Name + "Fragment " + s.generateType(s.schema.NamedTypes()[op.TypeCondition.Name.Name], op.SelectionSet.Selections, true) + "\n\n"
			}
		}
	}

	return ret
}

func (s *generateState) processFile(path string) []error {
	source, err := ioutil.ReadFile(path)
	if err != nil {
		return []error{err}
	}

	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, "", source, 0)
	if err != nil {
		return []error{fmt.Errorf("parse error: %w", err)}
	}

	var errs []error

	goast.Inspect(f, func(node goast.Node) bool {
		switch node := node.(type) {
		case *goast.CallExpr:
			if ident, ok := node.Fun.(*goast.Ident); !ok || ident.Name != s.wrapper {
				return true
			} else if len(node.Args) != 1 {
				errs = append(errs, fmt.Errorf("%v: expected 1 argument to %v", fset.Position(node.Lparen), s.wrapper))
			} else if lit, ok := node.Args[0].(*goast.BasicLit); !ok || lit.Kind != token.STRING {
				errs = append(errs, fmt.Errorf("%v: %v argument must be a string literal", fset.Position(node.Args[0].Pos()), s.wrapper))
			} else if q, err := strconv.Unquote(lit.Value); err != nil {
				errs = append(errs, fmt.Errorf("%v: error parsing argument: %w", fset.Position(node.Args[0].Pos()), err))
			} else {
				for _, err := range s.processQuery(q) {
					errs = append(errs, fmt.Errorf("%v: %w", fset.Position(node.Args[0].Pos()), err))
				}
			}
		}
		return true
	})

	return errs
}

func Generate(schema *schema.Schema, pkg string, inputGlobs []string, wrapper string) (string, []error) {
	state := &generateState{
		output:      "package " + pkg + "\n\n",
		schema:      schema,
		wrapper:     wrapper,
		outputEnums: map[string]struct{}{},
	}

	var errs []error
	for _, glob := range inputGlobs {
		matches, err := filepath.Glob(glob)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, match := range matches {
			for _, err := range state.processFile(match) {
				errs = append(errs, fmt.Errorf("%v: %w", match, err))
			}
		}
	}

	if len(errs) > 0 {
		return "", errs
	}

	out, err := format.Source([]byte(state.output))
	if err != nil {
		return "", []error{fmt.Errorf("error formatting result: %w", err)}
	}
	return string(out), nil
}

func LoadSchema(path string) (*schema.Schema, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var result struct {
		Data struct {
			Schema introspection.SchemaData `json:"__schema"`
		}
	}
	if err := json.NewDecoder(f).Decode(&result); err != nil {
		return nil, err
	}

	def, err := result.Data.Schema.GetSchemaDefinition()
	if err != nil {
		return nil, err
	}

	return schema.New(def)
}

func main() {
	pkg := pflag.String("pkg", "", "the package name of the generated output")
	input := pflag.StringArrayP("input", "i", nil, "the input files to search")
	schemaPath := pflag.String("schema", "", "the path to the schema json file")
	wrapper := pflag.String("wrapper", "gql", "the wrapper name to look for")
	pflag.Parse()

	if *pkg == "" {
		fmt.Fprintln(os.Stderr, "the --pkg flag is required")
		os.Exit(1)
	}

	if *schemaPath == "" {
		fmt.Fprintln(os.Stderr, "the --schema flag is required")
		os.Exit(1)
	}

	schema, err := LoadSchema(*schemaPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading schema: "+err.Error())
		os.Exit(1)
	}

	output, errs := Generate(schema, *pkg, *input, *wrapper)
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		os.Exit(1)
	}

	fmt.Fprint(os.Stdout, output)
}
