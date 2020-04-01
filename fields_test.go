package apifu

import (
	"context"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	"github.com/ccbrown/api-fu/graphql"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFields(t *testing.T) {
	const nodeTypeId = 10

	testCfg := Config{
		SerializeNodeId: func(typeId int, id interface{}) string {
			assert.Equal(t, nodeTypeId, typeId)
			return id.(string)
		},
		DeserializeNodeId: func(id string) (int, interface{}) {
			return nodeTypeId, id
		},
	}

	type node struct {
		Id string
	}

	nodeType := testCfg.AddNodeType(&NodeType{
		Id:    nodeTypeId,
		Name:  "TestNode",
		Model: reflect.TypeOf(node{}),
		GetByIds: func(ctx context.Context, ids interface{}) (interface{}, error) {
			var ret []*node
			for _, id := range ids.([]string) {
				if id == "a" || id == "b" {
					ret = append(ret, &node{
						Id: id,
					})
				}
			}
			return ret, nil
		},
		Fields: map[string]*graphql.FieldDefinition{
			"id": OwnID("Id"),
		},
	})

	// If this is not executed asynchronously alongside a matching asyncReceiver, it will deadlock.
	testCfg.AddQueryField("obj", &graphql.FieldDefinition{
		Type: &graphql.ObjectType{
			Name: "Object",
			Fields: map[string]*graphql.FieldDefinition{
				"int": NonNull(graphql.IntType, "Int"),
				"s0":  NonEmptyString("S0"),
				"s1":  NonEmptyString("S1"),
				"n":   Node(nodeType, "NodeId"),
				"nid": NonNullNodeID(reflect.TypeOf(node{}), "NodeId"),
			},
		},
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			return struct {
				Int    int
				S0     string
				S1     string
				NodeId string
			}{
				S1:     "foo",
				NodeId: "foo",
			}, nil
		},
	})

	api, err := NewAPI(&testCfg)
	require.NoError(t, err)

	resp := executeGraphQL(t, api, `{
		obj {
			int
			s0
			s1
			n {
				id
			}
			nid
		}
	}`)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":{"obj":{"int":0,"s0":null,"s1":"foo","n":null,"nid":"foo"}}}`, string(body))
}
