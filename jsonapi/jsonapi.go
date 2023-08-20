// This is a package for creating [JSON:API](https://jsonapi.org) APIs. It is currently still
// experimental and not subject to any compatibility guarantees.
package jsonapi

import (
	"fmt"
	"strings"
)

type API struct {
	Schema *Schema
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
