package docwire

// Contact holds API contact information.
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License describes the license applied to the API.
type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// Info holds foundational metadata about the API.
type Info struct {
	Title          string   `json:"title"`
	Version        string   `json:"version"`
	Description    string   `json:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
}

// Server describes a target host for the API.
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// Property defines a single field within a schema, supporting both $ref and inline types.
type Property struct {
	Ref         string              `json:"$ref,omitempty"`
	Type        string              `json:"type,omitempty"`
	Format      string              `json:"format,omitempty"`
	Description string              `json:"description,omitempty"`
	Enum        []string            `json:"enum,omitempty"`
	Example     any                 `json:"example,omitempty"`
	Items       *Property           `json:"items,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"`
}

// Schema represents an object type definition stored in components/schemas.
type Schema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

// Parameter describes a single operation parameter (path, query, header, cookie).
type Parameter struct {
	Name        string   `json:"name"`
	In          string   `json:"in"`
	Description string   `json:"description,omitempty"`
	Required    bool     `json:"required"`
	Schema      Property `json:"schema"`
}

// RequestBody describes the payload expected in the request.
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required"`
	Content     map[string]MediaType `json:"content"`
}

// MediaType wraps a schema for a specific content type.
type MediaType struct {
	Schema Property `json:"schema"`
}

// Response models an HTTP response for a given status code.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// Operation documents a single HTTP method on a path.
type Operation struct {
	Tags        []string              `json:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
}

// PathItem defines all HTTP methods available on a specific path.
type PathItem struct {
	Get     *Operation `json:"get,omitempty"`
	Post    *Operation `json:"post,omitempty"`
	Put     *Operation `json:"put,omitempty"`
	Patch   *Operation `json:"patch,omitempty"`
	Delete  *Operation `json:"delete,omitempty"`
	Head    *Operation `json:"head,omitempty"`
	Options *Operation `json:"options,omitempty"`
}

// SecurityScheme defines a security mechanism available in the API.
type SecurityScheme struct {
	Type         string      `json:"type"`
	Description  string      `json:"description,omitempty"`
	Name         string      `json:"name,omitempty"`
	In           string      `json:"in,omitempty"`
	Scheme       string      `json:"scheme,omitempty"`
	BearerFormat string      `json:"bearerFormat,omitempty"`
	Flows        *OAuthFlows `json:"flows,omitempty"`
}

// OAuthFlows holds configuration for all supported OAuth 2.0 flows.
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

// OAuthFlow contains URLs and scopes for a single OAuth 2.0 flow.
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

// Components aggregates reusable schemas and security schemes.
type Components struct {
	Schemas         map[string]Schema         `json:"schemas"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

// SwaggerSpec is the root OpenAPI 3.0 document.
type SwaggerSpec struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Servers    []Server            `json:"servers,omitempty"`
	Paths      map[string]PathItem `json:"paths"`
	Components Components          `json:"components"`
}
