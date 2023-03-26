package jsonapi

// An object used to represent links.
//
// Within this object, a link MUST be represented as either:
//
// - a string whose value is a URI-reference [RFC3986 Section 4.1] pointing to the link’s target,
// - a link object or
// - null if the link does not exist.
type LinksObject map[string]string

// A “link object” is an object that represents a web link.
type LinkObject struct {
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
