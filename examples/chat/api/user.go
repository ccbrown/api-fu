package api

import (
	apifu "github.com/ccbrown/api-fu"
	"github.com/ccbrown/api-fu/examples/chat/model"
	"github.com/ccbrown/api-fu/graphql"
)

var userType = &graphql.ObjectType{
	Name: "User",
	Fields: map[string]*graphql.FieldDefinition{
		"id": {
			Type: graphql.NewNonNullType(graphql.IDType),
			Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
				return SerializeNodeId(UserTypeId, ctx.Object.(*model.User).Id), nil
			},
		},
		"handle": apifu.NonNull(graphql.StringType, "Handle"),
	},
	ImplementedInterfaces: []*graphql.InterfaceType{fuCfg.NodeInterface()},
	IsTypeOf: func(value interface{}) bool {
		_, ok := value.(*model.User)
		return ok
	},
}

func init() {
	fuCfg.AddMutation("createUser", &graphql.FieldDefinition{
		Type: &graphql.ObjectType{
			Name: "CreateUserResult",
			Fields: map[string]*graphql.FieldDefinition{
				"user": {
					Type: graphql.NewNonNullType(userType),
					Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
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
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return ctxSession(ctx.Context).CreateUser(ctx.Arguments["user"].(*model.User))
		},
	})

	fuCfg.AddQueryField("authenticatedUser", &graphql.FieldDefinition{
		Type: userType,
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return ctxSession(ctx.Context).User, nil
		},
	})
}
