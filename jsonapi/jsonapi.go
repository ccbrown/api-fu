package jsonapi

import (
	"fmt"
	"strings"
)

type API struct {
	Schema *Schema
}

// This object defines a document’s “top level”.
type ResponseDocument struct {
	// The document’s “primary data”.
	Data *any `json:"data,omitempty"`

	// An array of error objects.
	Errors []Error `json:"errors,omitempty"`

	// A meta object containing non-standard meta-information.
	Meta map[string]any `json:"meta,omitempty"`

	// An object describing the server’s implementation.
	Implementation map[string]any `json:"jsonapi,omitempty"`

	// A links object related to the primary data.
	Links Links `json:"links,omitempty"`
}

func isGloballyAllowedCharacter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func isInternallyAllowedCharacter(r rune) bool {
	return isGloballyAllowedCharacter(r) || r == '-' || r == '_'
}

// https://jsonapi.org/format/#document-member-names
func validateMemberName(name string) error {
	if len(name) < 1 {
		return fmt.Errorf("member names must have at least one character")
	} else if strings.IndexFunc(name, func(r rune) bool {
		return !isInternallyAllowedCharacter(r)
	}) >= 0 {
		return fmt.Errorf("member names may only contain numbers, letters, hyphens, and underscores")
	} else if !isGloballyAllowedCharacter(rune(name[0])) || !isGloballyAllowedCharacter(rune(name[len(name)-1])) {
		return fmt.Errorf("member names must begin and end with a number or letter")
	}
	return nil
}
