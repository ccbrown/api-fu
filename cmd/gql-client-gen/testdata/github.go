// +build ignore
package main

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
