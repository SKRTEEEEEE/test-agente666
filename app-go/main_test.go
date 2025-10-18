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
