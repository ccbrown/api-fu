package apifu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/ccbrown/api-fu/graphql"

	"github.com/stretchr/testify/assert"
)

type persistedQueryMap map[string]string

func (m persistedQueryMap) GetPersistedQuery(ctx context.Context, hash []byte) string {
	return m[string(hash)]
}

func (m persistedQueryMap) PersistQuery(ctx context.Context, query string, hash []byte) {
	m[string(hash)] = query
}

func TestPersistedQueryExtension(t *testing.T) {
	storage := persistedQueryMap{}
	success := &graphql.Response{}
	query := `{ __typename }`
	queryHash := sha256.Sum256([]byte(query))
	queryHashHex := hex.EncodeToString(queryHash[:])
	execute := PersistedQueryExtension(storage, func(r *graphql.Request) *graphql.Response {
		assert.Equal(t, query, r.Query)
		return success
	})

	assert.Equal(t, &graphql.Response{
		Errors: []*graphql.Error{
			{
				Message: "PersistedQueryNotFound",
			},
		},
	}, execute(&graphql.Request{
		Extensions: map[string]interface{}{
			"persistedQuery": map[string]interface{}{
				"version":    1,
				"sha256Hash": queryHashHex,
			},
		},
	}))

	assert.Equal(t, success, execute(&graphql.Request{
		Query: query,
		Extensions: map[string]interface{}{
			"persistedQuery": map[string]interface{}{
				"version":    1,
				"sha256Hash": queryHashHex,
			},
		},
	}))

	assert.Equal(t, success, execute(&graphql.Request{
		Extensions: map[string]interface{}{
			"persistedQuery": map[string]interface{}{
				"version":    1,
				"sha256Hash": queryHashHex,
			},
		},
	}))
}
