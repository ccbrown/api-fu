package jsonapi

// Error objects provide additional information about problems encountered while performing an
// operation.
type Error struct {
	// A unique identifier for this particular occurrence of the problem.
	Id string `json:"id,omitempty"`

	Links LinksObject `json:"links,omitempty"`

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
