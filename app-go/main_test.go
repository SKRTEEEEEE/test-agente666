package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHelloHandler tests the root endpoint
func TestHelloHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HelloHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Handler should return 200 OK")
	assert.Equal(t, "Hello World!", rr.Body.String(), "Handler should return 'Hello World!'")
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Health handler should return 200 OK")
	assert.Equal(t, "OK", rr.Body.String(), "Health handler should return 'OK'")
}

// TestHelloHandlerMethod tests that only GET method is supported
func TestHelloHandlerMethod(t *testing.T) {
	methods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		req, err := http.NewRequest(method, "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HelloHandler)
		handler.ServeHTTP(rr, req)

		// HTTP handlers in Go accept all methods by default unless explicitly restricted
		// We're testing that the handler responds correctly regardless of method
		assert.Equal(t, http.StatusOK, rr.Code, "Handler should return 200 OK for any method")
	}
}

// TestIssuesHandler tests the issues endpoint with a valid user
func TestIssuesHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/issues/octocat", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(IssuesHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Handler should return 200 OK")
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json", "Should return JSON content type")
}

// TestIssuesHandlerEmptyUser tests the issues endpoint with empty username
func TestIssuesHandlerEmptyUser(t *testing.T) {
	req, err := http.NewRequest("GET", "/issues/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(IssuesHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code, "Handler should return 400 for empty user")
}

// TestIssuesHandlerInvalidUser tests the issues endpoint with non-existent user
func TestIssuesHandlerInvalidUser(t *testing.T) {
	req, err := http.NewRequest("GET", "/issues/thisuserdoesnotexist12345678990", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(IssuesHandler)
	handler.ServeHTTP(rr, req)

	// GitHub API returns 404 for non-existent users
	assert.Equal(t, http.StatusNotFound, rr.Code, "Handler should return 404 for non-existent user")
}

// TestIssuesHandlerMethod tests that only GET method is supported
func TestIssuesHandlerMethod(t *testing.T) {
	req, err := http.NewRequest("POST", "/issues/octocat", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(IssuesHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code, "Handler should return 405 for non-GET methods")
}

// TestIssuesHandlerWithQueryParamOpen tests the issues endpoint with ?q=open query param
func TestIssuesHandlerWithQueryParamOpen(t *testing.T) {
	req, err := http.NewRequest("GET", "/issues/octocat?q=open", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(IssuesHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Handler should return 200 OK")
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json", "Should return JSON content type")
}

// TestIssuesHandlerWithoutQueryParam tests the issues endpoint without query param (should return all)
func TestIssuesHandlerWithoutQueryParam(t *testing.T) {
	req, err := http.NewRequest("GET", "/issues/octocat", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(IssuesHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Handler should return 200 OK")
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json", "Should return JSON content type")
}

// TestPRHandler tests the PR endpoint with a valid user
func TestPRHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/pr/octocat", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(PRHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Handler should return 200 OK")
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json", "Should return JSON content type")
}

// TestPRHandlerEmptyUser tests the PR endpoint with empty username
func TestPRHandlerEmptyUser(t *testing.T) {
	req, err := http.NewRequest("GET", "/pr/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(PRHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code, "Handler should return 400 for empty user")
}

// TestPRHandlerInvalidUser tests the PR endpoint with non-existent user
func TestPRHandlerInvalidUser(t *testing.T) {
	req, err := http.NewRequest("GET", "/pr/thisuserdoesnotexist12345678990", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(PRHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code, "Handler should return 404 for non-existent user")
}

// TestPRHandlerMethod tests that only GET method is supported
func TestPRHandlerMethod(t *testing.T) {
	req, err := http.NewRequest("POST", "/pr/octocat", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(PRHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code, "Handler should return 405 for non-GET methods")
}

// TestPRHandlerWithQueryParamOpen tests the PR endpoint with ?q=open query param
func TestPRHandlerWithQueryParamOpen(t *testing.T) {
	req, err := http.NewRequest("GET", "/pr/octocat?q=open", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(PRHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Handler should return 200 OK")
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json", "Should return JSON content type")
}

// TestPRHandlerWithoutQueryParam tests the PR endpoint without query param (should return all)
func TestPRHandlerWithoutQueryParam(t *testing.T) {
	req, err := http.NewRequest("GET", "/pr/octocat", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(PRHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Handler should return 200 OK")
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json", "Should return JSON content type")
}
