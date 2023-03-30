package jsonapi

import (
	"context"
	"fmt"
	"net/url"
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
	// Resolve implementations should compute a value and return a `Relationship` or an error. The
	// `Relationship` will automatically have links added to it, but resolvers may add additional
	// links to the result.
	//
	// If `dataRequested` is false, resolvers may choose to omit the `Data` field from the result.
	//
	// Generally you should use `ToOneRelationshipResolver` or `ToManyRelationshipResolver` instead
	// of implementing this directly.
	ResolveRelationship(ctx context.Context, resource T, dataRequested bool, params url.Values) (Relationship, *Error)
}

type ToOneRelationshipResolver[T any] struct {
	ResolveByDefault bool

	Resolve func(ctx context.Context, resource T) (*ResourceId, *Error)
}

func (r ToOneRelationshipResolver[T]) ResolveRelationship(ctx context.Context, resource T, dataRequested bool, params url.Values) (Relationship, *Error) {
	if dataRequested || r.ResolveByDefault {
		if id, err := r.Resolve(ctx, resource); err != nil {
			return Relationship{}, err
		} else if id != nil {
			var data any = *id
			return Relationship{Data: &data}, nil
		}
	}
	return Relationship{}, nil
}

type ToManyRelationshipResolver[T any] struct {
	ResolveByDefault bool

	Resolve func(ctx context.Context, resource T) ([]ResourceId, *Error)
}

func (r ToManyRelationshipResolver[T]) ResolveRelationship(ctx context.Context, resource T, dataRequested bool, params url.Values) (Relationship, *Error) {
	if dataRequested || r.ResolveByDefault {
		if ids, err := r.Resolve(ctx, resource); err != nil {
			return Relationship{}, err
		} else {
			if ids == nil {
				ids = []ResourceId{}
			}
			var data any = ids
			return Relationship{Data: &data}, nil
		}
	}
	return Relationship{}, nil
}
