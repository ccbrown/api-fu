package apifu

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql"
)

var testCfg = Config{}

var asyncChannel = make(chan struct{})

func init() {
	// If this is not executed asynchronously alongside a matching asyncReceiver, it will deadlock.
	testCfg.AddQueryField("asyncSender", &graphql.FieldDefinition{
		Type: graphql.BooleanType,
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			return Go(ctx.Context, func() (interface{}, error) {
				asyncChannel <- struct{}{}
				return true, nil
			}), nil
		},
	})

	// If this is not executed asynchronously alongside a matching asyncSender, it will deadlock.
	testCfg.AddQueryField("asyncReceiver", &graphql.FieldDefinition{
		Type: graphql.BooleanType,
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			return Go(ctx.Context, func() (interface{}, error) {
				<-asyncChannel
				return true, nil
			}), nil
		},
	})
}

func executeGraphQL(t *testing.T, api *API, query string) *http.Response {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", "", strings.NewReader(query))
	r.Header.Set("Content-Type", "application/graphql")
	require.NoError(t, err)
	api.ServeGraphQL(w, r)
	return w.Result()
}

func TestGo(t *testing.T) {
	api, err := NewAPI(&testCfg)
	require.NoError(t, err)

	resp := executeGraphQL(t, api, `{
		s: asyncSender
		r: asyncReceiver
	}`)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":{"s":true,"r":true}}`, string(body))
}
