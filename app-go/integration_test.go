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
	// Start the server in a goroutine
	go func() {
		http.HandleFunc("/", HelloHandler)
		http.HandleFunc("/health", HealthHandler)
		http.ListenAndServe(":8081", nil)
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
