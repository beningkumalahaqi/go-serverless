package core_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	handler "haqi/go-serverless/api"
	core "haqi/go-serverless/core"
)

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type echoResponse struct {
	Message    string    `json:"message"`
	ReceivedAt time.Time `json:"receivedAt"`
}

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
			path:         "/api/health",
			wantStatus:   http.StatusOK,
			wantContains: `"status":"ok"`,
		},
		{
			name:         "hello defaults to World",
			method:       http.MethodGet,
			path:         "/api/hello",
			wantStatus:   http.StatusOK,
			wantContains: `"message":"Hello, World!"`,
		},
		{
			name:         "hello uses the name parameter",
			method:       http.MethodGet,
			path:         "/api/hello?name=Go",
			wantStatus:   http.StatusOK,
			wantContains: `"message":"Hello, Go!"`,
		},
		{
			name:         "hello with an empty name falls back",
			method:       http.MethodGet,
			path:         "/api/hello?name=",
			wantStatus:   http.StatusOK,
			wantContains: `"message":"Hello, World!"`,
		},
		{
			name:         "echo",
			method:       http.MethodPost,
			path:         "/api/echo",
			body:         `{"message":"ping"}`,
			wantStatus:   http.StatusOK,
			wantContains: `"message":"ping","receivedAt":"`,
		},
		{
			name:          "echo rejects malformed JSON",
			method:        http.MethodPost,
			path:          "/api/echo",
			body:          `{`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_request",
		},
		{
			name:          "echo rejects an empty body",
			method:        http.MethodPost,
			path:          "/api/echo",
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_request",
		},
		{
			name:          "echo rejects unknown fields",
			method:        http.MethodPost,
			path:          "/api/echo",
			body:          `{"message":"ping","extra":true}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_request",
		},
		{
			name:          "echo rejects a missing message",
			method:        http.MethodPost,
			path:          "/api/echo",
			body:          `{}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_request",
		},
		{
			name:          "echo rejects a blank message",
			method:        http.MethodPost,
			path:          "/api/echo",
			body:          `{"message":"   "}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_request",
		},
		{
			name:          "echo rejects trailing content",
			method:        http.MethodPost,
			path:          "/api/echo",
			body:          `{"message":"ping"}{"message":"pong"}`,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_request",
		},
		{
			name:          "echo rejects bodies over 1 KB",
			method:        http.MethodPost,
			path:          "/api/echo",
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
			path:          "/api/health",
			wantStatus:    http.StatusMethodNotAllowed,
			wantErrorCode: "method_not_allowed",
			wantAllow:     "GET, HEAD",
		},
		{
			name:          "wrong method on the POST route",
			method:        http.MethodGet,
			path:          "/api/echo",
			wantStatus:    http.StatusMethodNotAllowed,
			wantErrorCode: "method_not_allowed",
			wantAllow:     "POST",
		},
		{
			name:       "HEAD on a GET route",
			method:     http.MethodHead,
			path:       "/api/health",
			wantStatus: http.StatusOK,
		},
		{
			name:         "swagger UI",
			method:       http.MethodGet,
			path:         "/api/docs",
			wantStatus:   http.StatusOK,
			wantContains: "SwaggerUIBundle",
		},
		{
			name:         "openapi document",
			method:       http.MethodGet,
			path:         "/api/openapi.yaml",
			wantStatus:   http.StatusOK,
			wantContains: "openapi: 3.0",
		},
	}

	router := core.NewRouter()
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

// A deployed function receives every method sent to its path, so each handler
// must reject the wrong method itself.
func TestHandlersGuardMethods(t *testing.T) {
	tests := []struct {
		name   string
		fn     http.HandlerFunc
		method string
		allow  string
	}{
		{"health", handler.Health, http.MethodPost, "GET, HEAD"},
		{"hello", handler.Hello, http.MethodPost, "GET, HEAD"},
		{"echo", handler.Echo, http.MethodGet, "POST"},
		{"docs", handler.Docs, http.MethodPost, "GET, HEAD"},
		{"openapi", handler.OpenAPI, http.MethodPost, "GET, HEAD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/api/"+tt.name, nil)

			tt.fn(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}
			if got := rec.Header().Get("Allow"); got != tt.allow {
				t.Errorf("Allow = %q, want %q", got, tt.allow)
			}
			var got errorBody
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode error body: %v", err)
			}
			if got.Error.Code != "method_not_allowed" {
				t.Errorf("error code = %q, want method_not_allowed", got.Error.Code)
			}
		})
	}
}

func TestEchoTimestampIsUTC(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/echo", strings.NewReader(`{"message":"ping"}`))
	core.NewRouter().ServeHTTP(rec, req)

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
		"/api/docs":         "text/html; charset=utf-8",
		"/api/openapi.yaml": "application/yaml; charset=utf-8",
	}
	router := core.NewRouter()
	for path, want := range tests {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if got := rec.Header().Get("Content-Type"); got != want {
			t.Errorf("%s Content-Type = %q, want %q", path, got, want)
		}
	}
}
