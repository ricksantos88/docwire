package main

import (
	"log"
	"net/http"

	"github.com/ricksantos88/docwire"
	"github.com/ricksantos88/docwire/adapters"
	"github.com/ricksantos88/docwire/example/handlers"
	"github.com/ricksantos88/docwire/parser"
)

func main() {
	engine := docwire.NewEngine("Financial Customer Core API", "v3.1.2",
		docwire.WithDescription("Manages customer accounts, tiers, and profile settings."),
		docwire.WithContact("Core API Team", "api@example.com", "https://example.com"),
		docwire.WithLicense("MIT", "https://opensource.org/licenses/MIT"),
		docwire.WithServer("http://localhost:8080", "Local Development"),
		docwire.WithSecurityScheme("bearer", docwire.BearerJWT()),
	)

	routes, err := parser.ParseDir("./example/handlers")
	if err != nil {
		log.Fatalf("parse handlers: %v", err)
	}

	mux := http.NewServeMux()

	adapters.Load(engine, routes, handlers.NetHTTPRegistry, responseResolver,
		func(method, path string, h http.HandlerFunc) {
			mux.HandleFunc(method+" "+path, h)
		},
	)

	mux.Handle("/docwire/", engine.Handler())

	log.Println("API:        http://localhost:8080/api/v1/customers")
	log.Println("Swagger UI: http://localhost:8080/docwire/")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func responseResolver(typeName string) any {
	switch typeName {
	case "CustomerResponse":
		return handlers.CustomerResponse{}
	case "[]CustomerResponse":
		return []handlers.CustomerResponse{}
	case "ErrorResponse":
		return handlers.ErrorResponse{}
	}
	return nil
}
