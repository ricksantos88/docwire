package docwire

import (
	"reflect"
	"strings"
	"time"
)

// registerStructLocked adds a struct schema to components/schemas.
// Assumes write lock is held by caller.
func (e *Engine) registerStructLocked(t reflect.Type) string {
	name := t.Name()
	if name == "" {
		return ""
	}
	ref := "#/components/schemas/" + name
	if _, exists := e.spec.Components.Schemas[name]; exists {
		return ref
	}

	// placeholder to break cycles in self-referential types (e.g. Node { Children []Node })
	e.spec.Components.Schemas[name] = Schema{Type: "object", Properties: make(map[string]Property)}

	schema := Schema{Type: "object", Properties: make(map[string]Property)}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		fieldName := strings.Split(jsonTag, ",")[0]
		if fieldName == "" {
			fieldName = field.Name
		}
		schema.Properties[fieldName] = e.buildFieldPropertyLocked(field)
		if isRequiredField(field) {
			schema.Required = append(schema.Required, fieldName)
		}
	}
	e.spec.Components.Schemas[name] = schema
	return ref
}

// buildFieldPropertyLocked reads struct field tags and overrides what buildPropertyLocked inferred.
func (e *Engine) buildFieldPropertyLocked(field reflect.StructField) Property {
	prop := e.buildPropertyLocked(field.Type)
	if v := field.Tag.Get("description"); v != "" {
		prop.Description = v
	}
	if v := field.Tag.Get("format"); v != "" {
		prop.Format = v
	}
	if v := field.Tag.Get("example"); v != "" {
		prop.Example = v
	}
	if v := field.Tag.Get("enum"); v != "" {
		prop.Enum = strings.Split(v, ",")
	}
	return prop
}

// buildPropertyLocked maps any reflect.Type to its OpenAPI Property.
// Recursively registers nested structs — lock must be held by caller.
func (e *Engine) buildPropertyLocked(t reflect.Type) Property {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		elem := t.Elem()
		for elem.Kind() == reflect.Pointer {
			elem = elem.Elem()
		}
		items := e.buildPropertyLocked(elem)
		return Property{Type: "array", Items: &items}
	case reflect.Struct:
		if t == reflect.TypeOf(time.Time{}) {
			return Property{Type: "string", Format: "date-time"}
		}
		ref := e.registerStructLocked(t)
		return Property{Ref: ref}
	default:
		p := Property{Type: mapKindToType(t.Kind())}
		if f := mapKindToFormat(t); f != "" {
			p.Format = f
		}
		return p
	}
}

func mapKindToType(k reflect.Kind) string {
	switch k {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	default:
		return "string"
	}
}

func mapKindToFormat(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int32, reflect.Uint32:
		return "int32"
	case reflect.Int64, reflect.Uint64:
		return "int64"
	case reflect.Float32:
		return "float"
	case reflect.Float64:
		return "double"
	default:
		return ""
	}
}

// isRequiredField detects required via validate:"required", binding:"required", or required:"true".
func isRequiredField(f reflect.StructField) bool {
	for _, tagKey := range []string{"validate", "binding"} {
		for _, part := range strings.Split(f.Tag.Get(tagKey), ",") {
			if strings.TrimSpace(part) == "required" {
				return true
			}
		}
	}
	return f.Tag.Get("required") == "true"
}
