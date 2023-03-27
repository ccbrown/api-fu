package jsonapi

import (
	"context"
	"fmt"
	"reflect"
)

type AttributeResolver[T any] interface {
	// Resolve implementations should compute a value and return a JSON-serializable object.
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

type RelationshipResolver[T any] interface {
	// Resolve implementations should compute a value and return a JSON-serializable resource
	// linkage object.
	//
	// Generally you should use `ToOneRelationshipResolver` or `ToManyRelationshipResolver` instead
	// of implementing this directly.
	ResolveRelationship(ctx context.Context, resource T, resourceId ResourceId, relationshipName string) (Relationship, *Error)
}

type ToOneRelationshipResolver[T any] struct {
	Resolve func(ctx context.Context, resource T) (*ResourceId, *Error)
}

func (r ToOneRelationshipResolver[T]) ResolveRelationship(ctx context.Context, resource T, resourceId ResourceId, relationshipName string) (Relationship, *Error) {
	if id, err := r.Resolve(ctx, resource); err != nil {
		return Relationship{}, err
	} else {
		links := Links{
			"self":    "/" + resourceId.Type + "/" + resourceId.Id + "/relationships/" + relationshipName,
			"related": "/" + resourceId.Type + "/" + resourceId.Id + "/" + relationshipName,
		}
		var data any = nil
		if id != nil {
			data = *id
		}
		return Relationship{
			Links: links,
			Data:  data,
		}, nil
	}
}

type ToManyRelationshipResolver[T any] struct {
	Resolve func(ctx context.Context, resource T) ([]ResourceId, *Error)
}

func (r ToManyRelationshipResolver[T]) ResolveRelationship(ctx context.Context, resource T, resourceId ResourceId, relationshipName string) (Relationship, *Error) {
	if data, err := r.Resolve(ctx, resource); err != nil {
		return Relationship{}, err
	} else {
		if len(data) == 0 {
			data = []ResourceId{}
		}
		links := Links{
			"self":    "/" + resourceId.Type + "/" + resourceId.Id + "/relationships/" + relationshipName,
			"related": "/" + resourceId.Type + "/" + resourceId.Id + "/" + relationshipName,
		}
		return Relationship{
			Links: links,
			Data:  data,
		}, nil
	}
}

type RelationshipDefinition[T any] struct {
	// Defines the type and implementation of the relationship.
	Resolver RelationshipResolver[T]
}

func (def *RelationshipDefinition[T]) validate() error {
	if def.Resolver == nil {
		return fmt.Errorf("relationship definitions must have a resolver")
	}
	return nil
}

// An interface which all ResourceType instantiations implement.
type AnyResourceType interface {
	get(ctx context.Context, id ResourceId) (*Resource, *Error)
	getRelationship(ctx context.Context, id ResourceId, relationshipName string) (*Relationship, *Error)
	validate() error
}

type ResourceType[T any] struct {
	// The attributes of the resource type. These must not overlap with the resource relationships.
	Attributes map[string]*AttributeDefinition[T]

	// The relationships of the resource type. These must not overlap with the resource attributes.
	Relationships map[string]*RelationshipDefinition[T]

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

func (t ResourceType[T]) get(ctx context.Context, id ResourceId) (*Resource, *Error) {
	if t.Getter == nil {
		return nil, nil
	}

	resource, err := t.Getter(ctx, id.Id)
	if err != nil || isNil(resource) {
		return nil, err
	}

	ret := Resource{
		Type: id.Type,
		Id:   id.Id,
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

	if len(t.Relationships) > 0 {
		ret.Relationships = make(map[string]any, len(t.Relationships))

		for name, def := range t.Relationships {
			if rel, err := def.Resolver.ResolveRelationship(ctx, resource, id, name); err != nil {
				return nil, err
			} else {
				ret.Relationships[name] = rel
			}
		}
	}

	return &ret, nil
}

func (t ResourceType[T]) getRelationship(ctx context.Context, id ResourceId, relationshipName string) (*Relationship, *Error) {
	if t.Getter == nil {
		return nil, nil
	}

	resource, err := t.Getter(ctx, id.Id)
	if err != nil || isNil(resource) {
		return nil, err
	}

	if def, ok := t.Relationships[relationshipName]; ok {
		if ret, err := def.Resolver.ResolveRelationship(ctx, resource, id, relationshipName); err != nil {
			return nil, err
		} else {
			return &ret, nil
		}
	}

	return nil, nil
}

func (t ResourceType[T]) validate() error {
	for name, def := range t.Attributes {
		if name == "id" || name == "type" {
			return fmt.Errorf("illegal attribute name: %v", name)
		} else if _, ok := t.Relationships[name]; ok {
			return fmt.Errorf("attributes and relationships cannot have the same name: %v", name)
		} else if err := validateMemberName(name); err != nil {
			return fmt.Errorf("invalid attribute name %v: %w", name, err)
		} else if err := def.validate(); err != nil {
			return fmt.Errorf("invalid attribute %v: %w", name, err)
		}
	}

	for name, def := range t.Relationships {
		if name == "id" || name == "type" {
			return fmt.Errorf("illegal relationship name: %v", name)
		} else if err := validateMemberName(name); err != nil {
			return fmt.Errorf("invalid relationship name %v: %w", name, err)
		} else if err := def.validate(); err != nil {
			return fmt.Errorf("invalid relationship %v: %w", name, err)
		}
	}

	return nil
}

type Resource struct {
	Type string `json:"type"`

	Id string `json:"id"`

	// An attributes object representing some of the resource’s data.
	Attributes map[string]any `json:"attributes,omitempty"`

	// A relationships object describing relationships between the resource and other JSON:API
	// resources.
	Relationships map[string]any `json:"relationships,omitempty"`

	// A links object containing links related to the resource.
	Links Links `json:"links,omitempty"`

	// A meta object containing non-standard meta-information about the resource that can not be
	// represented as an attribute or relationship.
	Meta map[string]any `json:"meta,omitempty"`
}

type Relationship struct {
	// A links object containing at least one of the following:
	//
	// - self: a link for the relationship itself (a “relationship link”)
	// - related: a related resource link
	// - a member defined by an applied extension
	Links Links `json:"links,omitempty"`

	// The resource linkage.
	//
	// This must be `nil`, `ResourceId`, or `[]ResourceId`.
	Data any `json:"data,omitempty"`

	// A meta object containing non-standard meta-information about the relationship.
	Meta map[string]any `json:"meta,omitempty"`
}

type ResourceId struct {
	Type string `json:"type"`

	Id string `json:"id"`
}
