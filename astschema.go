package docwire

import (
	"go/ast"
	"reflect"
	"strconv"
	"strings"

	"github.com/ricksantos88/docwire/parser"
)

// typeIndex maps a bare struct type name (e.g. "UserResponse") to its source
// declaration, gathered across every scanned directory. It is the runtime,
// source-driven replacement for the hand-written responseResolver/bodyResolver
// switch maps: a @Response/@Body TypeName is resolved by looking the struct up
// here instead of being matched manually to a Go value.
type typeIndex struct {
	structs map[string]*ast.StructType
}

func newTypeIndex() *typeIndex {
	return &typeIndex{structs: make(map[string]*ast.StructType)}
}

// indexStructs records every top-level `type Name struct {...}` found in the file.
func (ix *typeIndex) indexStructs(f *ast.File) {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if st, ok := ts.Type.(*ast.StructType); ok {
				ix.structs[ts.Name.Name] = st
			}
		}
	}
}

// addRouteFromAST builds an Operation from a parsed annotation, resolving every
// referenced type name against the source index, and attaches it to the spec.
// bodyType is the request DTO inferred from the handler body (empty when none).
func (e *Engine) addRouteFromAST(r parser.RouteAnnotation, bodyType string, ix *typeIndex) {
	e.mu.Lock()
	defer e.mu.Unlock()

	responses := make(map[string]Response, len(r.Responses))
	for _, resp := range r.Responses {
		out := Response{Description: resp.Description}
		if prop, ok := e.astModelPropLocked(resp.TypeName, ix); ok {
			out.Content = map[string]MediaType{"application/json": {Schema: prop}}
		}
		responses[strconv.Itoa(resp.Code)] = out
	}
	if len(responses) == 0 {
		responses["200"] = Response{Description: "Successful Request Execution"}
	}

	var params []Parameter
	for _, q := range r.QueryParams {
		params = append(params, Parameter{
			Name: q.Name, In: "query", Description: q.Description,
			Required: q.Required, Schema: Property{Type: "string"},
		})
	}
	for _, p := range r.PathParams {
		params = append(params, Parameter{
			Name: p.Name, In: "path", Description: p.Description,
			Required: true, Schema: Property{Type: "string"},
		})
	}
	for _, h := range r.HeaderParams {
		params = append(params, Parameter{
			Name: h.Name, In: "header", Description: h.Description,
			Required: h.Required, Schema: Property{Type: "string"},
		})
	}

	var reqBody *RequestBody
	if r.Body != nil {
		var schema Property
		if prop, ok := e.astModelPropLocked(bodyType, ix); ok {
			schema = prop
		}
		reqBody = &RequestBody{
			Description: r.Body.Description,
			Required:    r.Body.Required,
			Content:     map[string]MediaType{"application/json": {Schema: schema}},
		}
	}

	var security []map[string][]string
	for _, scheme := range r.Auth {
		security = append(security, map[string][]string{scheme: {}})
	}

	e.applyOperationLocked(r.Path, r.Method, &Operation{
		Tags:        r.Tags,
		Summary:     r.Summary,
		Description: r.Description,
		Parameters:  params,
		RequestBody: reqBody,
		Responses:   responses,
		Security:    security,
	})
}

// astModelPropLocked resolves a type name to a Property ($ref, array, or scalar),
// registering struct schemas on demand. Returns ok=false for empty, "-", or
// unresolvable names so the caller can emit a body-less response/request.
func (e *Engine) astModelPropLocked(typeName string, ix *typeIndex) (Property, bool) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" || typeName == "-" {
		return Property{}, false
	}
	if elemType, ok := strings.CutPrefix(typeName, "[]"); ok {
		elemProp, ok := e.astModelPropLocked(elemType, ix)
		if !ok {
			return Property{}, false
		}
		return Property{Type: "array", Items: &elemProp}, true
	}
	if bt, ok := basicTypeOf(typeName); ok {
		return Property{Type: bt, Format: basicFormatOf(typeName)}, true
	}
	ref := e.registerStructFromASTLocked(typeName, ix)
	if ref == "" {
		return Property{}, false
	}
	return Property{Ref: ref}, true
}

// registerStructFromASTLocked adds a struct schema to components/schemas by
// reading its source declaration, mirroring the reflection path in schema.go.
// Returns the $ref, or "" when the type is not a known struct.
func (e *Engine) registerStructFromASTLocked(name string, ix *typeIndex) string {
	st, ok := ix.structs[name]
	if !ok {
		return ""
	}
	ref := "#/components/schemas/" + name
	if _, exists := e.spec.Components.Schemas[name]; exists {
		return ref
	}
	// placeholder first to break cycles in self-referential types
	e.spec.Components.Schemas[name] = Schema{Type: "object", Properties: make(map[string]Property)}

	schema := Schema{Type: "object", Properties: make(map[string]Property)}
	e.populateASTStructFieldsLocked(&schema, st, ix, make(map[string]bool))
	e.spec.Components.Schemas[name] = schema
	return ref
}

func (e *Engine) populateASTStructFieldsLocked(schema *Schema, st *ast.StructType, ix *typeIndex, visited map[string]bool) {
	for _, field := range st.Fields.List {
		tag := astFieldTag(field)
		jsonTag := tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		explicitName := strings.Split(jsonTag, ",")[0]

		if len(field.Names) == 0 {
			// Anonymous embedded field: flatten when no explicit json name is given.
			if explicitName != "" {
				prop := e.astPropertyLocked(field.Type, ix)
				applyTagOverrides(&prop, tag)
				schema.Properties[explicitName] = prop
				if astRequiredField(tag) {
					schema.Required = append(schema.Required, explicitName)
				}
				continue
			}
			embType := field.Type
			if star, ok := embType.(*ast.StarExpr); ok {
				embType = star.X
			}
			if id, ok := embType.(*ast.Ident); ok && !visited[id.Name] {
				if embSt, exists := ix.structs[id.Name]; exists {
					visited[id.Name] = true
					e.populateASTStructFieldsLocked(schema, embSt, ix, visited)
				}
			}
			continue
		}

		for _, ident := range field.Names {
			if !ident.IsExported() {
				continue
			}
			fieldName := explicitName
			if fieldName == "" {
				fieldName = ident.Name
			}
			prop := e.astPropertyLocked(field.Type, ix)
			applyTagOverrides(&prop, tag)
			schema.Properties[fieldName] = prop
			if astRequiredField(tag) {
				schema.Required = append(schema.Required, fieldName)
			}
		}
	}
}

// astPropertyLocked maps a source type expression to its OpenAPI Property,
// recursively registering nested struct types found in the index.
func (e *Engine) astPropertyLocked(expr ast.Expr, ix *typeIndex) Property {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return e.astPropertyLocked(t.X, ix)
	case *ast.ArrayType:
		items := e.astPropertyLocked(t.Elt, ix)
		return Property{Type: "array", Items: &items}
	case *ast.MapType:
		return Property{Type: "object"}
	case *ast.SelectorExpr:
		// qualified types from other packages, e.g. time.Time, uuid.UUID
		if t.Sel.Name == "Time" {
			return Property{Type: "string", Format: "date-time"}
		}
		return Property{Type: "string"}
	case *ast.Ident:
		if bt, ok := basicTypeOf(t.Name); ok {
			p := Property{Type: bt}
			if f := basicFormatOf(t.Name); f != "" {
				p.Format = f
			}
			return p
		}
		if _, ok := ix.structs[t.Name]; ok {
			return Property{Ref: e.registerStructFromASTLocked(t.Name, ix)}
		}
		return Property{Type: "string"} // named alias of unknown origin
	default:
		return Property{Type: "string"}
	}
}

// astFieldTag returns the struct tag of a field as a reflect.StructTag, so the
// same tag keys honored by the reflection path (json/example/format/enum/...)
// work identically here.
func astFieldTag(field *ast.Field) reflect.StructTag {
	if field.Tag == nil {
		return ""
	}
	if unquoted, err := strconv.Unquote(field.Tag.Value); err == nil {
		return reflect.StructTag(unquoted)
	}
	return reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
}

func applyTagOverrides(prop *Property, tag reflect.StructTag) {
	if v := tag.Get("description"); v != "" {
		prop.Description = v
	}
	if v := tag.Get("format"); v != "" {
		prop.Format = v
	}
	if v := tag.Get("example"); v != "" {
		prop.Example = v
	}
	if v := tag.Get("enum"); v != "" {
		prop.Enum = strings.Split(v, ",")
	}
}

// astRequiredField mirrors isRequiredField (schema.go) but reads the raw tag.
func astRequiredField(tag reflect.StructTag) bool {
	for _, key := range []string{"validate", "binding"} {
		for _, part := range strings.Split(tag.Get(key), ",") {
			if strings.TrimSpace(part) == "required" {
				return true
			}
		}
	}
	return tag.Get("required") == "true"
}

func basicTypeOf(name string) (string, bool) {
	switch name {
	case "string":
		return "string", true
	case "bool":
		return "boolean", true
	case "float32", "float64":
		return "number", true
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "byte", "rune":
		return "integer", true
	}
	return "", false
}

func basicFormatOf(name string) string {
	switch name {
	case "int32", "uint32", "rune":
		return "int32"
	case "int64", "uint64":
		return "int64"
	case "float32":
		return "float"
	case "float64":
		return "double"
	}
	return ""
}
