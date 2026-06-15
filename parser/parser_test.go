package parser

import (
	"testing"
)

// ── ParseDirStrict / ValidationError ─────────────────────────────────────────

func TestValidationError_Format(t *testing.T) {
	e := ValidationError{FuncName: "GetUser", Tag: "@Route", Message: "malformed"}
	want := `docwire/parser: func "GetUser" → @Route: malformed`
	if e.Error() != want {
		t.Errorf("want %q, got %q", want, e.Error())
	}
}

func TestParseDocComment_RouteOK(t *testing.T) {
	doc := "@Route GET /users\n@Summary List users\n"
	ann, found, errs := parseDocComment(doc)
	if !found {
		t.Fatal("should have found @Route")
	}
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if ann.Method != "GET" || ann.Path != "/users" {
		t.Errorf("method/path wrong: %s %s", ann.Method, ann.Path)
	}
}

func TestParseDocComment_RouteMissingPath(t *testing.T) {
	doc := "@Route GET\n"
	_, found, errs := parseDocComment(doc)
	if found {
		t.Error("malformed @Route should not set hasRoute=true")
	}
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].Tag != "@Route" {
		t.Errorf("error tag should be @Route, got %s", errs[0].Tag)
	}
}

func TestParseDocComment_RouteMissingMethod(t *testing.T) {
	doc := "@Route /users\n"
	_, _, errs := parseDocComment(doc)
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %d: %v", len(errs), errs)
	}
}

func TestParseDocComment_RouteUnknownMethod(t *testing.T) {
	doc := "@Route BANANA /users\n"
	_, found, errs := parseDocComment(doc)
	// hasRoute=true porque a linha tem method+path, mas method é inválido
	if !found {
		t.Error("should set hasRoute even with unknown method")
	}
	if len(errs) != 1 || errs[0].Tag != "@Route" {
		t.Errorf("want 1 @Route error, got %v", errs)
	}
}

func TestParseDocComment_MalformedQuery(t *testing.T) {
	// @Query sem nome → parseParam retorna false
	doc := "@Route GET /x\n@Query\n"
	_, _, errs := parseDocComment(doc)
	hasQueryErr := false
	for _, e := range errs {
		if e.Tag == "@Query" {
			hasQueryErr = true
		}
	}
	if !hasQueryErr {
		t.Error("want @Query validation error for empty param name")
	}
}

func TestParseDocComment_MalformedResponse(t *testing.T) {
	doc := "@Route GET /x\n@Response notanumber TypeName \"desc\"\n"
	_, _, errs := parseDocComment(doc)
	hasRespErr := false
	for _, e := range errs {
		if e.Tag == "@Response" {
			hasRespErr = true
		}
	}
	if !hasRespErr {
		t.Error("want @Response validation error for non-numeric code")
	}
}

func TestParseDocComment_ValidPathParam(t *testing.T) {
	doc := "@Route GET /users/{id}\n@Path id \"User ID\"\n"
	ann, _, errs := parseDocComment(doc)
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(ann.PathParams) != 1 || ann.PathParams[0].Name != "id" {
		t.Errorf("path param not parsed: %+v", ann.PathParams)
	}
}

func TestParseDocComment_MalformedPath(t *testing.T) {
	// @Path without name → empty name → parseParam returns false
	doc := "@Route GET /x\n@Path\n"
	_, _, errs := parseDocComment(doc)
	hasPathErr := false
	for _, e := range errs {
		if e.Tag == "@Path" {
			hasPathErr = true
		}
	}
	if !hasPathErr {
		t.Error("want @Path validation error for missing name")
	}
}

func TestParseDocComment_DefaultFramework(t *testing.T) {
	doc := "@Route GET /x\n"
	ann, _, _ := parseDocComment(doc)
	if ann.Framework != "nethttp" {
		t.Errorf("default framework should be nethttp, got %s", ann.Framework)
	}
}

func TestParseDocComment_ExplicitFramework(t *testing.T) {
	doc := "@Route GET /x\n@For fiber\n"
	ann, _, _ := parseDocComment(doc)
	if ann.Framework != "fiber" {
		t.Errorf("want fiber, got %s", ann.Framework)
	}
}

// ── extractQuoted ─────────────────────────────────────────────────────────────

func TestExtractQuoted_WithQuotes(t *testing.T) {
	text, rem := extractQuoted(`"hello world" required`)
	if text != "hello world" {
		t.Errorf("want 'hello world', got %q", text)
	}
	if rem != "required" {
		t.Errorf("want 'required', got %q", rem)
	}
}

func TestExtractQuoted_NoQuotes(t *testing.T) {
	text, rem := extractQuoted("noquotes")
	if text != "noquotes" || rem != "" {
		t.Errorf("want (noquotes, ''), got (%q, %q)", text, rem)
	}
}

// ── parseParam ───────────────────────────────────────────────────────────────

func TestParseParam_Valid(t *testing.T) {
	p, ok := parseParam(`page "Page number" optional`)
	if !ok {
		t.Fatal("should parse valid param")
	}
	if p.Name != "page" {
		t.Errorf("want name=page, got %s", p.Name)
	}
	if p.Description != "Page number" {
		t.Errorf("want desc='Page number', got %s", p.Description)
	}
	if p.Required {
		t.Error("should be optional")
	}
}

func TestParseParam_Required(t *testing.T) {
	p, ok := parseParam(`id "User ID" required`)
	if !ok {
		t.Fatal("should parse valid param")
	}
	if !p.Required {
		t.Error("should be required")
	}
}

func TestParseParam_EmptyName(t *testing.T) {
	_, ok := parseParam("")
	if ok {
		t.Error("empty string should return false")
	}
}

// ── parseResponse ─────────────────────────────────────────────────────────────

func TestParseResponse_Valid(t *testing.T) {
	r, ok := parseResponse(`200 UserResponse "Successful"`)
	if !ok {
		t.Fatal("should parse valid response")
	}
	if r.Code != 200 || r.TypeName != "UserResponse" || r.Description != "Successful" {
		t.Errorf("unexpected: %+v", r)
	}
}

func TestParseResponse_InvalidCode(t *testing.T) {
	_, ok := parseResponse(`abc UserResponse "desc"`)
	if ok {
		t.Error("non-numeric code should return false")
	}
}

// ── parseBody ────────────────────────────────────────────────────────────────

func TestParseBody_RequiredWithDesc(t *testing.T) {
	b := parseBody(`"User payload" required`)
	if b == nil {
		t.Fatal("should return a BodyDef")
	}
	if b.Description != "User payload" {
		t.Errorf("want desc 'User payload', got %q", b.Description)
	}
	if !b.Required {
		t.Error("should be required")
	}
}

func TestParseBody_OptionalWithDesc(t *testing.T) {
	b := parseBody(`"Optional body" optional`)
	if b.Required {
		t.Error("should not be required")
	}
}

func TestParseBody_NoQuotes(t *testing.T) {
	b := parseBody(`somepayload`)
	if b == nil {
		t.Fatal("parseBody should never return nil")
	}
	if b.Description != "somepayload" {
		t.Errorf("want 'somepayload', got %q", b.Description)
	}
}

// ── extractQuoted edge case ──────────────────────────────────────────────────

func TestExtractQuoted_UnclosedQuote(t *testing.T) {
	text, rem := extractQuoted(`"unclosed`)
	if text != "unclosed" {
		t.Errorf("want 'unclosed', got %q", text)
	}
	if rem != "" {
		t.Errorf("want empty remainder, got %q", rem)
	}
}

// ── parseDocComment — remaining tags ────────────────────────────────────────

func TestParseDocComment_AllTags(t *testing.T) {
	doc := `@Route GET /api/v1/users
@Summary List Users
@Desc Returns all users with pagination
@Tags users,admin
@Query page "Page number" optional
@Query limit "Results per page" required
@Path id "User ID"
@Body "User payload" required
@Response 200 UserResponse "OK"
@Response 401 ErrorResponse "Unauthorized"
@Auth bearer
@Cache 60s
@For fiber
`
	ann, found, errs := parseDocComment(doc)
	if !found {
		t.Fatal("should find @Route")
	}
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}

	if ann.Summary != "List Users" {
		t.Errorf("summary wrong: %q", ann.Summary)
	}
	if ann.Description != "Returns all users with pagination" {
		t.Errorf("description wrong: %q", ann.Description)
	}
	if len(ann.Tags) != 2 || ann.Tags[0] != "users" || ann.Tags[1] != "admin" {
		t.Errorf("tags wrong: %v", ann.Tags)
	}
	if len(ann.QueryParams) != 2 {
		t.Errorf("want 2 query params, got %d", len(ann.QueryParams))
	}
	if ann.QueryParams[0].Required {
		t.Error("page should be optional")
	}
	if !ann.QueryParams[1].Required {
		t.Error("limit should be required")
	}
	if len(ann.PathParams) != 1 || ann.PathParams[0].Name != "id" {
		t.Errorf("path params wrong: %v", ann.PathParams)
	}
	if ann.Body == nil || ann.Body.Description != "User payload" || !ann.Body.Required {
		t.Errorf("body wrong: %+v", ann.Body)
	}
	if len(ann.Responses) != 2 {
		t.Errorf("want 2 responses, got %d", len(ann.Responses))
	}
	if ann.Responses[0].Code != 200 || ann.Responses[0].TypeName != "UserResponse" {
		t.Errorf("first response wrong: %+v", ann.Responses[0])
	}
	if len(ann.Auth) != 1 || ann.Auth[0] != "bearer" {
		t.Errorf("auth wrong: %v", ann.Auth)
	}
	if ann.Cache != "60s" {
		t.Errorf("cache wrong: %q", ann.Cache)
	}
	if ann.Framework != "fiber" {
		t.Errorf("framework wrong: %q", ann.Framework)
	}
}

func TestParseDocComment_EmptyAuthIgnored(t *testing.T) {
	doc := "@Route GET /x\n@Auth\n"
	ann, _, _ := parseDocComment(doc)
	if len(ann.Auth) != 0 {
		t.Errorf("empty @Auth should be ignored, got %v", ann.Auth)
	}
}

func TestParseDocComment_NoRouteLine(t *testing.T) {
	doc := "@Summary List Users\n@Tags users\n"
	_, found, _ := parseDocComment(doc)
	if found {
		t.Error("should not be found without @Route")
	}
}

// ── ParseDir / ParseDirStrict ────────────────────────────────────────────────

func TestParseDir_ReturnsAnnotatedFunctions(t *testing.T) {
	routes, err := ParseDir("testdata")
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}
	// testdata has 3 annotated functions: ListUsers, GetUser, CreateUser
	if len(routes) != 3 {
		t.Fatalf("want 3 routes, got %d: %v", len(routes), funcNames(routes))
	}
}

func TestParseDir_IgnoresUnannotatedFunctions(t *testing.T) {
	routes, err := ParseDir("testdata")
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}
	for _, r := range routes {
		if r.FuncName == "NotAnnotated" || r.FuncName == "OnlyComment" {
			t.Errorf("should ignore %s", r.FuncName)
		}
	}
}

func TestParseDir_RouteFields(t *testing.T) {
	routes, err := ParseDir("testdata")
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}
	byName := make(map[string]RouteAnnotation)
	for _, r := range routes {
		byName[r.FuncName] = r
	}

	list, ok := byName["ListUsers"]
	if !ok {
		t.Fatal("ListUsers not found")
	}
	if list.Method != "GET" || list.Path != "/api/v1/users" {
		t.Errorf("ListUsers route wrong: %s %s", list.Method, list.Path)
	}
	if list.Summary != "List Users" {
		t.Errorf("ListUsers summary wrong: %q", list.Summary)
	}
	if len(list.Tags) != 2 {
		t.Errorf("ListUsers tags wrong: %v", list.Tags)
	}
	if len(list.QueryParams) != 2 {
		t.Errorf("ListUsers query params wrong: %v", list.QueryParams)
	}
	if list.Cache != "60s" {
		t.Errorf("ListUsers cache wrong: %q", list.Cache)
	}
	if list.Framework != "nethttp" {
		t.Errorf("ListUsers framework wrong: %q", list.Framework)
	}

	create, ok := byName["CreateUser"]
	if !ok {
		t.Fatal("CreateUser not found")
	}
	if create.Body == nil {
		t.Fatal("CreateUser body should be set")
	}
	if !create.Body.Required {
		t.Error("CreateUser body should be required")
	}
	if create.Framework != "fiber" {
		t.Errorf("CreateUser framework wrong: %q", create.Framework)
	}
}

func TestParseDir_InvalidDir(t *testing.T) {
	_, err := ParseDir("nonexistent_dir_xyz")
	if err == nil {
		t.Error("want error for nonexistent directory, got nil")
	}
}

func TestParseDirStrict_ReturnsValidationErrors(t *testing.T) {
	routes, errs, err := ParseDirStrict("testdata")
	if err != nil {
		t.Fatalf("ParseDirStrict error: %v", err)
	}
	// testdata file has no malformed annotations, so no validation errors
	if len(errs) != 0 {
		t.Errorf("want 0 validation errors, got %d: %v", len(errs), errs)
	}
	if len(routes) != 3 {
		t.Errorf("want 3 routes, got %d", len(routes))
	}
}

func funcNames(routes []RouteAnnotation) []string {
	names := make([]string, len(routes))
	for i, r := range routes {
		names[i] = r.FuncName
	}
	return names
}
