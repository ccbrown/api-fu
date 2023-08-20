package jsonapi

import "fmt"

type Schema struct {
	resourceTypes map[string]AnyResourceType
}

func NewSchema(def *SchemaDefinition) (*Schema, error) {
	ret := &Schema{
		resourceTypes: def.ResourceTypes,
	}

	for name, t := range def.ResourceTypes {
		if err := validateMemberName(name); err != nil {
			return nil, fmt.Errorf("invalid resource type name: %w", err)
		} else if err := t.validate(); err != nil {
			return nil, fmt.Errorf("invalid resource type %v: %w", name, err)
		}
	}

	return ret, nil
}

type SchemaDefinition struct {
	// The schema's resource types. Convention is for names to be lowercase, plural name such as
	// "articles".
	ResourceTypes map[string]AnyResourceType
}
