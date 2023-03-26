package jsonapi

import (
	"context"
	"fmt"
	"reflect"
)

type AttributeResolver[T any] interface {
	// Resolve implementations should compute a value and write its JSON representation to `w`.
	ResolveAttribute(ctx context.Context, resource T) (any, *Error)
}

type AttributeDefinition[T any] struct {
	// Defines the type and implementation of the attribute.
	Resolver AttributeResolver[T]
}

func (def *AttributeDefinition[T]) validate() error {
	if def.Resolver == nil {
		return fmt.Errorf("attribute definitions must have a resolver")
	}
	return nil
}

// An interface which all ResourceType instantiations implement.
type AnyResourceType interface {
	get(ctx context.Context, typeName, id string) (*ResourceObject, *Error)
	validate() error
}

type ResourceType[T any] struct {
	// The attributes of the resource type. These must not overlap with the resource relationships.
	Attributes map[string]*AttributeDefinition[T]

	// If given, the resource can be directly referenced using an id, e.g. via the /{type_name}/{id}
	// endpoint.
	Getter func(ctx context.Context, id string) (T, *Error)
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) && rv.IsNil()
}

func (t ResourceType[T]) get(ctx context.Context, typeName, id string) (*ResourceObject, *Error) {
	if t.Getter == nil {
		return nil, nil
	}

	resource, err := t.Getter(ctx, id)
	if err != nil || isNil(resource) {
		return nil, err
	}

	ret := ResourceObject{
		Type: typeName,
		Id:   id,
	}

	if len(t.Attributes) > 0 {
		ret.Attributes = make(map[string]any, len(t.Attributes))

		for name, def := range t.Attributes {
			if v, err := def.Resolver.ResolveAttribute(ctx, resource); err != nil {
				return nil, err
			} else {
				ret.Attributes[name] = v
			}
		}
	}

	return &ret, nil
}

func (t ResourceType[T]) validate() error {
	for name, def := range t.Attributes {
		if name == "id" || name == "type" {
			return fmt.Errorf("illegal attribute name: %v", name)
		} else if err := validateMemberName(name); err != nil {
			return fmt.Errorf("invalid attribute name %v: %w", name, err)
		} else if err := def.validate(); err != nil {
			return fmt.Errorf("invalid attribute %v: %w", name, err)
		}
	}

	return nil
}

type ResourceObject struct {
	Type string `json:"type"`

	Id string `json:"id"`

	// An attributes object representing some of the resource’s data.
	Attributes map[string]any `json:"attributes,omitempty"`

	// A relationships object describing relationships between the resource and other JSON:API
	// resources.
	Relationships map[string]any `json:"relationships,omitempty"`

	// A links object containing links related to the resource.
	Links LinksObject `json:"links,omitempty"`

	// A meta object containing non-standard meta-information about the resource that can not be
	// represented as an attribute or relationship.
	Meta map[string]any `json:"meta,omitempty"`
}
