package apifu

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/ccbrown/api-fu/graphql"
)

// PersistedQueryStorage represents the storage backend for persisted queries. Storage operations
// are done on a best effort basis and cannot return errors – any errors that happen internally will
// not prevent the execution of a query (though it might force clients to make additional requests).
type PersistedQueryStorage interface {
	// GetPersistedQuery should return the query if it's available or an empty string otherwise.
	GetPersistedQuery(ctx context.Context, hash []byte) string

	// PersistQuery should persist the query with the given hash.
	PersistQuery(ctx context.Context, query string, hash []byte)
}

var emptyStringHash = sha256.Sum256([]byte(""))

// PersistedQueryExtension implements Apollo persisted queries:
// https://www.apollographql.com/docs/react/api/link/persisted-queries/
//
// Typically this shouldn't be invoked directly. Instead, set the PersistedQueryStorage Config
// field.
func PersistedQueryExtension(storage PersistedQueryStorage, execute func(*graphql.Request) *graphql.Response) func(*graphql.Request) *graphql.Response {
	return func(input *graphql.Request) *graphql.Response {
		r := *input
		ext, _ := r.Extensions["persistedQuery"].(map[string]interface{})
		switch ext["version"] {
		case 1, 1.0:
			if r.Query == "" && r.Document == nil {
				// errors parsing the hash can be ignored: hash will end up empty and we'll error
				// out due to not being able to find the query
				hashHex, _ := ext["sha256Hash"].(string)
				hash, _ := hex.DecodeString(hashHex)

				found := false
				if bytes.Equal(hash, emptyStringHash[:]) {
					// i'm not really sure why anyone would do this, but we'll consider the query
					// found and let the executor error out
					found = true
				} else if len(hash) == sha256.Size {
					if query := storage.GetPersistedQuery(r.Context, hash); query != "" {
						r.Query = query
						found = true
					}
				}
				if !found {
					return &graphql.Response{
						Errors: []*graphql.Error{
							{
								Message: "PersistedQueryNotFound",
							},
						},
					}
				}
			} else if r.Query != "" {
				hash := sha256.Sum256([]byte(r.Query))
				storage.PersistQuery(r.Context, r.Query, hash[:])
			}
		}
		return execute(&r)
	}
}
