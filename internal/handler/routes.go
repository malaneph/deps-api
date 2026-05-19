package handler

import (
	"log/slog"
	"net/http"

	"gorm.io/gorm"

	"deps-api/internal/api"
	"deps-api/internal/feature/department"
)

func Register(mux *http.ServeMux, db *gorm.DB) {
	department.Register(mux, db)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		slog.Error("unhandled request", "method", r.Method, "path", r.URL.Path)
		api.HandleError(w, r, api.ErrNotFound("route not found"))
	})
}
