package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultMaxBodyBytes caps request bodies when a caller passes a limit that is
// zero or negative.
const DefaultMaxBodyBytes int64 = 1 << 20

// DecodeJSON decodes exactly one JSON object from the request body into dst. It
// caps the body at maxBytes, rejects unknown fields and trailing content, and
// returns *APIError values that Handle can write directly. A non-positive
// maxBytes uses DefaultMaxBodyBytes.
func DecodeJSON(r *http.Request, dst any, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}
	if r.Body == nil {
		return BadRequest("body is required")
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBytes+1))
	if err != nil {
		return Internal(fmt.Errorf("read request body: %w", err))
	}
	if int64(len(body)) > maxBytes {
		return PayloadTooLarge(maxBytes)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return BadRequest("body is required")
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return BadRequest("body must be valid JSON")
	}
	// Decode stops after the first value, so a second one would otherwise pass.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return BadRequest("body must contain exactly one JSON object")
	}
	return nil
}

// Query returns a trimmed query parameter, or def when it is blank.
func Query(r *http.Request, name, def string) string {
	if value := strings.TrimSpace(r.URL.Query().Get(name)); value != "" {
		return value
	}
	return def
}

// RequiredQuery returns a trimmed query parameter, or a 400 when it is blank.
func RequiredQuery(r *http.Request, name string) (string, error) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return "", BadRequest(name + " is required")
	}
	return value, nil
}
