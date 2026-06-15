package adapters_test

import (
	"net/http"
	"testing"

	docwire "github.com/ricksantos88/docwire"
	"github.com/ricksantos88/docwire/adapters"
	"github.com/ricksantos88/docwire/parser"
)

func newEngine() *docwire.Engine {
	return docwire.NewEngine("Test", "v1")
}

func dummyRoute(funcName, method, path string) parser.RouteAnnotation {
	return parser.RouteAnnotation{
		FuncName: funcName,
		Method:   method,
		Path:     path,
		Summary:  "summary",
	}
}

func noopRegister(_, _ string, _ http.HandlerFunc) {}

func TestLoad_HappyPath(t *testing.T) {
	registry := map[string]http.HandlerFunc{
		"GetUser": func(w http.ResponseWriter, r *http.Request) {},
	}
	routes := []parser.RouteAnnotation{dummyRoute("GetUser", "GET", "/users/{id}")}

	results := adapters.Load(newEngine(), routes, registry, nil, noopRegister)

	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("unexpected error: %v", results[0].Err)
	}
}

func TestLoad_MissingHandler_ReturnsError(t *testing.T) {
	registry := map[string]http.HandlerFunc{} // empty
	routes := []parser.RouteAnnotation{dummyRoute("GetUser", "GET", "/users/{id}")}

	results := adapters.Load(newEngine(), routes, registry, nil, noopRegister)

	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].Err == nil {
		t.Error("expected error for missing handler, got nil")
	}
}

func TestLoad_PartialMismatch(t *testing.T) {
	registry := map[string]http.HandlerFunc{
		"GetUser": func(w http.ResponseWriter, r *http.Request) {},
		// "CreateUser" intentionally absent
	}
	routes := []parser.RouteAnnotation{
		dummyRoute("GetUser", "GET", "/users/{id}"),
		dummyRoute("CreateUser", "POST", "/users"),
	}

	results := adapters.Load(newEngine(), routes, registry, nil, noopRegister)

	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("GetUser should succeed, got: %v", results[0].Err)
	}
	if results[1].Err == nil {
		t.Error("CreateUser should fail with missing handler error")
	}
}

func TestMustLoad_Panics_OnMissingHandler(t *testing.T) {
	registry := map[string]http.HandlerFunc{} // empty
	routes := []parser.RouteAnnotation{dummyRoute("GetUser", "GET", "/users/{id}")}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLoad should panic on missing handler")
		}
	}()

	adapters.MustLoad(newEngine(), routes, registry, nil, noopRegister)
}

func TestMustLoad_NoPanic_WhenAllPresent(t *testing.T) {
	registry := map[string]http.HandlerFunc{
		"GetUser": func(w http.ResponseWriter, r *http.Request) {},
	}
	routes := []parser.RouteAnnotation{dummyRoute("GetUser", "GET", "/users/{id}")}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustLoad should not panic, got: %v", r)
		}
	}()

	adapters.MustLoad(newEngine(), routes, registry, nil, noopRegister)
}

func TestLoad_EmptyRoutes(t *testing.T) {
	results := adapters.Load(newEngine(), nil, map[string]http.HandlerFunc{}, nil, noopRegister)
	if len(results) != 0 {
		t.Errorf("want 0 results, got %d", len(results))
	}
}

// ── buildOpts coverage via Load ──────────────────────────────────────────────

type userModel struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func testResolver(typeName string) any {
	switch typeName {
	case "UserResponse":
		return userModel{}
	case "ErrorResponse":
		return struct {
			Message string `json:"message"`
		}{}
	}
	return nil
}

func TestLoad_WithQueryAndPathParams(t *testing.T) {
	registry := map[string]http.HandlerFunc{
		"GetUser": func(w http.ResponseWriter, r *http.Request) {},
	}
	route := parser.RouteAnnotation{
		FuncName: "GetUser",
		Method:   "GET",
		Path:     "/users/{id}",
		Summary:  "Get user",
		QueryParams: []parser.Param{
			{Name: "expand", Description: "Expand relations", Required: false},
		},
		PathParams: []parser.Param{
			{Name: "id", Description: "User ID", Required: true},
		},
	}

	results := adapters.Load(newEngine(), []parser.RouteAnnotation{route}, registry, testResolver, noopRegister)
	if len(results) != 1 || results[0].Err != nil {
		t.Fatalf("unexpected error: %v", results[0].Err)
	}
}

func TestLoad_WithBody(t *testing.T) {
	registry := map[string]http.HandlerFunc{
		"CreateUser": func(w http.ResponseWriter, r *http.Request) {},
	}
	route := parser.RouteAnnotation{
		FuncName: "CreateUser",
		Method:   "POST",
		Path:     "/users",
		Body:     &parser.BodyDef{Description: "User payload", Required: true},
		Responses: []parser.ResponseDef{
			{Code: 201, TypeName: "UserResponse", Description: "Created"},
		},
	}

	results := adapters.Load(newEngine(), []parser.RouteAnnotation{route}, registry, testResolver, noopRegister)
	if len(results) != 1 || results[0].Err != nil {
		t.Fatalf("unexpected error: %v", results[0].Err)
	}
}

func TestLoad_WithBodyNilResolver(t *testing.T) {
	registry := map[string]http.HandlerFunc{
		"CreateUser": func(w http.ResponseWriter, r *http.Request) {},
	}
	route := parser.RouteAnnotation{
		FuncName: "CreateUser",
		Method:   "POST",
		Path:     "/users",
		Body:     &parser.BodyDef{Description: "payload", Required: false},
	}

	// resolver nil — should not panic
	results := adapters.Load(newEngine(), []parser.RouteAnnotation{route}, registry, nil, noopRegister)
	if len(results) != 1 || results[0].Err != nil {
		t.Fatalf("unexpected: %v", results[0].Err)
	}
}

func TestLoad_WithAuth(t *testing.T) {
	registry := map[string]http.HandlerFunc{
		"SecureEndpoint": func(w http.ResponseWriter, r *http.Request) {},
	}
	route := parser.RouteAnnotation{
		FuncName: "SecureEndpoint",
		Method:   "GET",
		Path:     "/secure",
		Auth:     []string{"bearer", "apiKey"},
	}

	results := adapters.Load(newEngine(), []parser.RouteAnnotation{route}, registry, nil, noopRegister)
	if len(results) != 1 || results[0].Err != nil {
		t.Fatalf("unexpected error: %v", results[0].Err)
	}
}

func TestLoad_WithResponseAndResolver(t *testing.T) {
	registry := map[string]http.HandlerFunc{
		"ListUsers": func(w http.ResponseWriter, r *http.Request) {},
	}
	route := parser.RouteAnnotation{
		FuncName: "ListUsers",
		Method:   "GET",
		Path:     "/users",
		Responses: []parser.ResponseDef{
			{Code: 200, TypeName: "UserResponse", Description: "OK"},
			{Code: 401, TypeName: "ErrorResponse", Description: "Unauthorized"},
			{Code: 404, TypeName: "UnknownType", Description: "Not Found"},
		},
	}

	results := adapters.Load(newEngine(), []parser.RouteAnnotation{route}, registry, testResolver, noopRegister)
	if len(results) != 1 || results[0].Err != nil {
		t.Fatalf("unexpected error: %v", results[0].Err)
	}
}

func TestLoad_RegisterCallbackInvoked(t *testing.T) {
	registry := map[string]http.HandlerFunc{
		"GetUser": func(w http.ResponseWriter, r *http.Request) {},
	}
	route := parser.RouteAnnotation{
		FuncName: "GetUser",
		Method:   "GET",
		Path:     "/users/{id}",
	}

	var registered []string
	adapters.Load(newEngine(), []parser.RouteAnnotation{route}, registry, nil,
		func(method, path string, _ http.HandlerFunc) {
			registered = append(registered, method+" "+path)
		},
	)
	if len(registered) != 1 || registered[0] != "GET /users/{id}" {
		t.Errorf("register callback wrong: %v", registered)
	}
}
