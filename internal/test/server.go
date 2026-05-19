package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/gorm"

	"deps-api/internal/handler"
)

func NewServer(t *testing.T, database *gorm.DB) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	handler.Register(mux, database)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}
