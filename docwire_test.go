package docwire

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type SimpleStruct struct {
	ID   int    `json:"id"   description:"Unique identifier" example:"42"`
	Name string `json:"name" description:"Display name"`
}

type RequiredByValidate struct {
	Email string `json:"email" validate:"required"`
	Name  string `json:"name"  validate:"required,min=2"`
	Bio   string `json:"bio"`
}

type RequiredByBinding struct {
	Token  string `json:"token"  binding:"required"`
	UserID int    `json:"user_id" binding:"required"`
	Note   string `json:"note"`
}

type RequiredByTag struct {
	Code string `json:"code" required:"true"`
	Desc string `json:"desc"`
}

type NestedParent struct {
	Child SimpleStruct `json:"child"`
	Label string       `json:"label"`
}

type SliceHolder struct {
	Items []SimpleStruct `json:"items"`
}

type SelfRef struct {
	Name     string    `json:"name"`
	Children []SelfRef `json:"children"`
}

type EnumStruct struct {
	Status string `json:"status" enum:"active,inactive,pending" example:"active"`
}

type TaggedFormats struct {
	CreatedAt time.Time `json:"created_at"`
	Score     int32     `json:"score"`
	Price     float32   `json:"price"`
	Amount    int64     `json:"amount"`
	Rate      float64   `json:"rate"`
}

type WithCustomFormat struct {
	UUID string `json:"uuid" format:"uuid" description:"RFC 4122 UUID"`
}

type unexportedFields struct {
	Public  string `json:"public"`
	private string //nolint
}

type IgnoredField struct {
	Keep   string `json:"keep"`
	Hidden string `json:"hidden,omitempty"`
	Skip   string `json:"-"`
}

func TestNewEngine_Defaults(t *testing.T) {
	e := NewEngine("Test API", "v1.0.0")
	if e.spec.OpenAPI != "3.0.3" {
		t.Errorf("openapi version: want 3.0.3, got %s", e.spec.OpenAPI)
	}
	if e.spec.Info.Title != "Test API" {
		t.Errorf("title: want Test API, got %s", e.spec.Info.Title)
	}
	if e.spec.Info.Version != "v1.0.0" {
		t.Errorf("version: want v1.0.0, got %s", e.spec.Info.Version)
	}
	if e.spec.Paths == nil {
		t.Error("paths map should be initialized")
	}
	if e.spec.Components.Schemas == nil {
		t.Error("schemas map should be initialized")
	}
	if e.spec.Components.SecuritySchemes == nil {
		t.Error("securitySchemes map should be initialized")
	}
}

func TestNewEngine_WithDescription(t *testing.T) {
	e := NewEngine("API", "v1", WithDescription("My cool API"))
	if e.spec.Info.Description != "My cool API" {
		t.Errorf("description not set: %s", e.spec.Info.Description)
	}
}

func TestNewEngine_WithContact(t *testing.T) {
	e := NewEngine("API", "v1", WithContact("Author", "author@test.com", "https://test.com"))
	c := e.spec.Info.Contact
	if c == nil {
		t.Fatal("contact should be set")
	}
	if c.Name != "Author" || c.Email != "author@test.com" || c.URL != "https://test.com" {
		t.Errorf("contact fields wrong: %+v", c)
	}
}

func TestNewEngine_WithLicense(t *testing.T) {
	e := NewEngine("API", "v1", WithLicense("MIT", "https://mit.com"))
	l := e.spec.Info.License
	if l == nil {
		t.Fatal("license should be set")
	}
	if l.Name != "MIT" || l.URL != "https://mit.com" {
		t.Errorf("license fields wrong: %+v", l)
	}
}

func TestNewEngine_WithTermsOfService(t *testing.T) {
	e := NewEngine("API", "v1", WithTermsOfService("https://tos.com"))
	if e.spec.Info.TermsOfService != "https://tos.com" {
		t.Errorf("tos not set: %s", e.spec.Info.TermsOfService)
	}
}

func TestNewEngine_WithServer(t *testing.T) {
	e := NewEngine("API", "v1",
		WithServer("http://localhost:8080", "Local"),
		WithServer("https://api.staging.com", "Staging"),
	)
	if len(e.spec.Servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(e.spec.Servers))
	}
	if e.spec.Servers[0].URL != "http://localhost:8080" {
		t.Errorf("first server URL wrong: %s", e.spec.Servers[0].URL)
	}
}

func TestNewEngine_WithSecurityScheme(t *testing.T) {
	e := NewEngine("API", "v1", WithSecurityScheme("bearer", BearerJWT()))
	s, ok := e.spec.Components.SecuritySchemes["bearer"]
	if !ok {
		t.Fatal("scheme not registered")
	}
	if s.Type != "http" || s.Scheme != "bearer" || s.BearerFormat != "JWT" {
		t.Errorf("BearerJWT fields wrong: %+v", s)
	}
}

func TestBearerJWT(t *testing.T) {
	s := BearerJWT()
	if s.Type != "http" || s.Scheme != "bearer" || s.BearerFormat != "JWT" {
		t.Errorf("unexpected: %+v", s)
	}
}

func TestAPIKeyHeader(t *testing.T) {
	s := APIKeyHeader("X-API-Key")
	if s.Type != "apiKey" || s.In != "header" || s.Name != "X-API-Key" {
		t.Errorf("unexpected: %+v", s)
	}
}

func TestAPIKeyQuery(t *testing.T) {
	s := APIKeyQuery("api_key")
	if s.Type != "apiKey" || s.In != "query" || s.Name != "api_key" {
		t.Errorf("unexpected: %+v", s)
	}
}

func TestBasicAuth(t *testing.T) {
	s := BasicAuth()
	if s.Type != "http" || s.Scheme != "basic" {
		t.Errorf("unexpected: %+v", s)
	}
}

func TestRegisterModel_Nil(t *testing.T) {
	e := NewEngine("Test", "v1")
	if ref := e.RegisterModel(nil); ref != "" {
		t.Errorf("nil should return empty ref, got %s", ref)
	}
}

func TestRegisterModel_NonStruct(t *testing.T) {
	e := NewEngine("Test", "v1")
	if ref := e.RegisterModel("a string"); ref != "" {
		t.Errorf("non-struct should return empty ref, got %s", ref)
	}
	if ref := e.RegisterModel(42); ref != "" {
		t.Errorf("int should return empty ref, got %s", ref)
	}
}

func TestRegisterModel_BasicStruct(t *testing.T) {
	e := NewEngine("Test", "v1")
	ref := e.RegisterModel(SimpleStruct{})
	if ref != "#/components/schemas/SimpleStruct" {
		t.Fatalf("unexpected ref: %s", ref)
	}
	schema, ok := e.spec.Components.Schemas["SimpleStruct"]
	if !ok {
		t.Fatal("schema not registered")
	}
	if schema.Type != "object" {
		t.Errorf("type should be object, got %s", schema.Type)
	}

	idProp := schema.Properties["id"]
	if idProp.Type != "integer" {
		t.Errorf("id: want integer, got %s", idProp.Type)
	}
	if idProp.Description != "Unique identifier" {
		t.Errorf("id description: want 'Unique identifier', got %q", idProp.Description)
	}
	if idProp.Example != "42" {
		t.Errorf("id example: want '42', got %v", idProp.Example)
	}

	nameProp := schema.Properties["name"]
	if nameProp.Type != "string" {
		t.Errorf("name: want string, got %s", nameProp.Type)
	}
}

func TestRegisterModel_PointerToStruct(t *testing.T) {
	e := NewEngine("Test", "v1")
	ref := e.RegisterModel(&SimpleStruct{})
	if ref != "#/components/schemas/SimpleStruct" {
		t.Errorf("unexpected ref: %s", ref)
	}
}

func TestRegisterModel_SliceOfStruct(t *testing.T) {
	e := NewEngine("Test", "v1")
	ref := e.RegisterModel([]SimpleStruct{})
	if ref != "#/components/schemas/SimpleStruct" {
		t.Errorf("unexpected ref: %s", ref)
	}
	if _, ok := e.spec.Components.Schemas["SimpleStruct"]; !ok {
		t.Error("schema should be registered")
	}
}

func TestRegisterModel_Idempotent(t *testing.T) {
	e := NewEngine("Test", "v1")
	ref1 := e.RegisterModel(SimpleStruct{})
	ref2 := e.RegisterModel(SimpleStruct{})
	if ref1 != ref2 {
		t.Error("repeated registration should return same ref")
	}
	if len(e.spec.Components.Schemas) != 1 {
		t.Errorf("should have exactly 1 schema, got %d", len(e.spec.Components.Schemas))
	}
}

func TestRegisterModel_NestedStruct(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.RegisterModel(NestedParent{})

	if _, ok := e.spec.Components.Schemas["NestedParent"]; !ok {
		t.Fatal("NestedParent not registered")
	}
	if _, ok := e.spec.Components.Schemas["SimpleStruct"]; !ok {
		t.Fatal("nested SimpleStruct not auto-registered")
	}

	parentSchema := e.spec.Components.Schemas["NestedParent"]
	childProp := parentSchema.Properties["child"]
	if childProp.Ref != "#/components/schemas/SimpleStruct" {
		t.Errorf("child property should be $ref, got %+v", childProp)
	}
}

func TestRegisterModel_SliceFieldInStruct(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.RegisterModel(SliceHolder{})

	schema := e.spec.Components.Schemas["SliceHolder"]
	itemsProp := schema.Properties["items"]
	if itemsProp.Type != "array" {
		t.Errorf("items should be array type, got %s", itemsProp.Type)
	}
	if itemsProp.Items == nil || itemsProp.Items.Ref != "#/components/schemas/SimpleStruct" {
		t.Errorf("items.items should ref SimpleStruct, got %+v", itemsProp.Items)
	}
	if _, ok := e.spec.Components.Schemas["SimpleStruct"]; !ok {
		t.Error("SimpleStruct should be auto-registered")
	}
}

func TestRegisterModel_SelfReferential(t *testing.T) {
	e := NewEngine("Test", "v1")
	// Must not deadlock or panic.
	e.RegisterModel(SelfRef{})
	if _, ok := e.spec.Components.Schemas["SelfRef"]; !ok {
		t.Error("SelfRef should be registered")
	}
	schema := e.spec.Components.Schemas["SelfRef"]
	childrenProp := schema.Properties["children"]
	if childrenProp.Type != "array" {
		t.Errorf("children should be array, got %s", childrenProp.Type)
	}
	if childrenProp.Items == nil || childrenProp.Items.Ref != "#/components/schemas/SelfRef" {
		t.Errorf("children items should ref SelfRef, got %+v", childrenProp.Items)
	}
}

func TestRegisterModel_TimeDotTime(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.RegisterModel(TaggedFormats{})
	schema := e.spec.Components.Schemas["TaggedFormats"]
	prop := schema.Properties["created_at"]
	if prop.Type != "string" || prop.Format != "date-time" {
		t.Errorf("time.Time should be string/date-time, got %s/%s", prop.Type, prop.Format)
	}
}

func TestRegisterModel_NumericFormats(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.RegisterModel(TaggedFormats{})
	schema := e.spec.Components.Schemas["TaggedFormats"]

	cases := []struct {
		field, wantType, wantFormat string
	}{
		{"score", "integer", "int32"},
		{"price", "number", "float"},
		{"amount", "integer", "int64"},
		{"rate", "number", "double"},
	}
	for _, tc := range cases {
		p := schema.Properties[tc.field]
		if p.Type != tc.wantType {
			t.Errorf("%s: want type %s, got %s", tc.field, tc.wantType, p.Type)
		}
		if p.Format != tc.wantFormat {
			t.Errorf("%s: want format %s, got %s", tc.field, tc.wantFormat, p.Format)
		}
	}
}

func TestRegisterModel_EnumAndExample(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.RegisterModel(EnumStruct{})
	schema := e.spec.Components.Schemas["EnumStruct"]
	prop := schema.Properties["status"]
	if len(prop.Enum) != 3 {
		t.Errorf("want 3 enum values, got %d: %v", len(prop.Enum), prop.Enum)
	}
	if prop.Example != "active" {
		t.Errorf("want example 'active', got %v", prop.Example)
	}
}

func TestRegisterModel_CustomFormatTag(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.RegisterModel(WithCustomFormat{})
	schema := e.spec.Components.Schemas["WithCustomFormat"]
	prop := schema.Properties["uuid"]
	if prop.Format != "uuid" {
		t.Errorf("want format uuid, got %s", prop.Format)
	}
	if prop.Description != "RFC 4122 UUID" {
		t.Errorf("want description 'RFC 4122 UUID', got %s", prop.Description)
	}
}

func TestRegisterModel_IgnoresUnexportedAndDashFields(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.RegisterModel(IgnoredField{})
	schema := e.spec.Components.Schemas["IgnoredField"]
	if _, ok := schema.Properties["Skip"]; ok {
		t.Error("json:\"-\" field should be ignored")
	}
	if _, ok := schema.Properties["-"]; ok {
		t.Error("json:\"-\" field should be ignored")
	}
	if _, ok := schema.Properties["keep"]; !ok {
		t.Error("'keep' field should be present")
	}
	if _, ok := schema.Properties["hidden"]; !ok {
		t.Error("'hidden,omitempty' field should be present with name 'hidden'")
	}
}

func TestAddRoute_AllHTTPMethods(t *testing.T) {
	methods := []struct {
		method string
		getter func(PathItem) *Operation
	}{
		{"GET", func(p PathItem) *Operation { return p.Get }},
		{"POST", func(p PathItem) *Operation { return p.Post }},
		{"PUT", func(p PathItem) *Operation { return p.Put }},
		{"PATCH", func(p PathItem) *Operation { return p.Patch }},
		{"DELETE", func(p PathItem) *Operation { return p.Delete }},
		{"HEAD", func(p PathItem) *Operation { return p.Head }},
		{"OPTIONS", func(p PathItem) *Operation { return p.Options }},
	}
	for _, tc := range methods {
		t.Run(tc.method, func(t *testing.T) {
			e := NewEngine("Test", "v1")
			e.AddRoute("/test", tc.method, "Test", "Test")
			if tc.getter(e.spec.Paths["/test"]) == nil {
				t.Errorf("operation for %s not set", tc.method)
			}
		})
	}
}

func TestAddRoute_DefaultResponse(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.AddRoute("/test", "GET", "Test", "Test")
	op := e.spec.Paths["/test"].Get
	if op == nil {
		t.Fatal("operation not set")
	}
	if _, ok := op.Responses["200"]; !ok {
		t.Error("default 200 response should be set when no WithResponse provided")
	}
}

func TestAddRoute_WithMultipleResponses(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.AddRoute("/users", "GET", "List Users", "Returns users",
		WithResponse(200, "OK", SimpleStruct{}),
		WithResponse(400, "Bad Request", nil),
		WithResponse(401, "Unauthorized", nil),
		WithResponse(500, "Internal Error", nil),
	)
	op := e.spec.Paths["/users"].Get
	for _, code := range []string{"200", "400", "401", "500"} {
		if _, ok := op.Responses[code]; !ok {
			t.Errorf("response %s not set", code)
		}
	}
	if _, ok := op.Responses["200"].Content["application/json"]; !ok {
		t.Error("200 should have application/json content with model")
	}
	if op.Responses["400"].Content != nil {
		t.Error("400 with nil model should have no content")
	}
}

func TestAddRoute_WithSliceResponse(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.AddRoute("/users", "GET", "List", "All users",
		WithResponse(200, "OK", []SimpleStruct{}),
	)
	op := e.spec.Paths["/users"].Get
	schema := op.Responses["200"].Content["application/json"].Schema
	if schema.Type != "array" {
		t.Errorf("want array schema, got %s", schema.Type)
	}
	if schema.Items == nil || schema.Items.Ref != "#/components/schemas/SimpleStruct" {
		t.Errorf("items should ref SimpleStruct, got %+v", schema.Items)
	}
	if _, ok := e.spec.Components.Schemas["SimpleStruct"]; !ok {
		t.Error("SimpleStruct should be auto-registered")
	}
}

func TestAddRoute_WithRequestBody(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.AddRoute("/users", "POST", "Create User", "Creates a new user",
		WithRequestBody("New user payload", true, SimpleStruct{}),
		WithResponse(201, "Created", SimpleStruct{}),
	)
	op := e.spec.Paths["/users"].Post
	if op.RequestBody == nil {
		t.Fatal("requestBody not set")
	}
	if op.RequestBody.Description != "New user payload" {
		t.Errorf("requestBody description wrong: %s", op.RequestBody.Description)
	}
	if !op.RequestBody.Required {
		t.Error("requestBody should be required")
	}
	if _, ok := op.RequestBody.Content["application/json"]; !ok {
		t.Error("requestBody should have application/json content")
	}
}

func TestAddRoute_WithPathParam(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.AddRoute("/users/{id}", "GET", "Get User", "Fetch by ID",
		WithPathParam("id", "User ID"),
	)
	op := e.spec.Paths["/users/{id}"].Get
	if len(op.Parameters) != 1 {
		t.Fatalf("want 1 parameter, got %d", len(op.Parameters))
	}
	p := op.Parameters[0]
	if p.Name != "id" || p.In != "path" || !p.Required {
		t.Errorf("path param wrong: %+v", p)
	}
}

func TestAddRoute_WithQueryParam(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.AddRoute("/users", "GET", "List", "All users",
		WithQueryParam("page", "Page number", false),
		WithQueryParam("limit", "Page size", false),
		WithQueryParam("filter", "Filter expression", false),
	)
	op := e.spec.Paths["/users"].Get
	if len(op.Parameters) != 3 {
		t.Fatalf("want 3 parameters, got %d", len(op.Parameters))
	}
	params := make(map[string]Parameter)
	for _, p := range op.Parameters {
		params[p.Name] = p
	}
	for _, name := range []string{"page", "limit", "filter"} {
		if p, ok := params[name]; !ok {
			t.Errorf("param %s not found", name)
		} else if p.In != "query" {
			t.Errorf("param %s should be in query, got %s", name, p.In)
		}
	}
}

func TestAddRoute_WithHeaderParam(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.AddRoute("/secure", "GET", "Secure", "Needs header",
		WithHeaderParam("X-Trace-ID", "Distributed trace ID", false),
	)
	op := e.spec.Paths["/secure"].Get
	if len(op.Parameters) != 1 {
		t.Fatalf("want 1 parameter, got %d", len(op.Parameters))
	}
	p := op.Parameters[0]
	if p.In != "header" || p.Name != "X-Trace-ID" {
		t.Errorf("header param wrong: %+v", p)
	}
}

func TestAddRoute_WithTags(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.AddRoute("/users", "GET", "List", "All users", WithTags("Users", "Admin"))
	op := e.spec.Paths["/users"].Get
	if len(op.Tags) != 2 || op.Tags[0] != "Users" || op.Tags[1] != "Admin" {
		t.Errorf("tags wrong: %v", op.Tags)
	}
}

func TestAddRoute_WithSecurity(t *testing.T) {
	e := NewEngine("Test", "v1",
		WithSecurityScheme("bearer", BearerJWT()),
	)
	e.AddRoute("/secure", "GET", "Secure", "Auth required",
		WithSecurity("bearer"),
	)
	op := e.spec.Paths["/secure"].Get
	if len(op.Security) != 1 {
		t.Fatalf("want 1 security entry, got %d", len(op.Security))
	}
	if _, ok := op.Security[0]["bearer"]; !ok {
		t.Error("bearer security not applied")
	}
}

func TestAddRoute_WithMultipleSecurity(t *testing.T) {
	e := NewEngine("Test", "v1",
		WithSecurityScheme("bearer", BearerJWT()),
		WithSecurityScheme("apiKey", APIKeyHeader("X-API-Key")),
	)
	e.AddRoute("/secure", "GET", "Secure", "Multi auth",
		WithSecurity("bearer"),
		WithSecurity("apiKey"),
	)
	op := e.spec.Paths["/secure"].Get
	if len(op.Security) != 2 {
		t.Errorf("want 2 security entries, got %d", len(op.Security))
	}
}

func TestAddRoute_MultipleMethods_SamePath(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.AddRoute("/users", "GET", "List", "Get all")
	e.AddRoute("/users", "POST", "Create", "Create one")
	item := e.spec.Paths["/users"]
	if item.Get == nil {
		t.Error("GET not set")
	}
	if item.Post == nil {
		t.Error("POST not set")
	}
}

func TestAddRoute_SummaryAndDescription(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.AddRoute("/test", "GET", "My Summary", "My detailed description")
	op := e.spec.Paths["/test"].Get
	if op.Summary != "My Summary" {
		t.Errorf("summary wrong: %s", op.Summary)
	}
	if op.Description != "My detailed description" {
		t.Errorf("description wrong: %s", op.Description)
	}
}

func TestAddRoute_AutoRegistersResponseModel(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.AddRoute("/users", "GET", "List", "All users",
		WithResponse(200, "OK", SimpleStruct{}),
	)
	if _, ok := e.spec.Components.Schemas["SimpleStruct"]; !ok {
		t.Error("response model should be auto-registered in components")
	}
}

// ── Handler ──────────────────────────────────────────────────────────────────

func TestHandler_DocJSON_StatusAndContentType(t *testing.T) {
	e := NewEngine("Test", "v1")
	req := httptest.NewRequest(http.MethodGet, "/docwire/doc.json", nil)
	rr := httptest.NewRecorder()
	e.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("want application/json content-type, got %s", ct)
	}
}

func TestHandler_DocJSON_ValidSpec(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.AddRoute("/users", "GET", "List", "All users",
		WithResponse(200, "OK", SimpleStruct{}),
	)
	req := httptest.NewRequest(http.MethodGet, "/docwire/doc.json", nil)
	rr := httptest.NewRecorder()
	e.Handler().ServeHTTP(rr, req)

	var spec SwaggerSpec
	if err := json.NewDecoder(rr.Body).Decode(&spec); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if spec.OpenAPI != "3.0.3" {
		t.Errorf("openapi version wrong: %s", spec.OpenAPI)
	}
	if _, ok := spec.Paths["/users"]; !ok {
		t.Error("/users path not present in spec")
	}
}

func TestHandler_UIPage_StatusAndContentType(t *testing.T) {
	e := NewEngine("Test", "v1")
	req := httptest.NewRequest(http.MethodGet, "/docwire/", nil)
	rr := httptest.NewRecorder()
	e.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("want text/html content-type, got %s", ct)
	}
	if !strings.Contains(rr.Body.String(), "swagger-ui") {
		t.Error("response should reference swagger-ui")
	}
}

func TestHandler_UIPage_WithoutTrailingSlash(t *testing.T) {
	e := NewEngine("Test", "v1")
	req := httptest.NewRequest(http.MethodGet, "/docwire", nil)
	rr := httptest.NewRecorder()
	// ServeMux faz 301 antes do nosso handler ver; via adaptor (fiber etc) chega direto
	e.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusMovedPermanently && rr.Code != http.StatusOK {
		t.Errorf("want 200 or 301, got %d", rr.Code)
	}
}

func TestHandler_UnknownPath_NotFound(t *testing.T) {
	e := NewEngine("Test", "v1")
	req := httptest.NewRequest(http.MethodGet, "/docwire/unknown-page", nil)
	rr := httptest.NewRecorder()
	e.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rr.Code)
	}
}

func TestRegisterModel_RequiredViaValidateTag(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.RegisterModel(RequiredByValidate{})
	schema := e.spec.Components.Schemas["RequiredByValidate"]

	required := make(map[string]bool)
	for _, f := range schema.Required {
		required[f] = true
	}
	if !required["email"] {
		t.Error("email (validate:required) should be in required list")
	}
	if !required["name"] {
		t.Error("name (validate:required,min=2) should be in required list")
	}
	if required["bio"] {
		t.Error("bio (no required tag) should NOT be in required list")
	}
}

func TestRegisterModel_RequiredViaBindingTag(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.RegisterModel(RequiredByBinding{})
	schema := e.spec.Components.Schemas["RequiredByBinding"]

	required := make(map[string]bool)
	for _, f := range schema.Required {
		required[f] = true
	}
	if !required["token"] {
		t.Error("token (binding:required) should be in required list")
	}
	if !required["user_id"] {
		t.Error("user_id (binding:required) should be in required list")
	}
	if required["note"] {
		t.Error("note should NOT be in required list")
	}
}

func TestRegisterModel_RequiredViaRequiredTag(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.RegisterModel(RequiredByTag{})
	schema := e.spec.Components.Schemas["RequiredByTag"]

	required := make(map[string]bool)
	for _, f := range schema.Required {
		required[f] = true
	}
	if !required["code"] {
		t.Error("code (required:true) should be in required list")
	}
	if required["desc"] {
		t.Error("desc should NOT be in required list")
	}
}

func TestRegisterModel_RequiredOmittedFromJSONWhenEmpty(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.RegisterModel(SimpleStruct{})

	data, err := json.Marshal(e.spec.Components.Schemas["SimpleStruct"])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"required"`) {
		t.Error("schema without required fields should not emit required key in JSON")
	}
}

func TestAddRoute_ConcurrentSafe(t *testing.T) {
	e := NewEngine("Test", "v1")
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			path := fmt.Sprintf("/route/%d", i)
			e.AddRoute(path, "GET", "Test", "Test",
				WithResponse(200, "OK", SimpleStruct{}),
			)
		}(i)
	}
	wg.Wait()
	if len(e.spec.Paths) != n {
		t.Errorf("want %d paths, got %d", n, len(e.spec.Paths))
	}
}

func TestRegisterModel_ConcurrentSafe(t *testing.T) {
	e := NewEngine("Test", "v1")
	var wg sync.WaitGroup
	const n = 50
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			e.RegisterModel(SimpleStruct{})
		}()
	}
	wg.Wait()
	if _, ok := e.spec.Components.Schemas["SimpleStruct"]; !ok {
		t.Error("SimpleStruct should be registered after concurrent calls")
	}
}

func TestHandler_ConcurrentRead(t *testing.T) {
	e := NewEngine("Test", "v1")
	e.AddRoute("/users", "GET", "List", "All users", WithResponse(200, "OK", SimpleStruct{}))
	h := e.Handler()

	var wg sync.WaitGroup
	const n = 50
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/docwire/doc.json", nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("concurrent read: want 200, got %d", rr.Code)
			}
		}()
	}
	wg.Wait()
}

func TestFullSpec_RoundTrip(t *testing.T) {
	e := NewEngine("Full API", "v2.0.0",
		WithDescription("Full test spec"),
		WithContact("Dev Team", "dev@example.com", "https://example.com"),
		WithLicense("Apache 2.0", "https://apache.org/licenses/LICENSE-2.0"),
		WithServer("http://localhost:8080", "Local"),
		WithServer("https://api.example.com", "Production"),
		WithSecurityScheme("bearer", BearerJWT()),
		WithSecurityScheme("apiKey", APIKeyHeader("X-API-Key")),
	)

	e.AddRoute("/users", "GET", "List Users", "Returns all users",
		WithTags("Users"),
		WithQueryParam("page", "Page number", false),
		WithQueryParam("limit", "Results per page", false),
		WithResponse(200, "OK", []SimpleStruct{}),
		WithResponse(401, "Unauthorized", nil),
		WithSecurity("bearer"),
	)

	e.AddRoute("/users/{id}", "GET", "Get User", "Returns one user",
		WithTags("Users"),
		WithPathParam("id", "User ID"),
		WithResponse(200, "OK", SimpleStruct{}),
		WithResponse(404, "Not Found", nil),
		WithSecurity("bearer"),
	)

	e.AddRoute("/users", "POST", "Create User", "Creates a user",
		WithTags("Users"),
		WithRequestBody("User data", true, SimpleStruct{}),
		WithResponse(201, "Created", SimpleStruct{}),
		WithResponse(400, "Bad Request", nil),
		WithSecurity("bearer", "apiKey"),
	)

	e.AddRoute("/users/{id}", "PUT", "Replace User", "Full update",
		WithTags("Users"),
		WithPathParam("id", "User ID"),
		WithRequestBody("Replacement data", true, SimpleStruct{}),
		WithResponse(200, "OK", SimpleStruct{}),
		WithSecurity("bearer"),
	)

	e.AddRoute("/users/{id}", "PATCH", "Update User", "Partial update",
		WithTags("Users"),
		WithPathParam("id", "User ID"),
		WithRequestBody("Partial data", false, SimpleStruct{}),
		WithResponse(200, "OK", SimpleStruct{}),
		WithSecurity("bearer"),
	)

	e.AddRoute("/users/{id}", "DELETE", "Delete User", "Removes a user",
		WithTags("Users"),
		WithPathParam("id", "User ID"),
		WithResponse(204, "No Content", nil),
		WithResponse(404, "Not Found", nil),
		WithSecurity("bearer"),
	)

	// serializa e deserializa pra garantir que o JSON não quebra nada
	req := httptest.NewRequest(http.MethodGet, "/docwire/doc.json", nil)
	rr := httptest.NewRecorder()
	e.Handler().ServeHTTP(rr, req)

	var spec SwaggerSpec
	if err := json.NewDecoder(rr.Body).Decode(&spec); err != nil {
		t.Fatalf("spec is not valid JSON: %v", err)
	}
	if spec.Info.Title != "Full API" {
		t.Errorf("title: want 'Full API', got %s", spec.Info.Title)
	}
	if len(spec.Servers) != 2 {
		t.Errorf("want 2 servers, got %d", len(spec.Servers))
	}
	if len(spec.Components.SecuritySchemes) != 2 {
		t.Errorf("want 2 security schemes, got %d", len(spec.Components.SecuritySchemes))
	}
	if _, ok := spec.Paths["/users"]; !ok {
		t.Error("/users path missing")
	}
	if _, ok := spec.Paths["/users/{id}"]; !ok {
		t.Error("/users/{id} path missing")
	}
}
