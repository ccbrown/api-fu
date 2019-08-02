package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVariables_AllUsed(t *testing.T) {
	assert.Empty(t, validateSource(t, `query ($id: ID!) {node(id: $id){id}}`))
	assert.Len(t, validateSource(t, `query ($id: ID!) {scalar}`), 1)
	assert.Empty(t, validateSource(t, `query ($id: ID!) {...obj} fragment obj on Object {node(id: $id){id}}`))
	assert.Len(t, validateSource(t, `query ($id: ID!) {...obj} fragment obj on Object {scalar}`), 1)
}

func TestVariables_AllUsesDefines(t *testing.T) {
	assert.Empty(t, validateSource(t, `query ($id: ID!) {node(id: $id){id}}`))
	assert.Len(t, validateSource(t, `query {node(id: $id){id}}`), 1)
	assert.Empty(t, validateSource(t, `query ($id: ID!) {...obj} fragment obj on Object {node(id: $id){id}}`))
	assert.Len(t, validateSource(t, `query {...obj} fragment obj on Object {node(id: $id){id}}`), 1)
}

func TestVariables_InputTypes(t *testing.T) {
	assert.Empty(t, validateSource(t, `query ($id: ID!) {node(id: $id){id}}`))
	assert.Len(t, validateSource(t, `query ($id: Object!) {node(id: $id){id}}`), 2)
}

func TestVariables_NameUniqueness(t *testing.T) {
	assert.Empty(t, validateSource(t, `query ($id: ID!) {node(id: $id){id}}`))
	assert.Len(t, validateSource(t, `query ($id: ID!, $id: ID!) {node(id: $id){id}}`), 1)
}

func TestVariables_UsagesAllowed(t *testing.T) {
	assert.Empty(t, validateSource(t, `query ($id: ID!) {node(id: $id){id}}`))
	assert.Len(t, validateSource(t, `query ($id: String!) {node(id: $id){id}}`), 1)
	assert.Empty(t, validateSource(t, `query ($id: ID = "") {node(id: $id){id}}`))
	assert.Len(t, validateSource(t, `query ($id: ID) {node(id: $id){id}}`), 1)
	assert.Empty(t, validateSource(t, `query ($s: String) {object(object:{defaultedString: $s, requiredString: ""}){scalar}}`))
	assert.Len(t, validateSource(t, `query ($s: String) {object(object:{requiredString: $s}){scalar}}`), 1)
}
