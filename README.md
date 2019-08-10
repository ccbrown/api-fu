# api-fu

**api-fu** (noun)
  1. (informal) Mastery of APIs.

## Packages

* The top level `apifu` package is an opinionated library that aims to make it as easy as possible to build APIs that conform to API-fu's ideals.
* The `graphql` package is an unopinionated library for building GraphQL APIs. If you agree with API-fu's ideals, you should use `apifu` instead, but if you want something lower level, the `graphql` package is still an excellent standalone GraphQL library.

## Ideals

These are the guiding principles behind API-fu's design.

* GraphQL is presently the best standard for web APIs. Thus API-fu's focus is on building excellent GraphQL APIs.
* All mutations should resolve to result types. No mutations should simply resolve to a node. For example, a `createUser` mutation should resolve to a `CreateUserResult` object with a `user` field rather than simply resolving to a `User`. This is necessary to keep mutations extensible.
* Nodes with 1-to-many relationships should make related nodes available via [Relay Cursor Connections](https://facebook.github.io/relay/graphql/connections.htm). Nodes should not have fields that simply resolve to lists of related nodes.
* Queries should have well-defined upper bounds on complexity. To this end, all connections must require a `first` or `last` argument that specifies the upper bound on the number of nodes returned by that connection. This makes it possible to determine an upper bound on the number of nodes returned by a query before that query begins execution, e.g. using rules similar to [GitHub's](https://developer.github.com/v4/guides/resource-limitations/).
