# gql-client-gen

This is a CLI tool that can be used to generate types for use by client code. If you wrap your queries in a "gql()" function call, the tool will validate your queries and generate types for it. For example, running the tool on this file:

```go
package main

// Wrapper to mark queries for gql-client-gen.
func gql(s string) string {
	return s
}

func main() {
	println(gql(`query FindIssueID {
	  repository(owner:"octocat", name:"Hello-World") {
		issue(number:349) {
		  id
		}
	  }
	}`))
}
```

Will generate output like this:

```go
package main

type FindIssueIDData struct {
	Repository *struct {
		Issue *struct {
			Id string
		}
	}
}
```

It will generate types for all named queries and mutations as well as all named fragments.

## Inline Fragments and Fragment Spreads

Types can also be generated for queries that involve fragments with type conditions. In these cases, your queries must select `__typename` so the generated types can know which spreads to unmarshal. For example:

```go
println(gql(`query User {
  node(id:"MDQ6VXNlcjU4MzIzMQ==") {
   __typename
   ... on User {
      name
      login
    }
  }
}`))
```

Will generate something like...:

```go
type selNode0 struct {
	Typename__ string `json:"__typename"`
	User       *struct {
		Name  *string
		Login string
	} `json:"-"`
}

func (s *selNode0) UnmarshalJSON(b []byte) error {
	var base struct {
		Typename__ string `json:"__typename"`
		User       *struct {
			Name  *string
			Login string
		} `json:"-"`
	}
	if err := json.Unmarshal(b, &base); err != nil {
		return err
	}
	*s = base
	switch base.Typename__ {
	case "User":
		if err := json.Unmarshal(b, &s.User); err != nil {
			return err
		}
	}
	return nil
}

type UserData struct {
	Node *selNode0
}
```

After unmarshaling, `UserData.Node.User` will be nil or non-nil depending on the type of the node returned.
