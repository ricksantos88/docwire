// Package adapters is part of the docwire library.
// It provides a single generic loader that feeds parsed route annotations into
// the docwire documentation engine and delegates actual HTTP handler registration
// to a caller-supplied callback — keeping this package free of any framework dependency.
//
// Usage:
//
//	// net/http
//	adapters.Load(engine, routes, handlers.NetHTTPRegistry, resolver,
//	    func(method, path string, h http.HandlerFunc) {
//	        mux.HandleFunc(method+" "+path, h)
//	    },
//	)
//
//	// Fiber
//	adapters.Load(engine, routes, handlers.FiberRegistry, resolver,
//	    func(method, path string, h fiber.Handler) {
//	        app.Add(method, path, h)
//	    },
//	)
//
//	// Any other router — just implement the register callback.
package adapters

import (
	"fmt"

	"github.com/ricksantos88/docwire"
	"github.com/ricksantos88/docwire/parser"
)

// BodyResolver maps a @Response type name string to a typed zero-value that
// docwire uses to infer the JSON schema via reflection. Return nil for unknown types.
type BodyResolver func(typeName string) any

// RegisterFunc is a framework-specific callback that wires a handler into the router.
// method is the HTTP verb (e.g. "GET"), path is the route pattern, handler is the
// framework-native handler value taken from the registry.
type RegisterFunc[H any] func(method, path string, handler H)

// LoadResult holds the outcome of a single route registration attempt.
// Err is non-nil when the handler function name from the annotation was not
// found in the registry — meaning the route is documented but not wired.
type LoadResult struct {
	Route parser.RouteAnnotation
	Err   error
}

// Load registers every annotated route with the documentation engine and the
// HTTP router. For each route whose handler is missing from the registry,
// a LoadResult with a non-nil Err is returned instead of silently skipping it.
//
//   - registry maps Go function names to framework-native handler values.
//   - resolver translates @Response type names to zero-value instances for schema inference.
//   - register is called once per matched route to wire the handler into the router.
func Load[H any](
	engine *docwire.Engine,
	routes []parser.RouteAnnotation,
	registry map[string]H,
	resolver BodyResolver,
	register RegisterFunc[H],
) []LoadResult {
	results := make([]LoadResult, 0, len(routes))
	for _, r := range routes {
		engine.AddRoute(r.Path, r.Method, r.Summary, r.Description, buildOpts(r, resolver)...)

		h, ok := registry[r.FuncName]
		if !ok {
			results = append(results, LoadResult{
				Route: r,
				Err:   fmt.Errorf("docwire: handler %q annotated at %s %s not found in registry", r.FuncName, r.Method, r.Path),
			})
			continue
		}
		register(r.Method, r.Path, h)
		results = append(results, LoadResult{Route: r})
	}
	return results
}

// MustLoad is identical to Load but panics on the first registry mismatch.
// Use in main() to fail fast during startup rather than serving undocumented gaps.
func MustLoad[H any](
	engine *docwire.Engine,
	routes []parser.RouteAnnotation,
	registry map[string]H,
	resolver BodyResolver,
	register RegisterFunc[H],
) {
	for _, res := range Load(engine, routes, registry, resolver, register) {
		if res.Err != nil {
			panic(res.Err)
		}
	}
}

// buildOpts translates a RouteAnnotation into the variadic option slice
// expected by engine.AddRoute.
func buildOpts(r parser.RouteAnnotation, resolve BodyResolver) []docwire.RouteOption {
	var opts []docwire.RouteOption

	if len(r.Tags) > 0 {
		opts = append(opts, docwire.WithTags(r.Tags...))
	}
	for _, q := range r.QueryParams {
		opts = append(opts, docwire.WithQueryParam(q.Name, q.Description, q.Required))
	}
	for _, p := range r.PathParams {
		opts = append(opts, docwire.WithPathParam(p.Name, p.Description))
	}
	if r.Body != nil {
		var bodyExample any
		if resolve != nil {
			bodyExample = resolve("Body")
		}
		opts = append(opts, docwire.WithRequestBody(r.Body.Description, r.Body.Required, bodyExample))
	}
	for _, resp := range r.Responses {
		var body any
		if resolve != nil {
			body = resolve(resp.TypeName)
		}
		opts = append(opts, docwire.WithResponse(resp.Code, resp.Description, body))
	}
	for _, scheme := range r.Auth {
		opts = append(opts, docwire.WithSecurity(scheme))
	}

	return opts
}
