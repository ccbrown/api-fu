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
		"id":     apifu.NonNullNodeID(reflect.TypeOf(model.User{}), "Id"),
		"handle": apifu.NonNullString("Handle"),
	},
})

func init() {
	fuCfg.AddMutation("createUser", &graphql.FieldDefinition{
		Type: &graphql.ObjectType{
			Name: "CreateUserResult",
			Fields: map[string]*graphql.FieldDefinition{
				"user": &graphql.FieldDefinition{
					Type: graphql.NewNonNullType(userType),
					Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
						return ctx.Object, nil
					},
				},
			},
		},
		Arguments: map[string]*graphql.InputValueDefinition{
			"user": &graphql.InputValueDefinition{
				Type: graphql.NewNonNullType(&graphql.InputObjectType{
					Name: "UserInput",
					Fields: map[string]*graphql.InputValueDefinition{
						"handle": &graphql.InputValueDefinition{
							Type: graphql.NewNonNullType(graphql.StringType),
						},
						"password": &graphql.InputValueDefinition{
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
}
