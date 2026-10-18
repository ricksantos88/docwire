package parser

import "strings"

// ServerMeta describes one @Server entry from the package-level doc comment.
type ServerMeta struct {
	URL         string
	Description string
}

// SecurityMeta describes one @Security entry from the package-level doc comment.
//
//	@Security bearer  bearerJWT
//	@Security apikey  apiKeyHeader  X-API-Key
//	@Security basic   basic
type SecurityMeta struct {
	Name string // the scheme name referenced by @Auth in handlers
	Type string // bearerJWT | basic | apiKeyHeader | apiKeyQuery
	Arg  string // header/query param name for the apiKey* types
}

// APIMeta holds the global API metadata extracted from a package doc comment.
// It is the annotation-driven replacement for the programmatic NewEngine(...) +
// WithServer/WithSecurityScheme options, so the whole spec can be derived from
// source with zero hand-written configuration.
type APIMeta struct {
	Title       string
	Version     string
	Description string
	Servers     []ServerMeta
	Securities  []SecurityMeta
}

// ParseMeta extracts global API metadata from a package doc comment.
// Returns (meta, true) when at least one @Title or @Version tag is present.
//
//	// @Title       Customer Hub Support API
//	// @Version     v1.0.0
//	// @Description API REST de suporte.
//	// @Server      http://localhost:8080  "Servidor"
//	// @Security    bearer  bearerJWT
//	package main
func ParseMeta(doc string) (APIMeta, bool) {
	var meta APIMeta
	found := false

	for _, raw := range strings.Split(doc, "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "@") {
			continue
		}
		tag, rest, _ := strings.Cut(line, " ")
		rest = strings.TrimSpace(rest)

		switch tag {
		case "@Title":
			meta.Title = rest
			found = true
		case "@Version":
			meta.Version = rest
			found = true
		case "@Description":
			meta.Description = rest
		case "@Server":
			url, rem, _ := strings.Cut(rest, " ")
			desc, _ := extractQuoted(strings.TrimSpace(rem))
			if url = strings.TrimSpace(url); url != "" {
				meta.Servers = append(meta.Servers, ServerMeta{URL: url, Description: desc})
			}
		case "@Security":
			fields := strings.Fields(rest)
			if len(fields) < 2 {
				continue
			}
			s := SecurityMeta{Name: fields[0], Type: fields[1]}
			if len(fields) >= 3 {
				s.Arg = fields[2]
			}
			meta.Securities = append(meta.Securities, s)
		}
	}
	return meta, found
}
