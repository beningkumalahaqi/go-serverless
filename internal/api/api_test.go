package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	largeBody := `{"message":"` + strings.Repeat("a", 2000) + `"}`

	tests := []struct {
		name          string
		method        string
		path          string
		body          string
		wantStatus    int
		wantContains  string
		wantErrorCode string
		wantAllow     string
	}{
		{
			name:         "health",
			method:       http.MethodGet,
			path:         "/health",
			wantStatus:   http.StatusOK,
			wantContains: `"status":"ok"`,
		},
		{
			name:         "hello defaults to World",
			method:       http.MethodGet,
			path:         "/hello",
			wantStatus:   http.StatusOK,
			wantContains: `"message":"Hello, World!"`,
		},
		{
			name:         "hello uses the name parameter",
			method:       http.MethodGet,
			path:         "/hello?name=Go",
			wantStatus:   http.StatusOK,
			wantContains: `"message":"Hello, Go!"`,
		},
		{
			name:         "hello with an empty name falls back",
			method:       http.MethodGet,
			path:         "/hello?name=",
			wantStatus:   http.StatusOK,
			wantContains: `"message":"Hello, World!"`,
		},
		{
			name:         "echo",
			method:       http.MethodPost,
			path:         "/echo",
			body:         `{"message":"ping"}`,
			wantStatus:   http.StatusOK,
			wantContains: `"message":"ping","receivedAt":"`,
		},
		{
			name:          "echo rejects malformed JSON",
			method:        http.MethodPost,
			path:          "/echo",
			body:          `{`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_request",
		},
		{
			name:          "echo rejects an empty body",
			method:        http.MethodPost,
			path:          "/echo",
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_request",
		},
		{
			name:          "echo rejects unknown fields",
			method:        http.MethodPost,
			path:          "/echo",
			body:          `{"message":"ping","extra":true}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_request",
		},
		{
			name:          "echo rejects a missing message",
			method:        http.MethodPost,
			path:          "/echo",
			body:          `{}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_request",
		},
		{
			name:          "echo rejects a blank message",
			method:        http.MethodPost,
			path:          "/echo",
			body:          `{"message":"   "}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_request",
		},
		{
			name:          "echo rejects trailing content",
			method:        http.MethodPost,
			path:          "/echo",
			body:          `{"message":"ping"}{"message":"pong"}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_request",
		},
		{
			name:          "echo rejects bodies over 1 KB",
			method:        http.MethodPost,
			path:          "/echo",
			body:          largeBody,
			wantStatus:    http.StatusRequestEntityTooLarge,
			wantErrorCode: "payload_too_large",
		},
		{
			name:          "unknown route",
			method:        http.MethodGet,
			path:          "/missing",
			wantStatus:    http.StatusNotFound,
			wantErrorCode: "not_found",
		},
		{
			name:          "wrong method on a GET route",
			method:        http.MethodPost,
			path:          "/health",
			wantStatus:    http.StatusMethodNotAllowed,
			wantErrorCode: "method_not_allowed",
			wantAllow:     "GET, HEAD",
		},
		{
			name:          "wrong method on the POST route",
			method:        http.MethodGet,
			path:          "/echo",
			wantStatus:    http.StatusMethodNotAllowed,
			wantErrorCode: "method_not_allowed",
			wantAllow:     "POST",
		},
		{
			name:       "HEAD on a GET route",
			method:     http.MethodHead,
			path:       "/health",
			wantStatus: http.StatusOK,
		},
		{
			name:         "swagger UI",
			method:       http.MethodGet,
			path:         "/docs",
			wantStatus:   http.StatusOK,
			wantContains: "SwaggerUIBundle",
		},
		{
			name:         "openapi document",
			method:       http.MethodGet,
			path:         "/openapi.yaml",
			wantStatus:   http.StatusOK,
			wantContains: "openapi: 3.0",
		},
	}

	router := NewRouter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.path, body)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantContains != "" && !strings.Contains(rec.Body.String(), tt.wantContains) {
				t.Errorf("body %q does not contain %q", rec.Body.String(), tt.wantContains)
			}
			if tt.wantAllow != "" {
				if got := rec.Header().Get("Allow"); got != tt.wantAllow {
					t.Errorf("Allow = %q, want %q", got, tt.wantAllow)
				}
			}
			if tt.wantErrorCode != "" {
				var got errorBody
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("decode error body: %v (body: %s)", err, rec.Body.String())
				}
				if got.Error.Code != tt.wantErrorCode {
					t.Errorf("error code = %q, want %q", got.Error.Code, tt.wantErrorCode)
				}
				if got.Error.Message == "" {
					t.Error("error message is empty")
				}
			}
		})
	}
}

func TestEchoTimestampIsUTC(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(`{"message":"ping"}`))
	NewRouter().ServeHTTP(rec, req)

	var got echoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ReceivedAt.IsZero() || got.ReceivedAt.Location().String() != "UTC" {
		t.Errorf("receivedAt = %v, want a non-zero UTC time", got.ReceivedAt)
	}
}

func TestDocsContentTypes(t *testing.T) {
	tests := map[string]string{
		"/docs":         "text/html; charset=utf-8",
		"/openapi.yaml": "application/yaml; charset=utf-8",
	}
	router := NewRouter()
	for path, want := range tests {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if got := rec.Header().Get("Content-Type"); got != want {
			t.Errorf("%s Content-Type = %q, want %q", path, got, want)
		}
	}
}
