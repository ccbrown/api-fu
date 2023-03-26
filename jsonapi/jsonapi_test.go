package jsonapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateMemberName(t *testing.T) {
	for name, tc := range map[string]struct {
		In   string
		Okay bool
	}{
		"Lowercase": {
			In:   "foo",
			Okay: true,
		},
		"Mixed": {
			In:   "fooBar12",
			Okay: true,
		},
		"Hyphens": {
			In:   "foo-Bar12",
			Okay: true,
		},
		"HyphenAtStart": {
			In:   "-foo",
			Okay: false,
		},
		"HyphenAtEnd": {
			In:   "foo-",
			Okay: false,
		},
		"Empty": {
			In:   "",
			Okay: false,
		},
		"IllegalCharacter": {
			In:   "foo!Bar",
			Okay: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := validateMemberName(tc.In)
			if tc.Okay {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
