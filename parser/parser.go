// Package parser extracts route annotations from Go source file comments using go/ast.
//
// Each handler function may carry a block of annotation lines directly above its
// declaration. The supported tags are:
//
//	@Route    METHOD /path
//	@Summary  Short one-line summary
//	@Desc     Longer description (one line)
//	@Tags     tag1,tag2
//	@Query    name  "description"  required|optional
//	@Path     name  "description"
//	@Body     "description"  required|optional
//	@Response code  TypeName  "description"
//	@Auth     schemeName
//	@Cache    duration  (e.g. 60s, 5m)
//	@For      nethttp|fiber  (which framework this func belongs to; default: nethttp)
//
// Example:
//
//	// ListCustomers returns paginated customers.
//	//
//	// @Route    GET /api/v1/customers
//	// @Summary  List Customers
//	// @Tags     customers
//	// @Query    page  "Page number (1-based)"  optional
//	// @Query    limit "Results per page"        optional
//	// @Response 200  CustomerResponse  "Successful"
//	// @Response 401  ErrorResponse     "Unauthorized"
//	// @Auth     bearer
//	// @Cache    60s
//	// @For      nethttp
//	func ListCustomers(w http.ResponseWriter, r *http.Request) { ... }
package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// ValidationError describes a malformed annotation found during parsing.
type ValidationError struct {
	FuncName string
	Tag      string
	Message  string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("docwire/parser: func %q → %s: %s", e.FuncName, e.Tag, e.Message)
}

// Param describes a single HTTP parameter (query or path).
type Param struct {
	Name        string
	Description string
	Required    bool
}

// BodyDef describes a request body annotation.
type BodyDef struct {
	Description string
	Required    bool
}

// ResponseDef describes one @Response line.
type ResponseDef struct {
	Code        int
	TypeName    string
	Description string
}

// RouteAnnotation holds all metadata extracted from a single annotated function.
type RouteAnnotation struct {
	// FuncName is the Go identifier of the handler function.
	FuncName string

	// Framework indicates which router this function targets ("nethttp" or "fiber").
	// Defaults to "nethttp" when @For is absent.
	Framework string

	Method      string
	Path        string
	Summary     string
	Description string
	Tags        []string
	QueryParams []Param
	PathParams  []Param
	Body        *BodyDef
	Responses   []ResponseDef
	Auth        []string
	Cache       string
}

// ParseDir walks every .go file inside dir and returns all annotated functions found.
// Validation warnings are silently discarded; use ParseDirStrict to inspect them.
func ParseDir(dir string) ([]RouteAnnotation, error) {
	routes, _, err := ParseDirStrict(dir)
	return routes, err
}

// ParseDirStrict walks every .go file inside dir and returns annotated functions
// plus any ValidationErrors found in malformed annotations.
func ParseDirStrict(dir string) ([]RouteAnnotation, []ValidationError, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parser.ParseDir(%q): %w", dir, err)
	}

	var routes []RouteAnnotation
	var errs []ValidationError
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			anns, fileErrs := extractFromFileStrict(file)
			routes = append(routes, anns...)
			errs = append(errs, fileErrs...)
		}
	}
	return routes, errs, nil
}

// extractFromFileStrict pulls annotations from every top-level function declaration.
func extractFromFileStrict(f *ast.File) ([]RouteAnnotation, []ValidationError) {
	var out []RouteAnnotation
	var errs []ValidationError
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Doc == nil {
			continue
		}
		ann, found, fnErrs := parseDocComment(fn.Doc.Text())
		if !found {
			continue
		}
		ann.FuncName = fn.Name.Name
		for i := range fnErrs {
			fnErrs[i].FuncName = fn.Name.Name
		}
		out = append(out, ann)
		errs = append(errs, fnErrs...)
	}
	return out, errs
}

var validHTTPMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true,
	"DELETE": true, "HEAD": true, "OPTIONS": true,
}

// parseDocComment parses the raw comment text of a function.
// Returns (annotation, true, warnings) when at least one @Route tag is present.
func parseDocComment(doc string) (RouteAnnotation, bool, []ValidationError) {
	var ann RouteAnnotation
	var errs []ValidationError
	hasRoute := false

	for _, raw := range strings.Split(doc, "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "@") {
			continue
		}

		tag, rest, _ := strings.Cut(line, " ")
		rest = strings.TrimSpace(rest)

		switch tag {
		case "@Route":
			parts := strings.SplitN(rest, " ", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
				errs = append(errs, ValidationError{Tag: "@Route", Message: fmt.Sprintf("malformed %q — expected: METHOD /path", rest)})
				continue
			}
			method := strings.ToUpper(strings.TrimSpace(parts[0]))
			if !validHTTPMethods[method] {
				errs = append(errs, ValidationError{Tag: "@Route", Message: fmt.Sprintf("unknown HTTP method %q", method)})
			}
			ann.Method = method
			ann.Path = strings.TrimSpace(parts[1])
			hasRoute = true

		case "@Summary":
			ann.Summary = rest

		case "@Desc":
			ann.Description = rest

		case "@Tags":
			for _, t := range strings.Split(rest, ",") {
				if v := strings.TrimSpace(t); v != "" {
					ann.Tags = append(ann.Tags, v)
				}
			}

		case "@Query":
			if p, ok := parseParam(rest); ok {
				ann.QueryParams = append(ann.QueryParams, p)
			} else {
				errs = append(errs, ValidationError{Tag: "@Query", Message: fmt.Sprintf("malformed param %q — expected: name \"description\" required|optional", rest)})
			}

		case "@Path":
			if p, ok := parseParam(rest); ok {
				p.Required = true // path params are always required
				ann.PathParams = append(ann.PathParams, p)
			} else {
				errs = append(errs, ValidationError{Tag: "@Path", Message: fmt.Sprintf("malformed param %q — expected: name \"description\"", rest)})
			}

		case "@Body":
			ann.Body = parseBody(rest)

		case "@Response":
			if r, ok := parseResponse(rest); ok {
				ann.Responses = append(ann.Responses, r)
			} else {
				errs = append(errs, ValidationError{Tag: "@Response", Message: fmt.Sprintf("malformed %q — expected: code TypeName \"description\"", rest)})
			}

		case "@Auth":
			if rest != "" {
				ann.Auth = append(ann.Auth, rest)
			}

		case "@Cache":
			ann.Cache = rest

		case "@For":
			ann.Framework = strings.ToLower(strings.TrimSpace(rest))
		}
	}

	if ann.Framework == "" {
		ann.Framework = "nethttp"
	}
	return ann, hasRoute, errs
}

// parseParam parses:  name  "description"  required|optional
func parseParam(s string) (Param, bool) {
	name, rem, _ := strings.Cut(s, " ")
	name = strings.TrimSpace(name)
	if name == "" {
		return Param{}, false
	}
	desc, reqStr := extractQuoted(strings.TrimSpace(rem))
	return Param{
		Name:        name,
		Description: desc,
		Required:    strings.TrimSpace(reqStr) == "required",
	}, true
}

// parseBody parses:  "description"  required|optional
func parseBody(s string) *BodyDef {
	desc, reqStr := extractQuoted(s)
	return &BodyDef{
		Description: desc,
		Required:    strings.TrimSpace(reqStr) == "required",
	}
}

// parseResponse parses:  code  TypeName  "description"
func parseResponse(s string) (ResponseDef, bool) {
	codeStr, rem, _ := strings.Cut(strings.TrimSpace(s), " ")
	code, err := strconv.Atoi(strings.TrimSpace(codeStr))
	if err != nil {
		return ResponseDef{}, false
	}
	typeName, rem2, _ := strings.Cut(strings.TrimSpace(rem), " ")
	desc, _ := extractQuoted(strings.TrimSpace(rem2))
	return ResponseDef{Code: code, TypeName: strings.TrimSpace(typeName), Description: desc}, true
}

// extractQuoted returns (text-inside-quotes, remainder-after-closing-quote).
// If no quotes are found the whole string is returned as text with empty remainder.
func extractQuoted(s string) (string, string) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, `"`) {
		return s, ""
	}
	s = s[1:] // drop opening quote
	idx := strings.Index(s, `"`)
	if idx < 0 {
		return s, ""
	}
	return s[:idx], strings.TrimSpace(s[idx+1:])
}
