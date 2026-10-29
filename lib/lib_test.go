package lib_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"haqi/go-serverless/lib"
)

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestHandleWrapsSuccess(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/thing", nil)

	lib.Handle(rec, req, func(*http.Request) (any, error) {
		return map[string]string{"id": "abc123"}, nil
	}, http.MethodGet)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", got)
	}
	var body struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, rec.Body.String())
	}
	if body.Data["id"] != "abc123" {
		t.Errorf("data = %v, want id abc123", body.Data)
	}
}

func TestHandleGuardsMethods(t *testing.T) {
	called := false
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/thing", nil)

	lib.Handle(rec, req, func(*http.Request) (any, error) {
		called = true
		return nil, nil
	}, http.MethodGet, http.MethodHead)

	if called {
		t.Error("handler ran for a method it does not accept")
	}
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("Allow"); got != "GET, HEAD" {
		t.Errorf("Allow = %q, want GET, HEAD", got)
	}
	assertErrorCode(t, rec, "method_not_allowed")
}

func TestHandleAPIError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/thing", nil)

	lib.Handle(rec, req, func(*http.Request) (any, error) {
		return nil, lib.BadRequest("thing is required")
	}, http.MethodGet)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	assertErrorCode(t, rec, "invalid_request")
	if !strings.Contains(rec.Body.String(), `"message":"thing is required"`) {
		t.Errorf("body = %s, want the handler message", rec.Body.String())
	}
}

// An error that is not an *APIError must not leak its text to the client.
func TestHandlePlainErrorBecomes500(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/thing", nil)

	lib.Handle(rec, req, func(*http.Request) (any, error) {
		return nil, errors.New("database password rejected")
	}, http.MethodGet)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	assertErrorCode(t, rec, "internal_error")
	if strings.Contains(rec.Body.String(), "database password") {
		t.Errorf("internal error leaked to the client: %s", rec.Body.String())
	}
}

func TestHandleRecoversPanics(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/thing", nil)

	lib.Handle(rec, req, func(*http.Request) (any, error) {
		panic("boom")
	}, http.MethodGet)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	assertErrorCode(t, rec, "internal_error")
	if strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("panic value leaked to the client: %s", rec.Body.String())
	}
}

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	tests := []struct {
		name       string
		body       string
		limit      int64
		wantStatus int
		wantCode   string
	}{
		{name: "valid", body: `{"name":"go"}`, wantStatus: http.StatusOK},
		{name: "empty body", body: "", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "blank body", body: "  \n", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "malformed", body: `{`, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "unknown field", body: `{"name":"go","extra":true}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "trailing value", body: `{"name":"go"}{"name":"go"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "over the limit", body: `{"name":"toolong"}`, limit: 8, wantStatus: http.StatusRequestEntityTooLarge, wantCode: "payload_too_large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/thing", strings.NewReader(tt.body))

			lib.Handle(rec, req, func(r *http.Request) (any, error) {
				var p payload
				if err := lib.DecodeJSON(r, &p, tt.limit); err != nil {
					return nil, err
				}
				return p, nil
			}, http.MethodPost)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantCode != "" {
				assertErrorCode(t, rec, tt.wantCode)
			}
		})
	}
}

func TestQueryHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/thing?name=%20Go%20", nil)

	if got := lib.Query(req, "name", "World"); got != "Go" {
		t.Errorf("Query = %q, want Go", got)
	}
	if got := lib.Query(req, "missing", "World"); got != "World" {
		t.Errorf("Query fallback = %q, want World", got)
	}

	if got, err := lib.RequiredQuery(req, "name"); err != nil || got != "Go" {
		t.Errorf("RequiredQuery = %q, %v, want Go, nil", got, err)
	}
	if _, err := lib.RequiredQuery(req, "missing"); err == nil {
		t.Error("RequiredQuery accepted a missing parameter")
	} else {
		var apiErr *lib.APIError
		if !errors.As(err, &apiErr) || apiErr.Code != lib.CodeInvalidRequest {
			t.Errorf("RequiredQuery error = %v, want invalid_request", err)
		}
	}
}

func TestRespondStatuses(t *testing.T) {
	rec := httptest.NewRecorder()
	lib.Created(rec, map[string]string{"id": "abc"})
	if rec.Code != http.StatusCreated {
		t.Errorf("Created status = %d, want %d", rec.Code, http.StatusCreated)
	}

	rec = httptest.NewRecorder()
	lib.NoContent(rec)
	if rec.Code != http.StatusNoContent {
		t.Errorf("NoContent status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("NoContent body = %q, want empty", rec.Body.String())
	}
}

func TestHandleStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	lib.HandleStatus(rec, httptest.NewRequest(http.MethodPost, "/api/thing", nil),
		http.StatusCreated, func(*http.Request) (any, error) {
			return map[string]string{"id": "abc"}, nil
		}, http.MethodPost)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	rec = httptest.NewRecorder()
	lib.HandleStatus(rec, httptest.NewRequest(http.MethodDelete, "/api/thing", nil),
		http.StatusNoContent, func(*http.Request) (any, error) {
			return nil, nil
		}, http.MethodDelete)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("204 body = %q, want empty", rec.Body.String())
	}
}

func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v (body: %s)", err, rec.Body.String())
	}
	if body.Error.Code != want {
		t.Errorf("error code = %q, want %q", body.Error.Code, want)
	}
	if body.Error.Message == "" {
		t.Error("error message is empty")
	}
}
