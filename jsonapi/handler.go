package jsonapi

import (
	"context"
	"mime"
	"net/http"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

func (api API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := api.executeRequest(r)
	resp.JSONAPI = &JSONAPI{
		Version: "1.1",
	}

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

func (api API) getResource(ctx context.Context, id ResourceId) (*Resource, *Error) {
	if resourceType, ok := api.Schema.resourceTypes[id.Type]; ok {
		return resourceType.get(ctx, id)
	}
	return nil, nil
}

func (api API) getResources(ctx context.Context, ids []ResourceId) ([]Resource, *Error) {
	var ret []Resource
	for _, id := range ids {
		if resourceType, ok := api.Schema.resourceTypes[id.Type]; ok {
			if resource, err := resourceType.get(ctx, id); err != nil {
				return nil, err
			} else if resource != nil {
				ret = append(ret, *resource)
			}
		}
	}
	return ret, nil
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

	q := r.URL.Query()

	// If an endpoint does not support the include parameter, it MUST respond with 400 Bad Request
	// to any requests that include it.
	//
	// If the server does not support sorting as specified in the query parameter sort, it MUST
	// return 400 Bad Request.
	//
	// If a server encounters a query parameter that does not follow the naming conventions defined
	// by section 10.3, Implementation-Specific Query Parameters, or the server does not know how to
	// process it as a query parameter from this specification, it MUST return 400 Bad Request.
	if len(q) > 0 {
		// We don't support any query parameters currently.
		return &ResponseDocument{
			Errors: []Error{errorForHTTPStatus(http.StatusBadRequest)},
		}
	}

	if r.Method == "GET" && len(pathComponents) >= 1 {
		typeName := pathComponents[0]
		if resourceType, ok := api.Schema.resourceTypes[typeName]; ok {
			if len(pathComponents) >= 2 {
				resourceId := ResourceId{
					Type: typeName,
					Id:   pathComponents[1],
				}

				if len(pathComponents) == 2 {
					// just return the resource
					if resource, err := resourceType.get(ctx, resourceId); err != nil {
						return &ResponseDocument{
							Errors: []Error{*err},
						}
					} else if resource != nil {
						var data any = resource
						return &ResponseDocument{
							Data: &data,
							Links: Links{
								"self": r.URL.Path,
							},
						}
					}
				} else if len(pathComponents) == 3 {
					// get a related resource
					relationshipName := pathComponents[2]
					if relationship, err := resourceType.getRelationship(ctx, resourceId, relationshipName); err != nil {
						return &ResponseDocument{
							Errors: []Error{*err},
						}
					} else if relationship != nil {
						var data any = nil
						var err *Error
						switch ids := relationship.Data.(type) {
						case ResourceId:
							data, err = api.getResource(ctx, ids)
						case []ResourceId:
							data, err = api.getResources(ctx, ids)
						}
						if err != nil {
							return &ResponseDocument{
								Errors: []Error{*err},
							}
						}
						return &ResponseDocument{
							Data: &data,
							Links: Links{
								"self": r.URL.Path,
							},
						}
					}
				} else if len(pathComponents) == 4 && pathComponents[2] == "relationships" {
					// get a relationship
					relationshipName := pathComponents[3]
					if relationship, err := resourceType.getRelationship(ctx, resourceId, relationshipName); err != nil {
						return &ResponseDocument{
							Errors: []Error{*err},
						}
					} else if relationship != nil {
						return &ResponseDocument{
							Data:  &relationship.Data,
							Links: relationship.Links,
						}
					}
				}
			}
		}
	}

	return &ResponseDocument{
		Errors: []Error{errorForHTTPStatus(http.StatusNotFound)},
	}
}
