package api

import (
	"context"
	"reflect"

	"github.com/ccbrown/api-fu"
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
	Fields: map[string]*graphql.FieldDefinition{
		"id":           apifu.NonNullNodeID(reflect.TypeOf(model.Channel{}), "Id"),
		"name":         apifu.NonNullString("Name"),
		"creationTime": apifu.NonNullDateTime("CreationTime"),
		"creator":      apifu.Node(userType, "CreatorUserId"),
	},
})

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

type ChannelCursor struct {
	Name string
	Id   model.Id
}

func init() {
	fuCfg.AddQueryField("channelsConnection", apifu.Connection(&apifu.ConnectionConfig{
		NamePrefix:  "QueryChannels",
		Description: "Provides channels sorted by name.",
		CursorLess: func(a, b interface{}) bool {
			aCursor := a.(ChannelCursor)
			bCursor := b.(ChannelCursor)
			return aCursor.Name < bCursor.Name || (aCursor.Name == bCursor.Name && aCursor.Id.Before(bCursor.Id))
		},
		EdgeCursor: func(edge interface{}) interface{} {
			channel := edge.(*model.Channel)
			return ChannelCursor{
				Name: channel.Name,
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
		ResolveAllEdges: func(ctx *graphql.FieldContext) (interface{}, error) {
			return ctxSession(ctx.Context).GetChannels()
		},
	}))
}
