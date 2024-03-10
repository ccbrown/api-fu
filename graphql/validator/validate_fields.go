package validator

import (
	"fmt"

	"github.com/ccbrown/api-fu/graphql/ast"
	"github.com/ccbrown/api-fu/graphql/schema"
	"github.com/ccbrown/api-fu/graphql/schema/introspection"
)

func validateFields(doc *ast.Document, s *schema.Schema, features schema.FeatureSet, typeInfo *TypeInfo) []*Error {
	var ret []*Error

	fragmentDefinitions := map[string]*ast.FragmentDefinition{}
	for _, def := range doc.Definitions {
		if def, ok := def.(*ast.FragmentDefinition); ok {
			fragmentDefinitions[def.Name.Name] = def
		}
	}

	var selectionSetTypes []schema.NamedType
	ast.Inspect(doc, func(node ast.Node) bool {
		if node == nil {
			selectionSetTypes = selectionSetTypes[:len(selectionSetTypes)-1]
			return true
		}

		var selectionSetType schema.NamedType

		switch node := node.(type) {
		case *ast.SelectionSet:
			selectionSetType = typeInfo.SelectionSetTypes[node]
		case *ast.Field:
			name := node.Name.Name

			shouldHaveSubselection := false

			if def := typeInfo.FieldDefinitions[node]; def != nil {
				switch schema.UnwrappedType(def.Type).(type) {
				case *schema.ObjectType, *schema.InterfaceType, *schema.UnionType:
					shouldHaveSubselection = true
				}
			} else if def == nil && name != "__typename" {
				ret = append(ret, newSecondaryError(node, "no type info for field"))
			}

			fieldExists := true

			if name != "__typename" {
				switch parent := selectionSetTypes[len(selectionSetTypes)-1].(type) {
				case *schema.ObjectType:
					if parent.GetField(name, features) == nil && (parent != s.QueryType() || introspection.MetaFields[name] == nil) {
						ret = append(ret, newError(node.Name, "field %v does not exist on %v", name, parent.Name))
						fieldExists = false
					}
				case *schema.InterfaceType:
					if parent.GetField(name, features) == nil {
						ret = append(ret, newError(node.Name, "field %v does not exist on %v", name, parent.Name))
						fieldExists = false
					}
				case *schema.UnionType:
					ret = append(ret, newError(node.Name, "field %v does not exist on %v", name, parent.Name))
					fieldExists = false
				}
			}

			if fieldExists {
				if shouldHaveSubselection {
					if node.SelectionSet == nil || len(node.SelectionSet.Selections) == 0 {
						ret = append(ret, newError(node, "%v field must have a subselection", name))
					}
				} else {
					if node.SelectionSet != nil {
						ret = append(ret, newError(node, "%v field cannot have a subselection", name))
					}
				}
			}
		}

		selectionSetTypes = append(selectionSetTypes, selectionSetType)
		return true
	})

	ast.Inspect(doc, func(node ast.Node) bool {
		if node, ok := node.(*ast.SelectionSet); ok {
			set := map[string][]fieldAndParent{}
			if err := addFieldSelections(set, node, fragmentDefinitions); err != nil {
				ret = append(ret, err)
				return false
			} else if err := validateFieldsInSetCanMerge(set, fragmentDefinitions, typeInfo); err != nil {
				ret = append(ret, err)
				return false
			}
		}
		return true
	})

	return ret
}

type fieldAndParent struct {
	field  *ast.Field
	parent *ast.SelectionSet
}

func validateFieldsInSetCanMerge(fieldsForName map[string][]fieldAndParent, fragmentDefinitions map[string]*ast.FragmentDefinition, typeInfo *TypeInfo) *Error {
	for _, fields := range fieldsForName {
		for i := 0; i < len(fields); i++ {
			for j := i + 1; j < len(fields); j++ {
				fieldA := fields[i].field
				fieldB := fields[j].field
				if err := validateSameResponseShape(fieldA, fieldB, fragmentDefinitions, typeInfo); err != nil {
					return err
				}

				parentTypeA := typeInfo.SelectionSetTypes[fields[i].parent]
				parentTypeB := typeInfo.SelectionSetTypes[fields[j].parent]
				if parentTypeA == nil {
					return newSecondaryError(fields[i].parent, "no type info for selection set")
				} else if parentTypeB == nil {
					return newSecondaryError(fields[j].parent, "no type info for selection set")
				}

				if parentTypeA.IsSameType(parentTypeB) || !schema.IsObjectType(parentTypeA) || !schema.IsObjectType(parentTypeB) {
					if fieldA.Name.Name != fieldB.Name.Name {
						return newErrorWithNodes([]ast.Node{fieldA.Name, fieldB.Name}, "cannot merge fields with different names")
					}

					if len(fieldA.Arguments) != len(fieldB.Arguments) {
						return newErrorWithNodes([]ast.Node{fieldA, fieldB}, "cannot merge fields with differing arguments")
					} else {
						argsA := map[string]*ast.Argument{}
						for _, arg := range fieldA.Arguments {
							argsA[arg.Name.Name] = arg
						}
						for _, argB := range fieldB.Arguments {
							if argA, ok := argsA[argB.Name.Name]; !ok || !valuesAreIdentical(argA.Value, argB.Value) {
								return newErrorWithNodes([]ast.Node{argA, argB}, "cannot merge fields with differing arguments")
							}
						}
					}

					mergedSet := map[string][]fieldAndParent{}
					if err := addFieldSelections(mergedSet, fieldA.SelectionSet, fragmentDefinitions); err != nil {
						return err
					} else if err := addFieldSelections(mergedSet, fieldB.SelectionSet, fragmentDefinitions); err != nil {
						return err
					} else if err := validateFieldsInSetCanMerge(mergedSet, fragmentDefinitions, typeInfo); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func valuesAreIdentical(a, b ast.Value) bool {
	switch a := a.(type) {
	case *ast.Variable:
		b, ok := b.(*ast.Variable)
		return ok && b.Name.Name == a.Name.Name
	case *ast.BooleanValue:
		b, ok := b.(*ast.BooleanValue)
		return ok && b.Value == a.Value
	case *ast.FloatValue:
		b, ok := b.(*ast.FloatValue)
		return ok && b.Value == a.Value
	case *ast.IntValue:
		b, ok := b.(*ast.IntValue)
		return ok && b.Value == a.Value
	case *ast.StringValue:
		b, ok := b.(*ast.StringValue)
		return ok && b.Value == a.Value
	case *ast.EnumValue:
		b, ok := b.(*ast.EnumValue)
		return ok && b.Value == a.Value
	case *ast.NullValue:
		_, ok := b.(*ast.NullValue)
		return ok
	case *ast.ListValue:
		b, ok := b.(*ast.ListValue)
		if !ok || len(a.Values) != len(b.Values) {
			return false
		}
		for i := 0; i < len(a.Values); i++ {
			if !valuesAreIdentical(a.Values[i], b.Values[i]) {
				return false
			}
		}
		return true
	case *ast.ObjectValue:
		b, ok := b.(*ast.ObjectValue)
		if !ok || len(a.Fields) != len(b.Fields) {
			return false
		}
		for i := 0; i < len(a.Fields); i++ {
			a := a.Fields[i]
			b := b.Fields[i]
			if a.Name.Name != b.Name.Name || !valuesAreIdentical(a.Value, b.Value) {
				return false
			}
		}
		return true
	}
	panic(fmt.Sprintf("unexpected value type: %T", a))
}

func validateSameResponseShape(fieldA, fieldB *ast.Field, fragmentDefinitions map[string]*ast.FragmentDefinition, typeInfo *TypeInfo) *Error {
	var typeA, typeB schema.Type

	if fieldA.Name.Name == "__typename" {
		typeA = schema.NewNonNullType(schema.StringType)
	} else {
		fieldDefA := typeInfo.FieldDefinitions[fieldA]
		if fieldDefA == nil {
			return newSecondaryError(fieldA, "no type info for field")
		}
		typeA = fieldDefA.Type
	}

	if fieldB.Name.Name == "__typename" {
		typeB = schema.NewNonNullType(schema.StringType)
	} else {
		fieldDefB := typeInfo.FieldDefinitions[fieldB]
		if fieldDefB == nil {
			return newSecondaryError(fieldB, "no type info for field")
		}
		typeB = fieldDefB.Type
	}

	for {
		if schema.IsNonNullType(typeA) || schema.IsNonNullType(typeB) {
			if nonNullTypeA, ok := typeA.(*schema.NonNullType); ok {
				typeA = nonNullTypeA.Type
			} else {
				return newErrorWithNodes([]ast.Node{fieldA, fieldB}, "cannot merge non-null and nullable fields")
			}
			if nonNullTypeB, ok := typeB.(*schema.NonNullType); ok {
				typeB = nonNullTypeB.Type
			} else {
				return newErrorWithNodes([]ast.Node{fieldA, fieldB}, "cannot merge non-null and nullable fields")
			}
		}

		if schema.IsListType(typeA) || schema.IsListType(typeB) {
			if listTypeA, ok := typeA.(*schema.ListType); ok {
				typeA = listTypeA.Type
			} else {
				return newErrorWithNodes([]ast.Node{fieldA, fieldB}, "cannot merge list and non-list fields")
			}
			if listTypeB, ok := typeB.(*schema.ListType); ok {
				typeB = listTypeB.Type
			} else {
				return newErrorWithNodes([]ast.Node{fieldA, fieldB}, "cannot merge list and non-list fields")
			}
		} else {
			break
		}
	}

	if schema.IsScalarType(typeA) || schema.IsScalarType(typeB) || schema.IsEnumType(typeA) || schema.IsEnumType(typeB) {
		if typeA.IsSameType(typeB) {
			return nil
		}
		return newErrorWithNodes([]ast.Node{fieldA, fieldB}, "non-composite fields of the same name must be the same")
	}

	fieldsForName := map[string][]fieldAndParent{}
	if err := addFieldSelections(fieldsForName, fieldA.SelectionSet, fragmentDefinitions); err != nil {
		return err
	} else if err := addFieldSelections(fieldsForName, fieldB.SelectionSet, fragmentDefinitions); err != nil {
		return err
	}

	for _, fields := range fieldsForName {
		for i := 0; i < len(fields); i++ {
			for j := i + 1; j < len(fields); j++ {
				if err := validateSameResponseShape(fields[i].field, fields[j].field, fragmentDefinitions, typeInfo); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func addFieldSelections(fieldsForName map[string][]fieldAndParent, selectionSet *ast.SelectionSet, fragmentDefinitions map[string]*ast.FragmentDefinition) *Error {
	visited := map[*ast.SelectionSet]struct{}{}
	return addFieldSelectionsWithCycleDetection(fieldsForName, selectionSet, fragmentDefinitions, visited)
}

func addFieldSelectionsWithCycleDetection(fieldsForName map[string][]fieldAndParent, selectionSet *ast.SelectionSet, fragmentDefinitions map[string]*ast.FragmentDefinition, visited map[*ast.SelectionSet]struct{}) *Error {
	if selectionSet == nil {
		return nil
	}

	if _, ok := visited[selectionSet]; ok {
		return newSecondaryError(selectionSet, "cycle detected")
	}
	visited[selectionSet] = struct{}{}

	for _, selection := range selectionSet.Selections {
		switch selection := selection.(type) {
		case *ast.Field:
			name := selection.Name.Name
			if selection.Alias != nil {
				name = selection.Alias.Name
			}
			fieldsForName[name] = append(fieldsForName[name], fieldAndParent{
				field:  selection,
				parent: selectionSet,
			})
		case *ast.InlineFragment:
			if err := addFieldSelectionsWithCycleDetection(fieldsForName, selection.SelectionSet, fragmentDefinitions, visited); err != nil {
				return err
			}
		case *ast.FragmentSpread:
			if def, ok := fragmentDefinitions[selection.FragmentName.Name]; !ok {
				return newSecondaryError(selection.FragmentName, "undefined fragment")
			} else if err := addFieldSelectionsWithCycleDetection(fieldsForName, def.SelectionSet, fragmentDefinitions, visited); err != nil {
				return err
			}
		}
	}
	return nil
}
