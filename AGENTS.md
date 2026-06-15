# AGENTS.md — docwire

Guidelines for AI agents (Claude Code, Copilot, etc.) working in this repository.

## Project overview

`docwire` is a minimal OpenAPI 3.0 documentation middleware for Go. The core is split
across focused source files in the root package:

- [engine.go](engine.go) — engine state, route registration, `net/http` handler
- [spec.go](spec.go) / [schema.go](schema.go) — OpenAPI spec types and reflection-based model registration
- [options.go](options.go) — functional options (servers, security, params, responses)
- [ui.go](ui.go) — generates the docs HTML page, rendered with Swagger UI (CDN-backed, no embedded assets)

The generated document is a standard OpenAPI 3.0 spec; Swagger UI is used only as the
viewer. The module path is `github.com/ricksantos88/docwire`.

## Scope rules

**Only touch** `swagger.go` and `ui.go` for core changes. Examples live under `example/` and exist solely to demonstrate usage — do not alter them unless the API surface changes.

**Do not** add dependencies to `go.mod` for the core package. The core must remain dependency-free (only stdlib). Fiber is a dev/example dependency only.

## Code style

- No unnecessary comments. Existing godoc comments on exported symbols are intentional — keep them.
- No error wrapping or recovery for internal logic; only validate at the HTTP boundary.
- Struct tags drive the public API surface (`json`, `description`) — do not change tag semantics without updating the README.
- The `Engine` type is the single entry point — do not expose internal types or helpers.

## Adding HTTP methods

Currently only `GET` and `POST` are supported in `AddRoute`. To add more methods:

1. Add the corresponding field to `PathItem` in [swagger.go](swagger.go).
2. Add the `case` to the `switch` inside `AddRoute`.
3. Update the README method support table.

## Adding tests

Tests should use the standard `testing` package — no test framework. Test files go in the root package (`package docwire`). The main things worth testing:

- `RegisterModel` — correct schema generation from struct reflection
- `AddRoute` — correct `PathItem` and `Operation` population
- `Handler` — HTTP responses for `/docwire/` and `/docwire/doc.json`
- Concurrency — call `AddRoute` from multiple goroutines to validate mutex safety

## Endpoints served

| Path | Handler |
|---|---|
| `GET /docwire/` | HTML — Swagger UI via CDN |
| `GET /docwire/doc.json` | JSON — OpenAPI 3.0 spec |

Any path under `/docwire/` that is not exactly `/docwire` returns 404.

## Running examples

```bash
go run ./example/nethttp   # http://localhost:8080/docwire/
go run ./example/fiber     # http://localhost:3000/docwire/
```

## What agents should NOT do

- Do not refactor the reflection logic in `RegisterModel` without running the examples end-to-end.
- Do not replace the CDN Swagger UI with embedded assets unless the user explicitly asks — it would add binary weight to the module.
- Do not add logging to the core package.
- Do not change the mount prefix `/docwire` without updating both the handler and the examples.
