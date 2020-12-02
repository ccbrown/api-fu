package main

import (
	"encoding/json"
	"fmt"
	goast "go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
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
	output             string
	schema             *schema.Schema
	wrapper            string
	outputStructCount  int
	outputEnums        map[string]struct{}
	requiresJSONImport bool
}

func (s *generateState) generateType(t schema.Type, selections []ast.Selection, nonNull bool, fragTypes map[string]string) (string, error) {
	if t, ok := t.(*schema.NonNullType); ok {
		return s.generateType(t.Type, selections, true, fragTypes)
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
		gen, err := s.generateType(t.Type, selections, false, fragTypes)
		if err != nil {
			return "", err
		}
		ret = "[]" + gen
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
	case *schema.ObjectType, *schema.InterfaceType, *schema.UnionType:
		fields := map[string]string{}

		hasTypename := false
		for _, sel := range selections {
			if field, ok := sel.(*ast.Field); ok {
				if field.Name.Name == "__typename" {
					hasTypename = true
					break
				}
			}
		}

		// type => field names
		typeConditions := map[string][]string{}

		for _, sel := range selections {
			switch sel := sel.(type) {
			case *ast.FragmentSpread:
				if !hasTypename {
					if _, ok := t.(*schema.ObjectType); !ok {
						return "", fmt.Errorf("__typename is required by fragment spread")
					}
				}
				name := sel.FragmentName.Name
				fields[name] = "*" + name + "Fragment `json:\"-\"`"
				typeConditions[fragTypes[name]] = append(typeConditions[fragTypes[name]], name)
			case *ast.InlineFragment:
				if !hasTypename {
					if _, ok := t.(*schema.ObjectType); !ok {
						return "", fmt.Errorf("__typename is required by inline fragment")
					}
				}
				cond := s.schema.NamedTypes()[sel.TypeCondition.Name.Name]
				gen, err := s.generateType(cond, sel.SelectionSet.Selections, false, fragTypes)
				if err != nil {
					return "", err
				}
				fields[cond.TypeName()] = gen + " `json:\"-\"`"
				typeConditions[cond.TypeName()] = append(typeConditions[cond.TypeName()], cond.TypeName())
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
				if sel.Name.Name == "__typename" {
					fields["Typename__"] = "string `json:\"__typename\"`"
				} else {
					var err error
					switch t := t.(type) {
					case *schema.ObjectType:
						fields[k], err = s.generateType(t.Fields[sel.Name.Name].Type, selections, false, fragTypes)
					case *schema.InterfaceType:
						fields[k], err = s.generateType(t.Fields[sel.Name.Name].Type, selections, false, fragTypes)
					}
					if err != nil {
						return "", err
					}
				}
			}
		}

		parts := make([]string, 0, len(fields))
		for k, v := range fields {
			parts = append(parts, k+" "+v+"\n")
		}
		ret = "struct {\n" + strings.Join(parts, "") + "}"

		if len(typeConditions) > 0 {
			s.requiresJSONImport = true
			tName := t.(schema.NamedType).TypeName()
			name := "sel" + tName + strconv.Itoa(s.outputStructCount)
			s.output += `
				type ` + name + ` ` + ret + `

				func (s *` + name + `) UnmarshalJSON(b []byte) error {
					var base ` + ret + `
					if err := json.Unmarshal(b, &base); err != nil {
						return err
					}
					*s = base
			`
			for typeCond, fields := range typeConditions {
				isKnown := typeCond == tName
				if obj, ok := t.(*schema.ObjectType); ok && !isKnown {
					for _, iface := range obj.ImplementedInterfaces {
						if iface.Name == typeCond {
							isKnown = true
							break
						}
					}
				}
				if isKnown {
					for _, field := range fields {
						s.output += `if err := json.Unmarshal(b, &s.` + field + `); err != nil {
								return err
							}
						`
					}
					continue
				}

				typeCondType := s.schema.NamedTypes()[typeCond]
				var okTypes []string
				switch t := typeCondType.(type) {
				case *schema.InterfaceType:
					for _, t := range s.schema.InterfaceImplementations(t.Name) {
						okTypes = append(okTypes, t.Name)
					}
				case *schema.ObjectType:
					okTypes = []string{t.Name}
				}

				for _, field := range fields {
					s.output += `switch base.Typename__ {
						case "` + strings.Join(okTypes, `", "`) + `":
							if err := json.Unmarshal(b, &s.` + field + `); err != nil {
								return err
							}
						}
					`
				}
			}
			s.output += "return nil\n}\n\n"
			ret = name
			s.outputStructCount++
		}

		if !nonNull {
			ret = "*" + ret
		}
	}

	return ret, nil
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

	fragTypes := map[string]string{}
	for _, op := range doc.Definitions {
		if def, ok := op.(*ast.FragmentDefinition); ok {
			fragTypes[def.Name.Name] = def.TypeCondition.Name.Name
		}
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
					t = s.schema.SubscriptionType()
				}
			}
			if op.Name != nil {
				gen, err := s.generateType(t, op.SelectionSet.Selections, true, fragTypes)
				if err != nil {
					ret = append(ret, err)
					continue
				}
				s.output += "type " + op.Name.Name + "Data " + gen + "\n\n"
			}
		case *ast.FragmentDefinition:
			if op.Name != nil {
				gen, err := s.generateType(s.schema.NamedTypes()[op.TypeCondition.Name.Name], op.SelectionSet.Selections, true, fragTypes)
				if err != nil {
					ret = append(ret, err)
					continue
				}
				s.output += "type " + op.Name.Name + "Fragment " + gen + "\n\n"
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

	tmp := state.output
	state.output = "package " + pkg + "\n\n"
	if state.requiresJSONImport {
		state.output += "import \"encoding/json\"\n\n"
	}
	state.output += tmp

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

func Run(w io.Writer, args ...string) []error {
	flags := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	pkg := flags.String("pkg", "", "the package name of the generated output")
	input := flags.StringArrayP("input", "i", nil, "the input files to search")
	schemaPath := flags.String("schema", "", "the path to the schema json file")
	wrapper := flags.String("wrapper", "gql", "the wrapper name to look for")
	flags.Parse(args)

	if *pkg == "" {
		return []error{fmt.Errorf("the --pkg flag is required")}
	}

	if *schemaPath == "" {
		return []error{fmt.Errorf("the --schema flag is required")}
	}

	schema, err := LoadSchema(*schemaPath)
	if err != nil {
		return []error{fmt.Errorf("error loading schema: %w", err)}
	}

	output, errs := Generate(schema, *pkg, *input, *wrapper)
	if len(errs) > 0 {
		return errs
	}

	fmt.Fprint(w, output)
	return nil
}

func main() {
	if errs := Run(os.Stdout, os.Args[1:]...); len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		os.Exit(1)
	}
}
