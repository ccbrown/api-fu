package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser(t *testing.T) {
	api := NewTestAPI()

	resp := api.execGraphQL(t, `
		mutation {
			createUser(user: {
				handle: "zerocool",
				password: "password",
			}) {
				user {
					id
				}
			}
		}
	`, nil)
	assert.Empty(t, resp.Errors)

	id := (*resp.Data).(map[string]interface{})["createUser"].(map[string]interface{})["user"].(map[string]interface{})["id"].(string)

	resp = api.execGraphQL(t, `
		query User($id: ID!) {
			node(id: $id) {
				... on User {
					handle
				}
			}
		}
	`, map[string]interface{}{
		"id": id,
	})
	assert.Empty(t, resp.Errors)
	assert.Equal(t, map[string]interface{}{
		"node": map[string]interface{}{
			"handle": "zerocool",
		},
	}, *resp.Data)
}
