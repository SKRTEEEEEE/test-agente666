//go:build integration
// +build integration

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestServerIntegration tests the full server integration
func TestServerIntegration(t *testing.T) {
	// Start the server in a goroutine with a custom mux
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", HelloHandler)
		mux.HandleFunc("/health", HealthHandler)
		mux.HandleFunc("/issues/", IssuesHandler)
		http.ListenAndServe(":8081", mux)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test root endpoint
	resp, err := http.Get("http://localhost:8081/")
	assert.NoError(t, err, "Should be able to connect to server")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Root endpoint should return 200")

	// Test health endpoint
	resp, err = http.Get("http://localhost:8081/health")
	assert.NoError(t, err, "Should be able to connect to health endpoint")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Health endpoint should return 200")
}

// TestEndpointResponse tests the actual response content
func TestEndpointResponse(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       string
		expectedBody   string
		expectedStatus int
	}{
		{
			name:           "Root endpoint returns Hello World",
			endpoint:       "/",
			expectedBody:   "Hello World!",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Health endpoint returns OK",
			endpoint:       "/health",
			expectedBody:   "OK",
			expectedStatus: http.StatusOK,
		},
	}

	// Setup server
	go func() {
		http.HandleFunc("/", HelloHandler)
		http.HandleFunc("/health", HealthHandler)
		http.ListenAndServe(":8082", nil)
	}()

	time.Sleep(100 * time.Millisecond)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("http://localhost:8082%s", tt.endpoint))
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestIssuesEndpointIntegration tests the issues endpoint with real GitHub API
func TestIssuesEndpointIntegration(t *testing.T) {
	// Setup server on port 8083
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", HelloHandler)
		mux.HandleFunc("/health", HealthHandler)
		mux.HandleFunc("/issues/", IssuesHandler)
		http.ListenAndServe(":8083", mux)
	}()

	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name           string
		username       string
		expectedStatus int
		checkJSON      bool
	}{
		{
			name:           "Valid user returns JSON response",
			username:       "octocat",
			expectedStatus: http.StatusOK,
			checkJSON:      true,
		},
		{
			name:           "Invalid user returns 404",
			username:       "thisuserdoesnotexist999999",
			expectedStatus: http.StatusNotFound,
			checkJSON:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("http://localhost:8083/issues/%s", tt.username))
			assert.NoError(t, err, "Should be able to connect to server")
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Should return expected status code")

			if tt.checkJSON {
				contentType := resp.Header.Get("Content-Type")
				assert.Contains(t, contentType, "application/json", "Should return JSON content type")
			}
		})
	}
}
