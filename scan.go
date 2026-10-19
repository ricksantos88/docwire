package docwire

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"io/fs"
	"strings"

	"github.com/ricksantos88/docwire/parser"
)

// Scan builds a documentation Engine directly from Go source, with zero
// hand-written configuration. It reads the given handler directories (and any
// directory holding the referenced DTOs), extracting:
//
//   - the global API metadata from a package doc comment (@Title/@Version/...);
//   - one route per annotated handler (@Route/@Summary/@Response/...);
//   - the JSON schema of every referenced type, resolved against the structs
//     declared in the same source — no reflection, no resolver maps.
//
// It is meant to run once at startup (gate it behind your "docs enabled" flag).
// Because it reads .go files, the source must be present at run time, which is
// the case under `go run`. Unparseable directories are skipped silently so a
// half-built tree never takes the service down; the resulting spec simply omits
// what could not be read.
//
//	func main() {
//	    h := docwire.Scan("./internal/interfaces/http").Handler()
//	    http.Handle("/docwire/", h)
//	}
func Scan(dirs ...string) *Engine {
	fset := token.NewFileSet()
	idx := newTypeIndex()

	var meta parser.APIMeta
	var routes []scannedRoute

	for _, dir := range dirs {
		pkgs, err := goparser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, goparser.ParseComments)
		if err != nil {
			continue // skip unreadable dirs rather than failing the whole boot
		}
		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				idx.indexStructs(file)
				if file.Doc != nil {
					if m, ok := parser.ParseMeta(file.Doc.Text()); ok {
						meta = mergeMeta(meta, m)
					}
				}
				routes = append(routes, scanFileRoutes(file)...)
			}
		}
	}

	e := newEngineFromMeta(meta)
	for _, r := range routes {
		e.addRouteFromAST(r.annotation, r.bodyType, idx)
	}
	return e
}

// scannedRoute pairs a parsed annotation with the request DTO inferred from the
// handler body, so the body schema is resolved per handler — not by parse order.
type scannedRoute struct {
	annotation parser.RouteAnnotation
	bodyType   string
}

// scanFileRoutes extracts every annotated handler in a file, inferring the body
// DTO from the function body when an @Body annotation is present.
func scanFileRoutes(file *ast.File) []scannedRoute {
	var out []scannedRoute
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Doc == nil {
			continue
		}
		ann, found := parser.ParseComment(fn.Doc.Text())
		if !found {
			continue
		}
		ann.FuncName = fn.Name.Name
		sr := scannedRoute{annotation: ann}
		if ann.Body != nil {
			if ann.Body.TypeName != "" {
				sr.bodyType = ann.Body.TypeName
			} else {
				sr.bodyType = inferBodyType(fn)
			}
		}
		out = append(out, sr)
	}
	return out
}

// inferBodyType returns the type name of the first non-primitive locally-declared
// variable (`var req T` or `req := T{}`) inside the handler — the idiomatic decode
// target (e.g. `var req RegisterRequest`). Empty when no candidate declaration exists.
func inferBodyType(fn *ast.FuncDecl) string {
	if fn.Body == nil {
		return ""
	}
	var name string
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if name != "" {
			return false
		}
		switch stmt := n.(type) {
		case *ast.GenDecl:
			if stmt.Tok != token.VAR {
				return true
			}
			for _, spec := range stmt.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || vs.Type == nil {
					continue
				}
				if candidate := candidateIdentName(vs.Type); candidate != "" {
					name = candidate
					return false
				}
			}
		case *ast.AssignStmt:
			if stmt.Tok != token.DEFINE {
				return true
			}
			for _, rhs := range stmt.Rhs {
				if candidate := candidateCompositeTypeName(rhs); candidate != "" {
					name = candidate
					return false
				}
			}
		}
		return true
	})
	return name
}

func candidateIdentName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	id, ok := expr.(*ast.Ident)
	if !ok || id.Name == "error" || id.Name == "any" {
		return ""
	}
	if _, isBasic := basicTypeOf(id.Name); isBasic {
		return ""
	}
	return id.Name
}

func candidateCompositeTypeName(expr ast.Expr) string {
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		expr = unary.X
	}
	cl, ok := expr.(*ast.CompositeLit)
	if !ok || cl.Type == nil {
		return ""
	}
	return candidateIdentName(cl.Type)
}

// newEngineFromMeta constructs an Engine from the scanned global metadata,
// applying sensible defaults when @Title/@Version are absent.
func newEngineFromMeta(meta parser.APIMeta) *Engine {
	title := meta.Title
	if title == "" {
		title = "API"
	}
	version := meta.Version
	if version == "" {
		version = "v0.0.0"
	}

	opts := []EngineOption{WithDescription(meta.Description)}
	for _, s := range meta.Servers {
		opts = append(opts, WithServer(s.URL, s.Description))
	}
	for _, s := range meta.Securities {
		if scheme, ok := securitySchemeFromMeta(s); ok {
			opts = append(opts, WithSecurityScheme(s.Name, scheme))
		}
	}
	return NewEngine(title, version, opts...)
}

// securitySchemeFromMeta maps a @Security type token to a SecurityScheme.
func securitySchemeFromMeta(s parser.SecurityMeta) (SecurityScheme, bool) {
	switch s.Type {
	case "bearerJWT":
		return BearerJWT(), true
	case "basic":
		return BasicAuth(), true
	case "apiKeyHeader":
		return APIKeyHeader(s.Arg), true
	case "apiKeyQuery":
		return APIKeyQuery(s.Arg), true
	}
	return SecurityScheme{}, false
}

// mergeMeta folds a later package's metadata into the accumulated one.
// Scalars are taken from the first package that sets them; lists accumulate.
func mergeMeta(dst, src parser.APIMeta) parser.APIMeta {
	if dst.Title == "" {
		dst.Title = src.Title
	}
	if dst.Version == "" {
		dst.Version = src.Version
	}
	if dst.Description == "" {
		dst.Description = src.Description
	}
	dst.Servers = append(dst.Servers, src.Servers...)
	dst.Securities = append(dst.Securities, src.Securities...)
	return dst
}
