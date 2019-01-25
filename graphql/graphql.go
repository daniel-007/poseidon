package graphql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/heroiclabs/nakama/api"
	"github.com/heroiclabs/nakama/runtime"

	"github.com/mastern2k3/poseidon/rpc"
)

type ContextKey string

const (
	GRAPHQL_CTX_NAKAMA_MODULE ContextKey = "nakama_module"
)

var (
	userType = graphql.NewObject(graphql.ObjectConfig{
		Name:        "User",
		Description: "A registered Nakama user.",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The id of the user.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(*api.User).GetId(), nil
				},
			},
			"username": &graphql.Field{
				Type:        graphql.String,
				Description: "The username of the user.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(*api.User).GetUsername(), nil
				},
			},
		},
	})

	fields = graphql.Fields{
		"userByUsername": &graphql.Field{
			Type: userType,
			Args: graphql.FieldConfigArgument{
				"username": &graphql.ArgumentConfig{
					// Description: "If omitted, returns the hero of the whole saga. If " +
					// 	"provided, returns the hero of that particular episode.",
					Type: graphql.NewNonNull(graphql.String),
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				nk := p.Context.Value(GRAPHQL_CTX_NAKAMA_MODULE).(runtime.NakamaModule)
				usernameParam := p.Args["username"].(string)
				users, err := nk.UsersGetUsername(p.Context, []string{usernameParam})
				if err != nil {
					return nil, err
				}
				if len(users) < 1 {
					return nil, fmt.Errorf("no user with username `%s`", usernameParam)
				}
				return users[0], nil
			},
		},
	}
	rootQuery    = graphql.ObjectConfig{Name: "RootQuery", Fields: fields}
	schemaConfig = graphql.SchemaConfig{Query: graphql.NewObject(rootQuery)}
	schema       graphql.Schema
)

var (
	graphQLRoutes = []rpc.RPCRoute{
		&rpc.JsonRoute{"graphql", func() interface{} { return new(GraphQLRequest) }, query},
	}
)

func RegisterGraphQL(init runtime.Initializer) error {
	var err error
	schema, err = graphql.NewSchema(schemaConfig)
	if err != nil {
		return err
	}
	return rpc.RegisterRoutes(init, graphQLRoutes)
}

type GraphQLRequest struct {
	Query string `json:"query"`
}

func query(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, request interface{}) (interface{}, error) {
	query := request.(*GraphQLRequest)
	logger.Info("query: %+v %s", query, runtime.RUNTIME_CTX_MATCH_NODE)
	newCtx := context.WithValue(ctx, GRAPHQL_CTX_NAKAMA_MODULE, nk)
	params := graphql.Params{
		Schema:        schema,
		RequestString: query.Query,
		Context:       newCtx,
	}
	r := graphql.Do(params)
	if len(r.Errors) > 0 {
		logger.Error("failed to execute graphql operation, errors: %+v", r.Errors)
	}
	return r, nil
}
