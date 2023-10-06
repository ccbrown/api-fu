package api

import (
	apifu "github.com/ccbrown/api-fu"
	"github.com/ccbrown/api-fu/examples/chat/model"
	"github.com/ccbrown/api-fu/graphql"
)

var messageType = &graphql.ObjectType{
	Name:                  "Message",
	ImplementedInterfaces: []*graphql.InterfaceType{fuCfg.NodeInterface()},
	IsTypeOf: func(value interface{}) bool {
		_, ok := value.(*model.Message)
		return ok
	},
}

func init() {
	messageType.Fields = map[string]*graphql.FieldDefinition{
		"id": {
			Type: graphql.NewNonNullType(graphql.IDType),
			Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
				return SerializeNodeId(MessageTypeId, ctx.Object.(*model.Message).Id), nil
			},
		},
		"time": apifu.NonNull(apifu.DateTimeType, "Time"),
		"user": {
			Type: userType,
			Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
				users, err := ctxSession(ctx.Context).GetUsersByIds(ctx.Object.(*model.Message).UserId)
				if err != nil || len(users) == 0 {
					return nil, err
				}
				return users[0], nil
			},
		},
		"body": apifu.NonNull(graphql.StringType, "Body"),
		"channel": {
			Type: channelType,
			Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
				users, err := ctxSession(ctx.Context).GetChannelsByIds(ctx.Object.(*model.Message).ChannelId)
				if err != nil || len(users) == 0 {
					return nil, err
				}
				return users[0], nil
			},
		},
	}
}

func init() {
	fuCfg.AddMutation("createMessage", &graphql.FieldDefinition{
		Type: &graphql.ObjectType{
			Name: "CreateMessageResult",
			Fields: map[string]*graphql.FieldDefinition{
				"message": {
					Type: graphql.NewNonNullType(messageType),
					Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
						return ctx.Object, nil
					},
				},
			},
		},
		Arguments: map[string]*graphql.InputValueDefinition{
			"message": {
				Type: graphql.NewNonNullType(&graphql.InputObjectType{
					Name: "MessageInput",
					Fields: map[string]*graphql.InputValueDefinition{
						"channelId": {
							Type: graphql.NewNonNullType(graphql.IDType),
						},
						"body": {
							Type: graphql.NewNonNullType(graphql.StringType),
						},
					},
					InputCoercion: func(input map[string]interface{}) (interface{}, error) {
						_, channelId := DeserializeNodeId(input["channelId"].(string))
						return &model.Message{
							ChannelId: channelId,
							Body:      input["body"].(string),
						}, nil
					},
				}),
			},
		},
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return ctxSession(ctx.Context).CreateMessage(ctx.Arguments["message"].(*model.Message))
		},
	})
}
