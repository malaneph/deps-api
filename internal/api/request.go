package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const maxBodyBytes = 1 << 20 // 1 MB

// Decode reads and JSON-decodes the request body into v.
// Unknown fields and bodies exceeding 1 MB are rejected.
// Returns an *APIError suitable for passing directly to Error().
func Decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		var maxBytesErr *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxErr):
			return ErrBadRequest(fmt.Sprintf("malformed JSON at position %d", syntaxErr.Offset))
		case errors.As(err, &typeErr):
			return ErrBadRequest(fmt.Sprintf("invalid value for field %q", typeErr.Field))
		case errors.As(err, &maxBytesErr):
			return ErrBadRequest("request body must not exceed 1 MB")
		case errors.Is(err, io.EOF):
			return ErrBadRequest("request body must not be empty")
		case errors.Is(err, io.ErrUnexpectedEOF):
			return ErrBadRequest("malformed JSON")
		default:

		}
	}

	return nil
}
