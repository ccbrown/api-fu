package graphql

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ccbrown/api-fu/graphql/executor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequestFromHTTP(t *testing.T) {
	for name, tc := range map[string]struct {
		Method       string
		Query        url.Values
		ContentType  string
		Body         string
		ExpectedCode int
	}{
		"GET": {
			Method: "GET",
			Query: url.Values{
				"query":     []string{"{__typename}"},
				"variables": []string{`{"foo":"bar"}`},
			},
			ExpectedCode: http.StatusOK,
		},
		"GETNoQuery": {
			Method: "GET",
			Query: url.Values{
				"variables": []string{`{"foo":"bar"}`},
			},
			ExpectedCode: http.StatusBadRequest,
		},
		"GETBadVariables": {
			Method: "GET",
			Query: url.Values{
				"query":     []string{"{__typename}"},
				"variables": []string{`foo`},
			},
			ExpectedCode: http.StatusBadRequest,
		},
		"POSTGraphQL": {
			Method:       "POST",
			ContentType:  "application/graphql",
			Body:         `{__typename}`,
			ExpectedCode: http.StatusOK,
		},
		"POSTJSON": {
			Method:      "POST",
			ContentType: "application/json",
			Query: url.Values{
				"query": []string{"{__typename}"},
			},
			Body:         `{}`,
			ExpectedCode: http.StatusOK,
		},
		"POSTBadJSON": {
			Method:       "POST",
			ContentType:  "application/json",
			Body:         `asd}`,
			ExpectedCode: http.StatusBadRequest,
		},
		"POSTBadContentType": {
			Method:       "POST",
			ContentType:  "application/foo",
			Body:         `{}`,
			ExpectedCode: http.StatusBadRequest,
		},
		"PUT": {
			Method:       "PUT",
			ExpectedCode: http.StatusMethodNotAllowed,
		},
	} {
		t.Run(name, func(t *testing.T) {
			var body io.Reader
			if tc.Body != "" {
				body = strings.NewReader(tc.Body)
			}
			httpReq, err := http.NewRequest(tc.Method, "/?"+tc.Query.Encode(), body)
			require.NoError(t, err)
			if tc.ContentType != "" {
				httpReq.Header.Set("Content-Type", tc.ContentType)
			}
			req, code, err := NewRequestFromHTTP(httpReq)
			assert.Equal(t, tc.ExpectedCode, code)
			if tc.ExpectedCode == http.StatusOK {
				assert.NotNil(t, req)
				assert.NoError(t, err)
			} else {
				assert.Nil(t, req)
				assert.Error(t, err)
			}
		})
	}
}

func TestNewErrorFromExecutorError(t *testing.T) {
	assert.Equal(t, &Error{
		Message: "message",
		Locations: []Location{
			{
				Line:   1,
				Column: 2,
			},
		},
	}, newErrorFromExecutorError(&executor.Error{
		Message: "message",
		Locations: []executor.Location{
			{
				Line:   1,
				Column: 2,
			},
		},
	}))
}
