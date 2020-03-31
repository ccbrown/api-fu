package api

import (
	"context"
	"reflect"

	"github.com/ccbrown/api-fu"
	"github.com/ccbrown/api-fu/examples/chat/model"
	"github.com/ccbrown/api-fu/graphql"
)

var userType = fuCfg.AddNodeType(&apifu.NodeType{
	Id:    1,
	Name:  "User",
	Model: reflect.TypeOf(model.User{}),
	GetByIds: func(ctx context.Context, ids interface{}) (interface{}, error) {
		return ctxSession(ctx).GetUsersByIds(ids.([]model.Id)...)
	},
	Fields: map[string]*graphql.FieldDefinition{
		"id":     apifu.OwnID("Id"),
		"handle": apifu.NonNull(graphql.StringType, "Handle"),
	},
})

func init() {
	fuCfg.AddMutation("createUser", &graphql.FieldDefinition{
		Type: &graphql.ObjectType{
			Name: "CreateUserResult",
			Fields: map[string]*graphql.FieldDefinition{
				"user": {
					Type: graphql.NewNonNullType(userType),
					Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
						return ctx.Object, nil
					},
				},
			},
		},
		Arguments: map[string]*graphql.InputValueDefinition{
			"user": {
				Type: graphql.NewNonNullType(&graphql.InputObjectType{
					Name: "UserInput",
					Fields: map[string]*graphql.InputValueDefinition{
						"handle": {
							Type: graphql.NewNonNullType(graphql.StringType),
						},
						"password": {
							Type: graphql.NewNonNullType(graphql.StringType),
						},
					},
					InputCoercion: func(input map[string]interface{}) (interface{}, error) {
						return &model.User{
							Handle:       input["handle"].(string),
							PasswordHash: model.PasswordHash(input["password"].(string)),
						}, nil
					},
				}),
			},
		},
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			return ctxSession(ctx.Context).CreateUser(ctx.Arguments["user"].(*model.User))
		},
	})

	fuCfg.AddQueryField("authenticatedUser", &graphql.FieldDefinition{
		Type: userType,
		Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
			return ctxSession(ctx.Context).User, nil
		},
	})
}
