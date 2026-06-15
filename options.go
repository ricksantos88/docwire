package docwire

import "fmt"

// RouteOption configures a route registration.
type RouteOption func(*routeConfig)

type routeConfig struct {
	tags      []string
	params    []Parameter
	reqModel  any
	reqDesc   string
	reqReq    bool
	responses map[string]responseEntry
	security  []map[string][]string
}

type responseEntry struct {
	description string
	model       any
}

// WithTags assigns grouping tags to the operation (used by Swagger UI to group endpoints).
func WithTags(tags ...string) RouteOption {
	return func(c *routeConfig) { c.tags = append(c.tags, tags...) }
}

// WithPathParam documents a required path parameter (e.g. {id} in /users/{id}).
func WithPathParam(name, description string) RouteOption {
	return func(c *routeConfig) {
		c.params = append(c.params, Parameter{
			Name:        name,
			In:          "path",
			Description: description,
			Required:    true,
			Schema:      Property{Type: "string"},
		})
	}
}

// WithQueryParam documents a query parameter with an optional required flag.
func WithQueryParam(name, description string, required bool) RouteOption {
	return func(c *routeConfig) {
		c.params = append(c.params, Parameter{
			Name:        name,
			In:          "query",
			Description: description,
			Required:    required,
			Schema:      Property{Type: "string"},
		})
	}
}

// WithHeaderParam documents a header parameter.
func WithHeaderParam(name, description string, required bool) RouteOption {
	return func(c *routeConfig) {
		c.params = append(c.params, Parameter{
			Name:        name,
			In:          "header",
			Description: description,
			Required:    required,
			Schema:      Property{Type: "string"},
		})
	}
}

// WithRequestBody documents the request body for the operation.
// model can be a struct or a slice of structs.
func WithRequestBody(description string, required bool, model any) RouteOption {
	return func(c *routeConfig) {
		c.reqModel = model
		c.reqDesc = description
		c.reqReq = required
	}
}

// WithResponse documents a response for the given HTTP status code.
// model can be nil (no body), a struct, or a slice of structs.
func WithResponse(statusCode int, description string, model any) RouteOption {
	return func(c *routeConfig) {
		if c.responses == nil {
			c.responses = make(map[string]responseEntry)
		}
		c.responses[fmt.Sprintf("%d", statusCode)] = responseEntry{
			description: description,
			model:       model,
		}
	}
}

// WithSecurity applies one or more named security scheme requirements to the operation.
func WithSecurity(schemeNames ...string) RouteOption {
	return func(c *routeConfig) {
		entry := make(map[string][]string, len(schemeNames))
		for _, name := range schemeNames {
			entry[name] = []string{}
		}
		c.security = append(c.security, entry)
	}
}
