package api

import (
	"context"
	"reflect"

	apifu "github.com/ccbrown/api-fu"
	"github.com/ccbrown/api-fu/examples/chat/model"
	"github.com/ccbrown/api-fu/graphql"
)

var messageType = fuCfg.AddNodeType(&apifu.NodeType{
	Id:    3,
	Name:  "Message",
	Model: reflect.TypeOf(model.Message{}),
	GetByIds: func(ctx context.Context, ids interface{}) (interface{}, error) {
		return ctxSession(ctx).GetMessagesByIds(ids.([]model.Id)...)
	},
})

func init() {
	messageType.Fields = map[string]*graphql.FieldDefinition{
		"id":      apifu.NonNullNodeID("Id"),
		"time":    apifu.NonNullDateTime("Time"),
		"user":    apifu.Node(userType, "UserId"),
		"body":    apifu.NonNullString("Body"),
		"channel": apifu.Node(channelType, "ChannelId"),
	}
}

func init() {
	fuCfg.AddMutation("createMessage", &graphql.FieldDefinition{
		Type: &graphql.ObjectType{
			Name: "CreateMessageResult",
			Fields: map[string]*graphql.FieldDefinition{
				"message": &graphql.FieldDefinition{
					Type: graphql.NewNonNullType(messageType),
					Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
						return ctx.Object, nil
					},
				},
			},
		},
		Arguments: map[string]*graphql.InputValueDefinition{
			"message": &graphql.InputValueDefinition{
				Type: graphql.NewNonNullType(&graphql.InputObjectType{
					Name: "MessageInput",
					Fields: map[string]*graphql.InputValueDefinition{
						"channelId": &graphql.InputValueDefinition{
							Type: graphql.NewNonNullType(graphql.IDType),
						},
						"body": &graphql.InputValueDefinition{
							Type: graphql.NewNonNullType(graphql.StringType),
						},
					},
					InputCoercion: func(input map[string]interface{}) (interface{}, error) {
						return &model.Message{
							ChannelId: DeserializeId(input["channelId"].(string)),
							Body:      input["body"].(string),
						}, nil
					},
				}),
			},
		},
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			return ctxSession(ctx.Context).CreateMessage(ctx.Arguments["message"].(*model.Message))
		},
	})
}
