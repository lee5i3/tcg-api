package graphql

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/graph-gophers/graphql-go/relay"

	"github.com/lee5i3/pokemon-invest/internal/catalog"
)

// GraphiQL is embedded (react + graphiql dist) so the service stays fully
// self-contained — the playground loads no CDN assets.
//
//go:embed graphiql
var graphiqlAssets embed.FS

// Mount registers the public GraphQL surface on mux:
//
//	POST /graphql    — queries
//	GET  /graphiql/  — embedded playground
func Mount(mux *http.ServeMux, cat *catalog.Catalog) {
	schema := NewSchema(cat)
	mux.Handle("POST /graphql", &relay.Handler{Schema: schema})

	sub, err := fs.Sub(graphiqlAssets, "graphiql")
	if err != nil {
		panic(err) // embedded path is fixed at compile time
	}
	mux.Handle("GET /graphiql/", http.StripPrefix("/graphiql/", http.FileServerFS(sub)))
	mux.Handle("GET /graphiql", http.RedirectHandler("/graphiql/", http.StatusMovedPermanently))
}
