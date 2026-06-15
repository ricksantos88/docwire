package docwire

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
)

// Engine manages the OpenAPI spec state and serves the documentation.
type Engine struct {
	mu   sync.RWMutex
	spec SwaggerSpec
}

// EngineOption configures an Engine at construction time.
type EngineOption func(*Engine)

// WithDescription sets the API description in the Info block.
func WithDescription(desc string) EngineOption {
	return func(e *Engine) { e.spec.Info.Description = desc }
}

// WithTermsOfService sets the terms-of-service URL in the Info block.
func WithTermsOfService(url string) EngineOption {
	return func(e *Engine) { e.spec.Info.TermsOfService = url }
}

// WithContact sets the API contact information.
func WithContact(name, email, url string) EngineOption {
	return func(e *Engine) {
		e.spec.Info.Contact = &Contact{Name: name, Email: email, URL: url}
	}
}

// WithLicense sets the API license name and optional URL.
func WithLicense(name, url string) EngineOption {
	return func(e *Engine) {
		e.spec.Info.License = &License{Name: name, URL: url}
	}
}

// WithServer adds a server entry to the spec (e.g. different environments).
func WithServer(url, description string) EngineOption {
	return func(e *Engine) {
		e.spec.Servers = append(e.spec.Servers, Server{URL: url, Description: description})
	}
}

// WithSecurityScheme registers a named security scheme in components/securitySchemes.
func WithSecurityScheme(name string, scheme SecurityScheme) EngineOption {
	return func(e *Engine) {
		e.spec.Components.SecuritySchemes[name] = scheme
	}
}

// BearerJWT returns a pre-configured HTTP Bearer + JWT security scheme.
func BearerJWT() SecurityScheme {
	return SecurityScheme{Type: "http", Scheme: "bearer", BearerFormat: "JWT"}
}

// APIKeyHeader returns a security scheme that reads an API key from a header.
func APIKeyHeader(headerName string) SecurityScheme {
	return SecurityScheme{Type: "apiKey", In: "header", Name: headerName}
}

// APIKeyQuery returns a security scheme that reads an API key from a query parameter.
func APIKeyQuery(paramName string) SecurityScheme {
	return SecurityScheme{Type: "apiKey", In: "query", Name: paramName}
}

// BasicAuth returns a pre-configured HTTP Basic Auth security scheme.
func BasicAuth() SecurityScheme {
	return SecurityScheme{Type: "http", Scheme: "basic"}
}

// NewEngine initializes a documentation Engine with the given title and version.
func NewEngine(title, version string, opts ...EngineOption) *Engine {
	e := &Engine{
		spec: SwaggerSpec{
			OpenAPI: "3.0.3",
			Info:    Info{Title: title, Version: version},
			Paths:   make(map[string]PathItem),
			Components: Components{
				Schemas:         make(map[string]Schema),
				SecuritySchemes: make(map[string]SecurityScheme),
			},
		},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// AddRoute registers an API endpoint in the OpenAPI spec.
// Supported methods: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS.
func (e *Engine) AddRoute(path, method, summary, description string, opts ...RouteOption) {
	cfg := &routeConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	if len(cfg.responses) == 0 {
		cfg.responses = map[string]responseEntry{
			"200": {description: "Successful Request Execution"},
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	responses := make(map[string]Response, len(cfg.responses))
	for code, entry := range cfg.responses {
		resp := Response{Description: entry.description}
		if entry.model != nil {
			prop := e.buildPropertyLocked(reflect.TypeOf(entry.model))
			resp.Content = map[string]MediaType{
				"application/json": {Schema: prop},
			}
		}
		responses[code] = resp
	}

	var reqBody *RequestBody
	if cfg.reqModel != nil {
		prop := e.buildPropertyLocked(reflect.TypeOf(cfg.reqModel))
		reqBody = &RequestBody{
			Description: cfg.reqDesc,
			Required:    cfg.reqReq,
			Content:     map[string]MediaType{"application/json": {Schema: prop}},
		}
	}

	op := &Operation{
		Tags:        cfg.tags,
		Summary:     summary,
		Description: description,
		Parameters:  cfg.params,
		RequestBody: reqBody,
		Responses:   responses,
		Security:    cfg.security,
	}

	pathItem := e.spec.Paths[path]
	switch strings.ToUpper(method) {
	case http.MethodGet:
		pathItem.Get = op
	case http.MethodPost:
		pathItem.Post = op
	case http.MethodPut:
		pathItem.Put = op
	case http.MethodPatch:
		pathItem.Patch = op
	case http.MethodDelete:
		pathItem.Delete = op
	case http.MethodHead:
		pathItem.Head = op
	case http.MethodOptions:
		pathItem.Options = op
	}
	e.spec.Paths[path] = pathItem
}

// RegisterModel parses a struct via reflection and registers it in components/schemas.
// Returns the $ref path. Accepts structs, pointers, and slices.
func (e *Engine) RegisterModel(model any) string {
	if model == nil {
		return ""
	}
	t := reflect.TypeOf(model)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
	}
	if t.Kind() != reflect.Struct || t == reflect.TypeOf(time.Time{}) {
		return ""
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.registerStructLocked(t)
}

// Handler returns an http.Handler that serves the UI and the spec JSON.
//
//	GET /docwire/         → Swagger UI (HTML)
//	GET /docwire/doc.json → OpenAPI 3.0 spec
func (e *Engine) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/docwire/doc.json", func(w http.ResponseWriter, r *http.Request) {
		e.mu.RLock()
		defer e.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(e.spec)
	})

	mux.HandleFunc("/docwire/", func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSuffix(r.URL.Path, "/") != "/docwire" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(DefaultSwaggerUIHTML("/docwire/doc.json")))
	})

	return mux
}
