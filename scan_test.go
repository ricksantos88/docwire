package docwire

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

const scanDir = "./testdata/scanapp"

func TestScanBuildsMetadataFromPackageDoc(t *testing.T) {
	e := Scan(scanDir)

	if got := e.spec.Info.Title; got != "Scan Fixture API" {
		t.Errorf("title = %q, want %q", got, "Scan Fixture API")
	}
	if got := e.spec.Info.Version; got != "v2.3.0" {
		t.Errorf("version = %q, want %q", got, "v2.3.0")
	}
	if got := e.spec.Info.Description; got != "Fixture used by docwire.Scan tests." {
		t.Errorf("description = %q", got)
	}
	if len(e.spec.Servers) != 1 || e.spec.Servers[0].URL != "http://localhost:9090" {
		t.Errorf("servers = %+v", e.spec.Servers)
	}
	if _, ok := e.spec.Components.SecuritySchemes["bearer"]; !ok {
		t.Errorf("bearer security scheme missing: %+v", e.spec.Components.SecuritySchemes)
	}
}

func TestScanRegistersRoutesAndResolvesTypes(t *testing.T) {
	e := Scan(scanDir)

	post := e.spec.Paths["/v1/register"].Post
	if post == nil {
		t.Fatalf("POST /v1/register not registered; paths=%v", e.spec.Paths)
	}
	if post.Summary != "Register a manager" {
		t.Errorf("summary = %q", post.Summary)
	}
	// @Body must resolve to RegisterRequest inferred from the handler body.
	if post.RequestBody == nil {
		t.Fatal("request body not set")
	}
	if ref := post.RequestBody.Content["application/json"].Schema.Ref; ref != "#/components/schemas/RegisterRequest" {
		t.Errorf("body ref = %q", ref)
	}
	if ref := post.Responses["201"].Content["application/json"].Schema.Ref; ref != "#/components/schemas/UserResponse" {
		t.Errorf("201 ref = %q", ref)
	}

	get := e.spec.Paths["/v1/users"].Get
	if get == nil {
		t.Fatal("GET /v1/users not registered")
	}
	if len(get.Security) != 1 {
		t.Errorf("expected @Auth bearer to add security, got %+v", get.Security)
	}
	if ref := get.Responses["200"].Content["application/json"].Schema.Ref; ref != "#/components/schemas/PaginatedUsers" {
		t.Errorf("200 ref = %q", ref)
	}
	hasHeader := false
	for _, p := range get.Parameters {
		if p.In == "header" && p.Name == "X-Request-ID" {
			hasHeader = true
		}
	}
	if !hasHeader {
		t.Errorf("expected header param X-Request-ID, got %+v", get.Parameters)
	}

	exp := e.spec.Paths["/v1/users/export"].Post
	if exp == nil {
		t.Fatal("POST /v1/users/export not registered")
	}
	if ref := exp.RequestBody.Content["application/json"].Schema.Ref; ref != "#/components/schemas/RegisterRequest" {
		t.Errorf("explicit @Body TypeName ref = %q", ref)
	}
	respSchema := exp.Responses["200"].Content["application/json"].Schema
	if respSchema.Type != "array" || respSchema.Items == nil || respSchema.Items.Ref != "#/components/schemas/UserResponse" {
		t.Errorf("slice @Response schema = %+v", respSchema)
	}
}

func TestScanBuildsSchemasFromAST(t *testing.T) {
	e := Scan(scanDir)

	reg, ok := e.spec.Components.Schemas["RegisterRequest"]
	if !ok {
		t.Fatalf("RegisterRequest schema missing; have %v", e.spec.Components.Schemas)
	}
	// json:"-" field must be dropped.
	if _, exists := reg.Properties["Password"]; exists {
		t.Error("json:\"-\" field should be skipped")
	}
	// validate:"required" promotes to required list.
	if !slices.Contains(reg.Required, "name") || !slices.Contains(reg.Required, "email") {
		t.Errorf("required = %v, want name & email", reg.Required)
	}
	// numeric format inference.
	if f := reg.Properties["age"].Format; f != "int32" {
		t.Errorf("age format = %q, want int32", f)
	}
	if f := reg.Properties["email"].Format; f != "email" {
		t.Errorf("email format override = %q", f)
	}

	// nested slice of struct must register UserResponse transitively.
	user, ok := e.spec.Components.Schemas["UserResponse"]
	if !ok {
		t.Fatal("UserResponse not registered transitively")
	}
	if user.Properties["roles"].Type != "array" {
		t.Errorf("roles type = %q, want array", user.Properties["roles"].Type)
	}
	if ex := user.Properties["id"].Example; ex != "u_123" {
		t.Errorf("id example = %v", ex)
	}
	// embedded AuditMeta field must be flattened into UserResponse.
	if _, exists := user.Properties["tenant_id"]; !exists {
		t.Errorf("embedded field tenant_id should be flattened into UserResponse: %+v", user.Properties)
	}
	if !slices.Contains(user.Required, "tenant_id") {
		t.Errorf("embedded required field tenant_id missing from Required: %v", user.Required)
	}
	// unexported field must be skipped.
	if _, exists := user.Properties["internal"]; exists {
		t.Error("unexported field should be skipped")
	}
}

func TestScanHandlerServesSpec(t *testing.T) {
	e := Scan(scanDir)
	req := httptest.NewRequest(http.MethodGet, "/docwire/doc.json", nil)
	rr := httptest.NewRecorder()
	e.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var spec map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&spec); err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	if spec["openapi"] != "3.0.3" {
		t.Errorf("openapi = %v", spec["openapi"])
	}
}

func TestScanSkipsUnreadableDir(t *testing.T) {
	// Non-existent dir must not panic; returns an engine with defaults.
	e := Scan("./testdata/does-not-exist")
	if e == nil || e.spec.Info.Title != "API" {
		t.Errorf("expected default engine, got %+v", e)
	}
}
