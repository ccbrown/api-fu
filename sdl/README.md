# sdl

⚠️  This package is experimental and subject to change without warning. ⚠️

This package defines a schema definition language (SDL) for API-fu. The SDL can be used to define the schema of an API, which can then be used to generate code for the API in various languages.

It is inspired by the GraphQL SDL, but is designed to describe a schema in sufficient detail to produce GraphQL server implementations, JSON:API server implementations, and client SDKs.

Here's an example:

```
resource Person {
    type: "people"

    attributes {
        firstName: String!
        lastName: String!
    }
}

interface Pet {
    attributes {
        name: String!
    }

    relationships {
        owner: Person!
    }
}

resource Dog implements Pet {
    type: "dogs"

    attributes {
        barkDecibels: Int!
    }
}

resource Cat implements Pet {
    type: "cats"

    attributes {
        meowDecibels: Int!
    }
}
```
