package jsonapi

import (
	"mime"
	"net/http"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

func (api API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := api.executeRequest(r)

	w.Header().Set("Content-Type", "application/vnd.api+json")

	status := http.StatusOK
	if len(resp.Errors) > 0 {
		status = http.StatusInternalServerError
		for _, err := range resp.Errors {
			if err.Status != "" {
				n, _ := strconv.ParseInt(err.Status, 10, 0)
				status = int(n)
				break
			}
		}
	}

	body, err := jsoniter.Marshal(resp)
	if err != nil {
		status = http.StatusInternalServerError
		newErr := errorForHTTPStatus(status)
		newErr.Detail = err.Error()
		body, _ = jsoniter.Marshal(newErr)
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	w.Write(body)
}

func errorForHTTPStatus(status int) Error {
	return Error{
		Status: strconv.Itoa(status),
		Title:  http.StatusText(status),
	}
}

func (api API) executeRequest(r *http.Request) *ResponseDocument {
	/*
		If a request’s Accept header contains an instance of the JSON:API media type, servers MUST
		ignore instances of that media type which are modified by a media type parameter other than ext
		or profile. If all instances of that media type are modified with a media type parameter other
		than ext or profile, servers MUST respond with a 406 Not Acceptable status code. If every
		instance of that media type is modified by the ext parameter and each contains at least one
		unsupported extension URI, the server MUST also respond with a 406 Not Acceptable.

		If the profile parameter is received, a server SHOULD attempt to apply any requested profile(s)
		to its response. A server MUST ignore any profiles that it does not recognize.
	*/
	isAcceptable := false
	for _, accept := range r.Header.Values("Accept") {
		mediaType, params, err := mime.ParseMediaType(accept)
		if mediaType != "application/vnd.api+json" || err != nil {
			continue
		}
		hasUnsupportedParams := false
		for k := range params {
			if k != "profile" {
				// TODO: support extensions?
				hasUnsupportedParams = true
				break
			}
		}
		if hasUnsupportedParams {
			continue
		}
		isAcceptable = true
		break
	}
	if !isAcceptable {
		return &ResponseDocument{
			Errors: []Error{errorForHTTPStatus(http.StatusNotAcceptable)},
		}
	}

	ctx := r.Context()
	pathComponents := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")

	linkURL := func(path string) string {
		return r.URL.Scheme + "://" + r.URL.Host + path
	}

	// handle requests for resources based on id
	if len(pathComponents) == 2 {
		typeName := pathComponents[0]
		resourceId := pathComponents[1]
		if resourceType, ok := api.Schema.resourceTypes[typeName]; ok {
			if resource, err := resourceType.get(ctx, typeName, resourceId); err != nil {
				return &ResponseDocument{
					Errors: []Error{*err},
				}
			} else if resource != nil {
				var data any = resource
				return &ResponseDocument{
					Data: &data,
					Links: LinksObject{
						"self": linkURL(r.URL.Path),
					},
				}
			}
		}
	}

	return &ResponseDocument{
		Errors: []Error{errorForHTTPStatus(http.StatusNotFound)},
	}
}
