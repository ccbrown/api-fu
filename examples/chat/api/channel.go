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
		"id":           apifu.OwnID("Id"),
		"name":         apifu.NonNull(graphql.StringType, "Name"),
		"creationTime": apifu.NonNull(apifu.DateTimeType, "CreationTime"),
		"creator":      apifu.Node(userType, "CreatorUserId"),
		"messagesConnection": apifu.TimeBasedConnection(&apifu.TimeBasedConnectionConfig{
			NamePrefix: "ChannelMessages",
			EdgeCursor: func(edge interface{}) apifu.TimeBasedCursor {
				message := edge.(*model.Message)
				return apifu.NewTimeBasedCursor(message.Time, string(message.Id))
			},
			EdgeFields: map[string]*graphql.FieldDefinition{
				"node": {
					Type: graphql.NewNonNullType(messageType),
					Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
						return ctx.Object, nil
					},
				},
			},
			EdgeGetter: func(ctx graphql.FieldContext, minTime time.Time, maxTime time.Time, limit int) (interface{}, error) {
				return ctxSession(ctx.Context).GetMessagesByChannelIdAndTimeRange(ctx.Object.(*model.Channel).Id, minTime, maxTime, limit)
			},
		}),
	}
}

func init() {
	fuCfg.AddMutation("createChannel", &graphql.FieldDefinition{
		Type: &graphql.ObjectType{
			Name: "CreateChannelResult",
			Fields: map[string]*graphql.FieldDefinition{
				"channel": {
					Type: graphql.NewNonNullType(channelType),
					Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
						return ctx.Object, nil
					},
				},
			},
		},
		Arguments: map[string]*graphql.InputValueDefinition{
			"channel": {
				Type: graphql.NewNonNullType(&graphql.InputObjectType{
					Name: "ChannelInput",
					Fields: map[string]*graphql.InputValueDefinition{
						"name": {
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
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
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
			"node": {
				Type: graphql.NewNonNullType(channelType),
				Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
					return ctx.Object, nil
				},
			},
		},
		CursorType: reflect.TypeOf(cursor{}),
		// If we assume the server will always have a relatively small number of channels, we can
		// keep things simple using ResolveAllEdges.
		ResolveAllEdges: func(ctx graphql.FieldContext) (interface{}, func(a, b interface{}) bool, error) {
			channels, err := ctxSession(ctx.Context).GetChannels()
			return channels, func(a, b interface{}) bool {
				ac, bc := a.(cursor), b.(cursor)
				return ac.Name < bc.Name || (ac.Name == bc.Name && ac.Id.Before(bc.Id))
			}, err
		},
	}))
}
