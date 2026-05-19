package handler_test

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"deps-api/internal/test"
)

func TestUnknownRoute(t *testing.T) {
	db := test.DB(t)
	srv := test.NewServer(t, db)

	t.Run("returns 404", func(t *testing.T) {
		resp := test.Do(t, srv, http.MethodGet, "/no-such-route", nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("logs an http error", func(t *testing.T) {
		spy := test.NewLogSpy(t)
		resp := test.Do(t, srv, http.MethodGet, "/no-such-route", nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.True(t, spy.HasLevel(slog.LevelError), "expected an error-level log")
		require.True(t, spy.HasMessage("http error"), "expected 'http error' log message")
	})
}
