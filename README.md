# docwire

Lightweight OpenAPI 3.0 documentation middleware for Go. Zero external dependencies for the core — works with any `net/http`-compatible router and adapts to frameworks like Fiber via their `net/http` adaptor.

## Features

- **Zero-config source scan** — `docwire.Scan(dirs...)` builds the whole spec from your source at startup: no registry, no resolver maps, no codegen, no extra commands
- **Declarative annotation system** — document routes directly in handler doc-comments; no separate registration blocks
- Generates an OpenAPI 3.0 spec from Go structs via reflection
- Serves Swagger UI at `/docwire/` and the raw JSON spec at `/docwire/doc.json`
- Generic, framework-agnostic adapter (`adapters.Load`) — works with `net/http`, Fiber, or any router
- Thread-safe spec building with `sync.RWMutex`
- Supports all HTTP methods: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
- Request body and multi-status response documentation
- Path, query, and header parameters
- Tags / endpoint grouping
- Security schemes: Bearer JWT, API Key, Basic Auth, OAuth2
- `enum`, `example`, and `format` via struct tags
- Automatic `time.Time` → `string/date-time` mapping
- Numeric format inference (`int32`, `int64`, `float`, `double`)
- Nested struct and slice schema auto-registration
- Self-referential type support
- Servers block (multi-environment URLs)
- Info block: description, contact, license, terms of service

## Installation

```bash
go get github.com/ricksantos88/docwire
```

## Quick Start — Zero-config Scan (recommended)

The fastest path: annotate your handlers, declare the API metadata once in a
package doc-comment, then call `docwire.Scan` with the directories that hold your
handlers. Scan reads the source at startup — both the handler annotations **and**
the referenced DTO structs — and resolves every `@Response`/`@Body` type by
reading its declaration. No registry, no resolver `switch`, no code generation,
no build step beyond `go run`.

### 1. Declare global metadata once (package doc-comment)

```go
// cmd/api/main.go
//
// @Title       My API
// @Version     v1.0.0
// @Description Customer-facing REST API.
// @Server      http://localhost:8080  "Local"
// @Security    bearer  bearerJWT
package main
```

### 2. Annotate your handlers (see the annotation reference below)

The `@Body` request DTO is inferred automatically from the handler body
(the first locally-declared `var x T`, e.g. `var req CreateUserRequest`), so it
needs no type name in the annotation.

### 3. Wire it with a single call

```go
package main

import (
    "net/http"

    "github.com/ricksantos88/docwire"
)

func main() {
    // Reads ./internal/http (and the DTOs it references) and serves the docs.
    handler := docwire.Scan("./internal/http").Handler()

    mux := http.NewServeMux()
    mux.Handle("/docwire/", handler)
    http.ListenAndServe(":8080", mux)
}
```

`go run ./cmd/api` → Swagger UI live at `http://localhost:8080/docwire/`.

> **Runtime requirement.** Scan reads `.go` files, so the source must be present
> where the program runs (always true under `go run`). For a stripped binary
> shipped without source (e.g. a `scratch` container), use the declarative
> `parser` + `adapters` flow below, or the programmatic API. Gate `Scan` behind
> your "docs enabled" flag so production never pays the parse cost.

---

## Quick Start — Declarative (annotations + registry)

Document routes directly in your handler doc-comments using `@` annotations. The `parser` sub-package reads those comments at startup and wires everything automatically via the generic `adapters.Load` function.

### 1. Annotate your handlers

```go
// package handlers

// ListUsers returns paginated user records.
//
// @Route    GET /api/users
// @Summary  List Users
// @Desc     Returns paginated user records.
// @Tags     users
// @Query    page  "Page number (1-based)"   optional
// @Query    limit "Results per page (max 100)" optional
// @Response 200 UserResponse  "Successful"
// @Response 401 ErrorResponse "Unauthorized"
// @Auth     bearer
// @Cache    60s
// @For      nethttp
func ListUsers(w http.ResponseWriter, r *http.Request) { ... }
```

**Annotation reference:**

| Tag | Format | Description |
|---|---|---|
| `@Route` | `METHOD /path` | HTTP method + path (required) |
| `@Summary` | `text` | Short operation title |
| `@Desc` | `text` | Longer description |
| `@Tags` | `tag1,tag2` | Grouping tags |
| `@Query` | `name "desc" required\|optional` | Query-string parameter |
| `@Path` | `name "desc"` | Path parameter (always required) |
| `@Header` | `name "desc" required\|optional` | Header parameter |
| `@Body` | `[TypeName] "desc" required\|optional` | Request body (optional `TypeName`; inferred from handler body in `Scan`) |
| `@Response` | `code TypeName "desc"` | Possible response (`TypeName` or `[]TypeName`; repeat for multiple) |
| `@Auth` | `schemeName` | Security scheme to apply |
| `@Cache` | `duration` | Suggested cache TTL (informational) |
| `@For` | `nethttp\|fiber` | Target framework (default: `nethttp`) |

**Global tags** (placed on a package doc-comment, read by `docwire.Scan`):

| Tag | Format | Description |
|---|---|---|
| `@Title` | `text` | API title |
| `@Version` | `text` | API version |
| `@Description` | `text` | API description |
| `@Server` | `url "desc"` | Server entry (repeat for multiple environments) |
| `@Security` | `name type [arg]` | Security scheme. `type` ∈ `bearerJWT`, `basic`, `apiKeyHeader`, `apiKeyQuery`; `arg` is the header/query name for the `apiKey*` types |

### 2. Build a registry

Map function-name strings to actual function references so the adapter can wire them without reflection on function values:

```go
// handlers/registry.go
var Registry = map[string]http.HandlerFunc{
    "ListUsers":   ListUsers,
    "GetUser":     GetUser,
    "CreateUser":  CreateUser,
}
```

### 3. Wire everything at startup

```go
package main

import (
    "net/http"
    "log"

    "github.com/ricksantos88/docwire"
    "github.com/ricksantos88/docwire/adapters"
    "github.com/ricksantos88/docwire/parser"
    "mymodule/handlers"
)

func main() {
    engine := docwire.NewEngine("My API", "v1.0.0",
        docwire.WithServer("http://localhost:8080", "Local"),
        docwire.WithSecurityScheme("bearer", docwire.BearerJWT()),
    )

    routes, err := parser.ParseDir("./handlers")
    if err != nil {
        log.Fatal(err)
    }

    mux := http.NewServeMux()

    adapters.Load(engine, routes, handlers.Registry, responseResolver,
        func(method, path string, h http.HandlerFunc) {
            mux.HandleFunc(method+" "+path, h)
        },
    )

    mux.Handle("/docwire/", engine.Handler())
    log.Fatal(http.ListenAndServe(":8080", mux))
}

func responseResolver(typeName string) any {
    switch typeName {
    case "UserResponse":
        return handlers.UserResponse{}
    case "[]UserResponse":
        return []handlers.UserResponse{}
    case "ErrorResponse":
        return handlers.ErrorResponse{}
    }
    return nil
}
```

Swagger UI → `http://localhost:8080/docwire/`
Raw spec   → `http://localhost:8080/docwire/doc.json`

### Fiber

The same pattern works with Fiber — only the registry type and the register callback change:

```go
var FiberRegistry = map[string]fiber.Handler{
    "ListUsersFiber":   ListUsersFiber,
    "CreateUsersFiber": CreateUsersFiber,
}

adapters.Load(engine, routes, handlers.FiberRegistry, responseResolver,
    func(method, path string, h fiber.Handler) {
        app.Add(method, path, h)
    },
)

// Mount Swagger UI (Fiber uses fasthttp internally)
app.All("/docwire/*", adaptor.HTTPHandler(engine.Handler()))
```

See [example/nethttp/main.go](example/nethttp/main.go) and [example/fiber/main.go](example/fiber/main.go) for full runnable examples.

---

## Programmatic API (lower-level)

If you prefer to register routes in code instead of doc-comments, you can call `engine.AddRoute` directly — no `parser` or `adapters` packages needed.

```go
engine.AddRoute("/api/users", "GET", "List Users", "Returns all users",
    docwire.WithTags("Users"),
    docwire.WithQueryParam("page", "Page number", false),
    docwire.WithResponse(200, "OK", []UserResponse{}),
    docwire.WithResponse(401, "Unauthorized", ErrorResponse{}),
    docwire.WithSecurity("bearer"),
)

engine.AddRoute("/api/users/{id}", "POST", "Create User", "Creates a user",
    docwire.WithTags("Users"),
    docwire.WithRequestBody("User data", true, CreateUserRequest{}),
    docwire.WithResponse(201, "Created", UserResponse{}),
    docwire.WithSecurity("bearer"),
)

mux := http.NewServeMux()
mux.Handle("/docwire/", engine.Handler())
http.ListenAndServe(":8080", mux)
```

---

## API Reference

### `NewEngine(title, version string, opts ...EngineOption) *Engine`

Creates a new documentation engine.

**Engine options:**

| Option | Description |
|---|---|
| `WithDescription(desc)` | API description in the Info block |
| `WithContact(name, email, url)` | API contact information |
| `WithLicense(name, url)` | API license |
| `WithTermsOfService(url)` | Terms of service URL |
| `WithServer(url, description)` | Add a server entry (call multiple times for multiple environments) |
| `WithSecurityScheme(name, scheme)` | Register a security scheme in `components/securitySchemes` |

### `Scan(dirs ...string) *Engine`

Builds a fully-populated `*Engine` from source — the zero-config entry point.
For each directory it reads the package doc-comment for global metadata
(`@Title`/`@Version`/`@Server`/`@Security`), extracts one route per annotated
handler, infers each `@Body` DTO from the handler body, and registers the JSON
schema of every `@Response`/`@Body` type by reading its struct declaration in the
scanned source (`time.Time` → `date-time`, slices → arrays, nested structs
recurse, `json:"-"` and unexported fields are skipped, `validate`/`binding`/
`required` tags drive the `required` list). Unreadable directories are skipped so
a partial tree never crashes startup. Pass every directory that holds either a
handler or a referenced DTO.

```go
handler := docwire.Scan("./internal/http", "./internal/dto").Handler()
```

### `engine.AddRoute(path, method, summary, description string, opts ...RouteOption)`

Registers an API endpoint. Supported methods: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`.

**Route options:**

| Option | Description |
|---|---|
| `WithTags(tags...)` | Group the endpoint under one or more tags in Swagger UI |
| `WithPathParam(name, description)` | Document a required path parameter (e.g. `{id}`) |
| `WithQueryParam(name, description, required)` | Document a query parameter |
| `WithHeaderParam(name, description, required)` | Document a header parameter |
| `WithRequestBody(description, required, model)` | Document the request body |
| `WithResponse(statusCode, description, model)` | Document a response (call multiple times for multiple status codes) |
| `WithSecurity(schemeNames...)` | Apply security scheme requirements to the operation |

### `parser.ParseDir(dir string) ([]RouteAnnotation, error)`

Scans all `.go` files in `dir`, extracts `@` annotations from doc-comments, and returns a slice of `RouteAnnotation` structs ready for `adapters.Load`.

### `adapters.Load[H any](...)`

Generic, framework-agnostic loader:

```go
func Load[H any](
    engine    *docwire.Engine,
    routes    []parser.RouteAnnotation,
    registry  map[string]H,
    resolver  BodyResolver,
    register  RegisterFunc[H],
)
```

- `registry` — map from function-name string to the actual handler value
- `resolver` — `func(typeName string) any` that returns a zero-value of each response/body type
- `register` — `func(method, path string, handler H)` that wires the handler into your router

### `engine.Handler() http.Handler`

Returns an `http.Handler` that serves:

| Path | Content |
|---|---|
| `GET /docwire/` | Swagger UI (HTML) |
| `GET /docwire/doc.json` | OpenAPI 3.0 spec (JSON) |

### Security scheme helpers

| Helper | Description |
|---|---|
| `BearerJWT()` | HTTP Bearer auth with JWT format |
| `APIKeyHeader(headerName)` | API key read from a request header |
| `APIKeyQuery(paramName)` | API key read from a query parameter |
| `BasicAuth()` | HTTP Basic authentication |

## Struct Tags

| Tag | Purpose |
|---|---|
| `json:"name"` | Sets the field name in the schema (falls back to field name) |
| `description:"..."` | Sets the property description |
| `example:"..."` | Sets an example value shown in Swagger UI |
| `enum:"a,b,c"` | Restricts the property to a set of allowed values |
| `format:"..."` | Overrides the inferred format (e.g. `uuid`, `email`, `date`) |

Fields tagged with `json:"-"` or unexported fields are ignored.

## Go Type Mapping

| Go type | OpenAPI type | Format (auto-inferred) |
|---|---|---|
| `string` | `string` | — |
| `int`, `int8`, `int16` | `integer` | — |
| `int32` | `integer` | `int32` |
| `int64` | `integer` | `int64` |
| `float32` | `number` | `float` |
| `float64` | `number` | `double` |
| `bool` | `boolean` | — |
| `[]T`, `[N]T` | `array` | — |
| `struct` | `object` / `$ref` | — |
| `time.Time` | `string` | `date-time` |

## Running the examples

```bash
# net/http example
go run ./example/nethttp

# Fiber example
go run ./example/fiber
```

## Acknowledgements

`docwire` is built on top of the [OpenAPI Specification](https://www.openapis.org/) — the
output is a standard **OpenAPI 3.0** document, used here as both the reference and the
inspiration for the whole project. The interactive docs page is rendered with
[Swagger UI](https://github.com/swagger-api/swagger-ui), loaded from a CDN.

These are open standards/tools `docwire` interoperates with — they are not bundled
dependencies of the core library.

## License

MIT
