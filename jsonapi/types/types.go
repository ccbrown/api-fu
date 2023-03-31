package types

import (
	"encoding/json"

	jsoniter "github.com/json-iterator/go"
)

// This object defines a document’s “top level”.
type ResponseDocument struct {
	// The document’s “primary data”.
	Data *any `json:"data,omitempty"`

	// An array of error objects.
	Errors []Error `json:"errors,omitempty"`

	// A meta object containing non-standard meta-information.
	Meta map[string]any `json:"meta,omitempty"`

	// An object describing the server’s implementation.
	JSONAPI *JSONAPI `json:"jsonapi,omitempty"`

	// A links object related to the primary data.
	Links Links `json:"links,omitempty"`
}

type JSONAPI struct {
	// A string indicating the highest JSON:API version supported.
	Version string `json:"version,omitempty"`

	// An array of URIs for all applied extensions.
	Ext []string `json:"ext,omitempty"`

	// An array of URIs for all applied profiles.
	Profile []string `json:"profile,omitempty"`

	// A meta object containing non-standard meta-information.
	Meta map[string]any `json:"meta,omitempty"`
}

// Error objects provide additional information about problems encountered while performing an
// operation.
type Error struct {
	// A unique identifier for this particular occurrence of the problem.
	Id string `json:"id,omitempty"`

	Links Links `json:"links,omitempty"`

	// The HTTP status code applicable to this problem, expressed as a string value.
	Status string `json:"status,omitempty"`

	// An application-specific error code, expressed as a string value.
	Code string `json:"code,omitempty"`

	// A short, human-readable summary of the problem that SHOULD NOT change from occurrence to
	// occurrence of the problem, except for purposes of localization.
	Title string `json:"title,omitempty"`

	// A human-readable explanation specific to this occurrence of the problem. Like title, this
	// field’s value can be localized.
	Detail string `json:"detail,omitempty"`

	// An object containing references to the primary source of the error.
	Source *ErrorSource `json:"source,omitempty"`

	// A meta object containing non-standard meta-information about the error.
	Meta map[string]any `json:"meta,omitempty"`
}

// An object containing references to the primary source of the error.
type ErrorSource struct {
	// A JSON Pointer [RFC6901] to the value in the request document that caused the error [e.g.
	// "/data" for a primary data object, or "/data/attributes/title" for a specific attribute].
	Pointer string `json:"pointer,omitempty"`

	// A string indicating which URI query parameter caused the error.
	Parameter string `json:"parameter,omitempty"`

	// A meta object containing non-standard meta-information about the error.
	Header string `json:"header,omitempty"`
}

// An object used to represent links.
//
// Within this object, a link MUST be represented as either:
//
// - a string whose value is a URI-reference [RFC3986 Section 4.1] pointing to the link’s target,
// - a link object or
// - null if the link does not exist.
type Links map[string]string

// A “link object” is an object that represents a web link.
type Link struct {
	// A string whose value is a URI-reference [RFC3986 Section 4.1] pointing to the link’s target.
	HREF string `json:"href"`

	// A string indicating the link’s relation type. The string MUST be a valid link relation type.
	RelationType string `json:"rel,omitempty"`

	// A link to a description document (e.g. OpenAPI or JSON Schema) for the link target.
	DescribedBy string `json:"describedBy,omitempty"`

	// A string which serves as a label for the destination of a link such that it can be used as a
	// human-readable identifier (e.g., a menu entry).
	Title string `json:"title,omitempty"`

	// A string indicating the media type of the link’s target.
	Type string `json:"type,omitempty"`

	// A string or an array of strings indicating the language(s) of the link’s target. An array of
	// strings indicates that the link’s target is available in multiple languages. Each string MUST
	// be a valid language tag [RFC5646].
	HREFLanguage any `json:"hreflang,omitempty"`

	// A meta object containing non-standard meta-information about the link.
	Meta map[string]any `json:"meta,omitempty"`
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
	// If given, this must be `nil`, `ResourceId`, or `[]ResourceId`.
	Data *any `json:"data,omitempty"`

	// A meta object containing non-standard meta-information about the relationship.
	Meta map[string]any `json:"meta,omitempty"`
}

type ResourceId struct {
	Type string `json:"type"`

	Id string `json:"id"`
}

type PatchRequest struct {
	// The document’s “primary data”.
	Data PatchRequestData `json:"data"`
}

type PatchRequestData struct {
	Type string `json:"type"`

	Id string `json:"id"`

	// An object containing the attributes to be updated.
	Attributes map[string]json.RawMessage `json:"attributes,omitempty"`

	// An object containing the relationships to be updated.
	Relationships map[string]PatchRequestDataRelationship `json:"relationships,omitempty"`
}

type PatchRequestDataRelationship struct {
	// Either nil, `ResourceId`, or `[]ResourceId`.
	Data any `json:"data"`
}

func (r *PatchRequestDataRelationship) UnmarshalJSON(buf []byte) error {
	var tmp struct {
		Data json.RawMessage `json:"data"`
	}
	if err := jsoniter.Unmarshal(buf, &tmp); err != nil {
		return err
	}

	if len(tmp.Data) > 0 {
		if tmp.Data[0] == '[' {
			var data []ResourceId
			if err := jsoniter.Unmarshal(tmp.Data, &data); err != nil {
				return err
			}
			r.Data = data
		} else {
			var data *ResourceId
			if err := jsoniter.Unmarshal(tmp.Data, &data); err != nil {
				return err
			}
			if data != nil {
				r.Data = *data
			}
		}
	}

	return nil
}
