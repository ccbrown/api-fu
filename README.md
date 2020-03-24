# api-fu ![GitHub Actions](https://github.com/ccbrown/api-fu/workflows/Build/badge.svg) [![Documentation](https://godoc.org/github.com/ccbrown/api-fu?status.svg)](https://godoc.org/github.com/ccbrown/api-fu)

**api-fu** (noun)
  1. (informal) Mastery of APIs.

## Packages

* The top level `apifu` package is an opinionated library that aims to make it as easy as possible to build APIs that conform to API-fu's ideals. See the examples directory for example usage.
* The `graphql` package is an unopinionated library for building GraphQL APIs. If you agree with API-fu's ideals, you should use `apifu` instead, but if you want something lower level, the `graphql` package is still an excellent standalone GraphQL library. It fully supports all features of the [June 2018 spec](https://graphql.github.io/graphql-spec/June2018/).
* The `graphqlws` package is an unopinionated library for using the [Apollo graphql-ws protocol](https://github.com/apollographql/subscriptions-transport-ws). This allows you to serve your GraphQL API via WebSockets and provide subscription functionality.

## Features

* Supports all features of the [latest GraphQL spec](https://spec.graphql.org/June2018/), including null literals, error extensions, subscriptions, and directives.
* Supports efficient batching and concurrency without the use of goroutines.
* Provides implementations for commonly used scalar types, such as date-time and long (but JavaScript safe) integers.
* Implements handlers for HTTP and the [Apollo graphql-ws protocol](https://github.com/apollographql/subscriptions-transport-ws).
* Provides easy-to-use helpers for creating connections adhering to the [Relay Cursor Connections Specification](https://facebook.github.io/relay/graphql/connections.htm).
* CLI for generating client-side type definitions and validating queries in source code.

## API Design Guidelines

API-fu aims to make it easy to conform to these guidelines for robust and future-proof APIs:

* All mutations should resolve to result types. No mutations should simply resolve to a node. For example, a `createUser` mutation should resolve to a `CreateUserResult` object with a `user` field rather than simply resolving to a `User`. This is necessary to keep mutations extensible. Likewise, subscriptions should not resolve directly to node types. For example, a subscription for messages in a chat room (`chatRoomMessages`) should resolve to a `ChatRoomMessagesEvent` type.
* Nodes with 1-to-many relationships should make related nodes available via [Relay Cursor Connections](https://facebook.github.io/relay/graphql/connections.htm). Nodes should not have fields that simply resolve to lists of related nodes. Additionally, all connections must require a `first` or `last` argument that specifies the upper bound on the number of nodes returned by that connection. This makes it possible to determine an upper bound on the number of nodes returned by a query before that query begins execution, e.g. using rules similar to [GitHub's](https://developer.github.com/v4/guides/resource-limitations/).
* Mutations that modify nodes should always include the updated version of that node in the result. This makes it easy for clients to maintain up-to-date state and tolerate eventual consistency (If a client updates a resource, then immediately requests it in a subsequent query, the server may provide a version of the resource that was cached before the update.).
* Nodes should provide revision numbers. Each time a node is modified, the revision number must increment. This helps clients maintain up-to-date state and enables simultaneous change detection.
* It should be easy for clients to query historical data and subscribe to real-time data without missing anything due to race conditions. The most transparent and fool-proof way to facilitate this is to make subscriptions immediately push a small history of events to clients as soon as they're started. The pushed history should generally only need to cover a few seconds' worth of events. If queries use eventual consistency, the pushed history should be at least as large as the query cache's TTL.

## Versioning and Compatibility Guarantees

This library is not versioned. However, one guarantee is made: Any backwards-incompatible changes made will break your build at compile-time. If your application compiles after updating API-fu, you're good to go.
