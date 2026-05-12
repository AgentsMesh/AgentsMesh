package main

import (
	"net/http"
	"strings"
)

// connectPathPrefix is the Connect-RPC canonical URL prefix —
// `/<package>.<Service>/`. Any incoming request whose URL.Path starts
// with `/proto.` is routed to the Connect mux before the Gin REST
// router gets a look at it, so adding new Connect services is purely
// additive against the existing REST surface.
const connectPathPrefix = "/proto."

// wrapWithConnect returns a top-level handler that prefers Connect for
// `/proto.*` paths and falls through to the Gin REST router for
// everything else. Service handlers are registered onto connectMux by
// per-service mount calls added as services migrate off REST; the REST
// router is untouched.
func wrapWithConnect(rest http.Handler) http.Handler {
	connectMux := http.NewServeMux()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, connectPathPrefix) {
			connectMux.ServeHTTP(w, r)
			return
		}
		rest.ServeHTTP(w, r)
	})
}
