package api

import (
	"context"
	"reflect"
	"strings"
	"time"

	apifu "github.com/ccbrown/api-fu"
	"github.com/ccbrown/api-fu/examples/chat/model"
	"github.com/ccbrown/api-fu/graphql"
)

var channelType = fuCfg.AddNodeType(&apifu.NodeType{
	Id:    2,
	Name:  "Channel",
	Model: reflect.TypeOf(model.Channel{}),
	GetByIds: func(ctx context.Context, ids interface{}) (interface{}, error) {
		return ctxSession(ctx).GetChannelsByIds(ids.([]model.Id)...)
	},
})

func init() {
	type messageCursor struct {
		Nano int64
		Id   model.Id
	}

	channelType.Fields = map[string]*graphql.FieldDefinition{
		"id":           apifu.NonNullNodeID("Id"),
		"name":         apifu.NonNullString("Name"),
		"creationTime": apifu.NonNullDateTime("CreationTime"),
		"creator":      apifu.Node(userType, "CreatorUserId"),
		"messagesConnection": apifu.Connection(&apifu.ConnectionConfig{
			NamePrefix:  "ChannelMessages",
			Description: "Provides messages sorted by time.",
			EdgeCursor: func(edge interface{}) interface{} {
				message := edge.(*model.Message)
				return messageCursor{
					Nano: message.Time.UnixNano(),
					Id:   message.Id,
				}
			},
			EdgeFields: map[string]*graphql.FieldDefinition{
				"node": &graphql.FieldDefinition{
					Type: graphql.NewNonNullType(messageType),
					Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
						return ctx.Object, nil
					},
				},
			},
			CursorType: reflect.TypeOf(messageCursor{}),
			// On an active server, it would get prohibitively expensive to fetch all messages each
			// time the connection is selected. So we use ResolveEdges instead of ResolveAllEdges.
			//
			// The business layer of our example application exposes a simple time-based range
			// getter. So to ensure that our results are complete, even in the unlikely event that
			// there are many messages with the exact same timestamp, we may have to do up to 3
			// queries.
			ResolveEdges: func(ctx *graphql.FieldContext, after, before interface{}, limit int) (edgeSlice interface{}, cursorLess func(a, b interface{}) bool, err error) {
				channel := ctx.Object.(*model.Channel)
				session := ctxSession(ctx.Context)

				type Query struct {
					Start time.Time
					End   time.Time
					Limit int
				}
				var queries []Query

				middle := Query{channel.CreationTime, time.Now().Add(time.Hour), limit}

				if after, ok := after.(messageCursor); ok {
					queries = append(queries, Query{time.Unix(0, after.Nano), time.Unix(0, after.Nano), 0})
					middle.Start = time.Unix(0, after.Nano+1)
				}

				if before, ok := before.(messageCursor); ok {
					if after, ok := after.(messageCursor); !ok || after.Nano != before.Nano {
						queries = append(queries, Query{time.Unix(0, before.Nano), time.Unix(0, before.Nano), 0})
					}
					middle.End = time.Unix(0, before.Nano-1)
				}

				queries = append(queries, middle)

				var messages []*model.Message
				for _, q := range queries {
					if msgs, err := session.GetMessagesByChannelIdAndTimeRange(channel.Id, q.Start, q.End, q.Limit); err != nil {
						return nil, nil, err
					} else {
						messages = append(messages, msgs...)
					}
				}

				return messages, func(a, b interface{}) bool {
					ac, bc := a.(messageCursor), b.(messageCursor)
					return ac.Nano < bc.Nano || (ac.Nano == bc.Nano && ac.Id.Before(bc.Id))
				}, err
			},
		}),
	}
}

func init() {
	fuCfg.AddMutation("createChannel", &graphql.FieldDefinition{
		Type: &graphql.ObjectType{
			Name: "CreateChannelResult",
			Fields: map[string]*graphql.FieldDefinition{
				"channel": &graphql.FieldDefinition{
					Type: graphql.NewNonNullType(channelType),
					Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
						return ctx.Object, nil
					},
				},
			},
		},
		Arguments: map[string]*graphql.InputValueDefinition{
			"channel": &graphql.InputValueDefinition{
				Type: graphql.NewNonNullType(&graphql.InputObjectType{
					Name: "ChannelInput",
					Fields: map[string]*graphql.InputValueDefinition{
						"name": &graphql.InputValueDefinition{
							Type: graphql.NewNonNullType(graphql.StringType),
						},
					},
					InputCoercion: func(input map[string]interface{}) (interface{}, error) {
						return &model.Channel{
							Name: input["name"].(string),
						}, nil
					},
				}),
			},
		},
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			return ctxSession(ctx.Context).CreateChannel(ctx.Arguments["channel"].(*model.Channel))
		},
	})
}

func init() {
	type cursor struct {
		Name string
		Id   model.Id
	}

	fuCfg.AddQueryField("channelsConnection", apifu.Connection(&apifu.ConnectionConfig{
		NamePrefix:  "QueryChannels",
		Description: "Provides channels sorted by name.",
		EdgeCursor: func(edge interface{}) interface{} {
			channel := edge.(*model.Channel)
			return cursor{
				Name: strings.ToLower(channel.Name),
				Id:   channel.Id,
			}
		},
		EdgeFields: map[string]*graphql.FieldDefinition{
			"node": &graphql.FieldDefinition{
				Type: graphql.NewNonNullType(channelType),
				Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
					return ctx.Object, nil
				},
			},
		},
		CursorType: reflect.TypeOf(cursor{}),
		// If we assume the server will always have a relatively small number of channels, we can
		// keep things simple using ResolveAllEdges.
		ResolveAllEdges: func(ctx *graphql.FieldContext) (interface{}, func(a, b interface{}) bool, error) {
			channels, err := ctxSession(ctx.Context).GetChannels()
			return channels, func(a, b interface{}) bool {
				ac, bc := a.(cursor), b.(cursor)
				return ac.Name < bc.Name || (ac.Name == bc.Name && ac.Id.Before(bc.Id))
			}, err
		},
	}))
}
