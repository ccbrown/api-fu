package jsonapi

import (
	"context"
	"mime"
	"net/http"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"

	"github.com/ccbrown/api-fu/jsonapi/types"
)

func (api API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := api.executeRequest(r)
	resp.JSONAPI = &types.JSONAPI{
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

func errorForHTTPStatus(status int) types.Error {
	return types.Error{
		Status: strconv.Itoa(status),
		Title:  http.StatusText(status),
	}
}

func (api API) getResource(ctx context.Context, id types.ResourceId) (*types.Resource, *types.Error) {
	if resourceType, ok := api.Schema.resourceTypes[id.Type]; ok {
		return resourceType.get(ctx, id)
	}
	return nil, nil
}

func (api API) getResources(ctx context.Context, ids []types.ResourceId) ([]types.Resource, *types.Error) {
	var ret []types.Resource
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

func (api API) executeRequest(r *http.Request) *types.ResponseDocument {
	// If a request’s Accept header contains an instance of the JSON:API media type, servers MUST
	// ignore instances of that media type which are modified by a media type parameter other than
	// ext or profile. If all instances of that media type are modified with a media type parameter
	// other than ext or profile, servers MUST respond with a 406 Not Acceptable status code. If
	// every instance of that media type is modified by the ext parameter and each contains at least
	// one unsupported extension URI, the server MUST also respond with a 406 Not Acceptable.
	//
	// If the profile parameter is received, a server SHOULD attempt to apply any requested
	// profile(s) to its response. A server MUST ignore any profiles that it does not recognize.
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
		return &types.ResponseDocument{
			Errors: []types.Error{errorForHTTPStatus(http.StatusNotAcceptable)},
		}
	}

	ctx := r.Context()
	pathComponents := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")

	q := r.URL.Query()

	// Check for unsupported parameters.
	for k := range q {
		parts := strings.Split(k, "[")
		familyName := parts[0]

		for _, part := range parts[1:] {
			if len(part) < 1 || part[len(part)-1] != ']' || validateMemberName(part[:len(part)-1]) != nil {
				// This is not a valid query parameter.
				return &types.ResponseDocument{
					Errors: []types.Error{errorForHTTPStatus(http.StatusBadRequest)},
				}
			}
		}

		if validateMemberName(familyName) != nil {
			// This is either an extension parameter or an invalid family name. Either way, we don't
			// support it.
			return &types.ResponseDocument{
				Errors: []types.Error{errorForHTTPStatus(http.StatusBadRequest)},
			}
		}

		if strings.IndexFunc(familyName, func(r rune) bool {
			return !(r >= 'a' && r <= 'z')
		}) < 0 {
			// This is not an implementation-specific parameter, and if it's not one we support, we
			// must return a 400 error.
			switch familyName {
			case "page":
			default:
				return &types.ResponseDocument{
					Errors: []types.Error{errorForHTTPStatus(http.StatusBadRequest)},
				}
			}
		}
	}

	if len(pathComponents) >= 1 {
		typeName := pathComponents[0]
		if resourceType, ok := api.Schema.resourceTypes[typeName]; ok {
			if len(pathComponents) >= 2 {
				resourceId := types.ResourceId{
					Type: typeName,
					Id:   pathComponents[1],
				}

				if len(pathComponents) == 2 {
					// resource request
					switch r.Method {
					case "GET":
						if resource, err := resourceType.get(ctx, resourceId); err != nil {
							return &types.ResponseDocument{
								Errors: []types.Error{*err},
							}
						} else if resource != nil {
							var data any = resource
							return &types.ResponseDocument{
								Data: &data,
								Links: types.Links{
									"self": r.URL.Path,
								},
							}
						}
					case "PATCH":
						var patch types.PatchRequest
						if err := jsoniter.NewDecoder(r.Body).Decode(&patch); err != nil {
							return &types.ResponseDocument{
								Errors: []types.Error{errorForHTTPStatus(http.StatusBadRequest)},
							}
						}

						if patch.Data.Type != resourceId.Type || patch.Data.Id != resourceId.Id {
							// A server MUST return 409 Conflict when processing a PATCH request in
							// which the resource object’s type or id do not match the server’s
							// endpoint.
							return &types.ResponseDocument{
								Errors: []types.Error{errorForHTTPStatus(http.StatusConflict)},
							}
						}

						relationships := make(map[string]any, len(patch.Data.Relationships))
						for k, v := range patch.Data.Relationships {
							relationships[k] = v.Data
						}

						if resource, err := resourceType.patch(ctx, resourceId, patch.Data.Attributes, relationships); err != nil {
							return &types.ResponseDocument{
								Errors: []types.Error{*err},
							}
						} else if resource != nil {
							var data any = resource
							return &types.ResponseDocument{
								Data: &data,
								Links: types.Links{
									"self": r.URL.Path,
								},
							}
						}
					case "DELETE":
						if err := resourceType.delete(ctx, resourceId); err != nil {
							return &types.ResponseDocument{
								Errors: []types.Error{*err},
							}
						}

						return &types.ResponseDocument{}
					default:
						return &types.ResponseDocument{
							Errors: []types.Error{errorForHTTPStatus(http.StatusMethodNotAllowed)},
						}
					}
				} else if len(pathComponents) == 3 {
					// related resource request
					switch r.Method {
					case "GET":
						relationshipName := pathComponents[2]
						if relationship, err := resourceType.getRelationship(ctx, resourceId, relationshipName, q); err != nil {
							return &types.ResponseDocument{
								Errors: []types.Error{*err},
							}
						} else if relationship != nil {
							var data any = nil
							var err *types.Error
							switch ids := (*relationship.Data).(type) {
							case types.ResourceId:
								data, err = api.getResource(ctx, ids)
							case []types.ResourceId:
								data, err = api.getResources(ctx, ids)
							}
							if err != nil {
								return &types.ResponseDocument{
									Errors: []types.Error{*err},
								}
							}
							return &types.ResponseDocument{
								Data: &data,
								Links: types.Links{
									"self": r.URL.Path,
								},
							}
						}
					case "PATCH":
						relationshipName := pathComponents[2]
						if relationship, err := resourceType.getRelationship(ctx, resourceId, relationshipName, q); err != nil {
							return &types.ResponseDocument{
								Errors: []types.Error{*err},
							}
						} else if relationship != nil {
							if relatedId, ok := (*relationship.Data).(types.ResourceId); ok {
								if relatedResourceType, ok := api.Schema.resourceTypes[relatedId.Type]; ok {
									var patch types.PatchRequest
									if err := jsoniter.NewDecoder(r.Body).Decode(&patch); err != nil {
										return &types.ResponseDocument{
											Errors: []types.Error{errorForHTTPStatus(http.StatusBadRequest)},
										}
									}

									if patch.Data.Type != relatedId.Type || patch.Data.Id != relatedId.Id {
										// A server MUST return 409 Conflict when processing a PATCH request in
										// which the resource object’s type or id do not match the server’s
										// endpoint.
										return &types.ResponseDocument{
											Errors: []types.Error{errorForHTTPStatus(http.StatusConflict)},
										}
									}

									relationships := make(map[string]any, len(patch.Data.Relationships))
									for k, v := range patch.Data.Relationships {
										relationships[k] = v.Data
									}

									if resource, err := relatedResourceType.patch(ctx, relatedId, patch.Data.Attributes, relationships); err != nil {
										return &types.ResponseDocument{
											Errors: []types.Error{*err},
										}
									} else if resource != nil {
										var data any = resource
										return &types.ResponseDocument{
											Data: &data,
											Links: types.Links{
												"self": r.URL.Path,
											},
										}
									}
								}
							}
						}
					default:
						return &types.ResponseDocument{
							Errors: []types.Error{errorForHTTPStatus(http.StatusMethodNotAllowed)},
						}
					}
				} else if len(pathComponents) == 4 && pathComponents[2] == "relationships" {
					// relationship request
					relationshipName := pathComponents[3]
					switch r.Method {
					case "GET":
						if relationship, err := resourceType.getRelationship(ctx, resourceId, relationshipName, q); err != nil {
							return &types.ResponseDocument{
								Errors: []types.Error{*err},
							}
						} else if relationship != nil {
							return &types.ResponseDocument{
								Data:  relationship.Data,
								Links: relationship.Links,
							}
						}
					case "PATCH":
						var patch types.PatchRequestDataRelationship
						if err := jsoniter.NewDecoder(r.Body).Decode(&patch); err != nil {
							return &types.ResponseDocument{
								Errors: []types.Error{errorForHTTPStatus(http.StatusBadRequest)},
							}
						} else if relationship, err := resourceType.patchRelationship(ctx, resourceId, relationshipName, patch.Data); err != nil {
							return &types.ResponseDocument{
								Errors: []types.Error{*err},
							}
						} else if relationship != nil {
							return &types.ResponseDocument{
								Data:  relationship.Data,
								Links: relationship.Links,
							}
						}
					default:
						return &types.ResponseDocument{
							Errors: []types.Error{errorForHTTPStatus(http.StatusMethodNotAllowed)},
						}
					}
				}
			}
		}
	}

	return &types.ResponseDocument{
		Errors: []types.Error{errorForHTTPStatus(http.StatusNotFound)},
	}
}
