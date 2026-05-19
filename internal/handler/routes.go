package handler

import (
	"log/slog"
	"net/http"

	"deps-api/internal/api"
)

func Register(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := api.ErrNotFound("route not found")
		slog.Error("unhandled request", "method", r.Method, "path", r.URL.Path)
		api.HandleError(w, r, err)
	})
}
