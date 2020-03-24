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

	println(gql(`mutation AddReactionToIssue {
	  addReaction(input:{subjectId:"MDU6SXNzdWUyMzEzOTE1NTE=",content:HOORAY}) {
		reaction {
		  content
		}
		subject {
		  id
		}
	  }
	}`))

	println(gql(`query User {
	  node(id:"MDQ6VXNlcjU4MzIzMQ==") {
	   ... on User {
		  name
		  login
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

type ReactionContent string

const (
	ReactionContentHooray     ReactionContent = "HOORAY"
	ReactionContentConfused   ReactionContent = "CONFUSED"
	ReactionContentHeart      ReactionContent = "HEART"
	ReactionContentRocket     ReactionContent = "ROCKET"
	ReactionContentEyes       ReactionContent = "EYES"
	ReactionContentThumbsUp   ReactionContent = "THUMBS_UP"
	ReactionContentThumbsDown ReactionContent = "THUMBS_DOWN"
	ReactionContentLaugh      ReactionContent = "LAUGH"
)

type AddReactionToIssueData struct {
	AddReaction *struct {
		Reaction *struct {
			Content ReactionContent
		}
		Subject *struct {
			Id string
		}
	}
}

type UserData struct {
	Node *struct {
		User *struct {
			Name  *string
			Login string
		}
	}
}
```

It will generate types for all named queries and mutations as well as all named fragments.
