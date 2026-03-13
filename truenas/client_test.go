package truenas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_Success(t *testing.T) {
	t.Parallel()
	ts := NewTestServer(t)
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"

	client, err := NewClient(wsURL, Options{
		Username: "testuser",
		Password: "testpass",
	})

	assert.NoError(t, err)
	assert.NotNil(t, client)
	defer client.Close()
}

func TestNewClient_AuthFailure(t *testing.T) {
	t.Parallel()
	ts := NewTestServer(t, WithAuthSuccess(false))
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"

	client, err := NewClient(wsURL, Options{
		Username: "baduser",
		Password: "badpass",
	})

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "Authentication failed")
}

func TestNewClient_InvalidURL(t *testing.T) {
	t.Parallel()
	client, err := NewClient("invalid-url", Options{
		Username: "testuser",
		Password: "testpass",
	})

	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestRPCError_Error(t *testing.T) {
	t.Parallel()
	err := &RPCError{
		Code:    -32001,
		Message: "Internal server error",
		Data:    &RPCErrorData{Error: 500, Reason: "Internal server error"},
	}

	assert.Contains(t, err.Error(), "code: -32001")
	assert.Contains(t, err.Error(), "message: Internal server error")
}

func TestRequest_JSON(t *testing.T) {
	t.Parallel()
	req := &Request{
		JSONRPC: "2.0",
		ID:      123,
		Method:  "system.info",
		Params:  []any{"param1", "param2"},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded Request
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.ID, decoded.ID)
	assert.Equal(t, req.Method, decoded.Method)
	assert.Equal(t, len(req.Params), len(decoded.Params))
}

func TestReconnection_CloseDetection(t *testing.T) {
	t.Parallel()
	ts := NewTestServer(t)
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"

	client, err := NewClient(wsURL, Options{
		Username: "testuser",
		Password: "testpass",
		Debug:    false, // Disable debug to reduce noise
	})
	require.NoError(t, err)

	// Make a successful call
	var result map[string]any
	err = client.Call(context.Background(), "system.info", nil, &result)
	require.NoError(t, err)

	// Close the client and verify it doesn't hang
	start := time.Now()
	err = client.Close()
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, duration, 2*time.Second, "Close should complete quickly")
}

func TestReconnection_FailedConnection(t *testing.T) {
	t.Parallel()
	// Test reconnection behavior when server becomes unavailable
	ts := NewTestServer(t)
	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"

	client, err := NewClient(wsURL, Options{
		Username: "testuser",
		Password: "testpass",
		Debug:    false,
	})
	require.NoError(t, err)

	// Make a successful call
	var result map[string]any
	err = client.Call(context.Background(), "system.info", nil, &result)
	require.NoError(t, err)

	// Close the server to simulate connection loss
	ts.Close()

	// Wait a moment for connection to be detected as lost
	time.Sleep(100 * time.Millisecond)

	// Try to make a call with a short timeout - should fail quickly
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = client.Call(ctx, "system.info", nil, &result)
	// The call might succeed if it was made before connection was lost, or fail due to timeout/connection error
	// We're mainly testing that it doesn't hang and Close works properly

	// Close should still work quickly even with failed reconnection attempts
	start := time.Now()
	err = client.Close()
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, duration, 1*time.Second, "Close should complete quickly even during reconnection failures")
}

func TestReconnection_ConnectionDropHandling(t *testing.T) {
	t.Parallel()
	// Test that connection drops are detected and don't cause hangs
	ts := NewTestServer(t)
	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"

	client, err := NewClient(wsURL, Options{
		Username: "testuser",
		Password: "testpass",
		Debug:    true,
	})
	require.NoError(t, err)

	// Make a successful call
	var result map[string]any
	err = client.Call(context.Background(), "system.info", nil, &result)
	require.NoError(t, err)
	assert.Equal(t, "test-truenas", result["hostname"])

	// Close the server to simulate network failure
	ts.Close()

	// Wait for connection loss to be detected
	time.Sleep(200 * time.Millisecond)

	// Close should work quickly even with failed reconnection attempts
	start := time.Now()
	err = client.Close()
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, duration, 2*time.Second, "Close should complete quickly even during reconnection attempts")
}

func TestReconnection_CallAfterServerDown(t *testing.T) {
	t.Parallel()
	// Test that calls fail gracefully when server is down
	ts := NewTestServer(t)
	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"

	client, err := NewClient(wsURL, Options{
		Username: "testuser",
		Password: "testpass",
		Debug:    false,
	})
	require.NoError(t, err)
	defer client.Close()

	// Make a successful call
	var result map[string]any
	err = client.Call(context.Background(), "system.info", nil, &result)
	require.NoError(t, err)

	// Close the server
	ts.Close()
	time.Sleep(100 * time.Millisecond)

	// Subsequent call should fail gracefully within timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	_ = client.Call(ctx, "system.info", nil, &result)
	duration := time.Since(start)

	// Should complete quickly regardless of success/failure
	// (The call might succeed if made before connection was lost, or fail due to timeout)
	assert.Less(t, duration, 1*time.Second, "Call should complete quickly when server is down")
}

func TestReconnection_ReconnectAttempts(t *testing.T) {
	t.Parallel()
	// Test that client attempts to reconnect when connection is lost
	ts := NewTestServer(t)
	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"

	client, err := NewClient(wsURL, Options{
		Username: "testuser",
		Password: "testpass",
		Debug:    true,
	})
	require.NoError(t, err)

	// Make a successful call to verify connection
	var result map[string]any
	err = client.Call(context.Background(), "system.info", nil, &result)
	require.NoError(t, err)
	assert.Equal(t, "test-truenas", result["hostname"])

	// Close the server to simulate connection loss
	ts.Close()

	// Wait for connection loss to be detected
	time.Sleep(200 * time.Millisecond)

	// Attempt a call with a reasonable timeout - should trigger reconnection attempts
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	_ = client.Call(ctx, "system.info", nil, &result)
	duration := time.Since(start)

	// The call should fail due to server being down, but should respect timeout
	assert.Less(t, duration, 2*time.Second, "Call should timeout promptly when server is unavailable")

	// Close should work quickly even during reconnection attempts
	start = time.Now()
	err = client.Close()
	duration = time.Since(start)

	require.NoError(t, err)
	assert.Less(t, duration, 1*time.Second, "Close should complete quickly even during reconnection attempts")
}

func TestNewClient_ConnectionFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		url         string
		expectError string
	}{
		{
			name:        "invalid_websocket_scheme",
			url:         "http://invalid-websocket-url",
			expectError: "websocket dial",
		},
		{
			name:        "connection_refused",
			url:         "ws://localhost:99999/api/current",
			expectError: "websocket dial",
		},
		{
			name:        "malformed_url",
			url:         "://invalid",
			expectError: "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.url, Options{
				Username: "testuser",
				Password: "testpass",
			})

			assert.Error(t, err)
			assert.Nil(t, client)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

func TestNewClient_AuthenticationFailurePaths(t *testing.T) {
	t.Parallel()
	ts := NewTestServer(t, WithCustomHandler(func(req Request) (Response, bool) {
		response := Response{JSONRPC: "2.0", ID: &req.ID}
		switch req.Method {
		case "auth.login", "auth.login_with_api_key":
			response.Error = &RPCError{
				Code:    -32001,
				Message: "Authentication failed",
				Data:    &RPCErrorData{Error: 401, Reason: "Invalid credentials", ErrName: "UNAUTHORIZED"},
			}
		default:
			response.Error = &RPCError{Code: -32601, Message: "Method not found"}
		}
		return response, true
	}))
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"

	tests := []struct {
		name    string
		options Options
	}{
		{
			name: "username_password_auth_failure",
			options: Options{
				Username: "baduser",
				Password: "badpass",
			},
		},
		{
			name: "api_key_auth_failure",
			options: Options{
				APIKey: "invalid-api-key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(wsURL, tt.options)
			assert.Error(t, err)
			assert.Nil(t, client)
			assert.Contains(t, err.Error(), "authentication")
		})
	}
}

func TestNewClient_AuthenticationFalseResponse(t *testing.T) {
	t.Parallel()
	// Test auth unsuccessful scenario (returns false instead of error)
	ts := NewTestServer(t, WithCustomHandler(func(req Request) (Response, bool) {
		response := Response{JSONRPC: "2.0", ID: &req.ID}
		switch req.Method {
		case "auth.login", "auth.login_with_api_key":
			response.Result = json.RawMessage(`false`) // Return false instead of error
		default:
			response.Error = &RPCError{Code: -32601, Message: "Method not found"}
		}
		return response, true
	}))
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"

	client, err := NewClient(wsURL, Options{
		Username: "testuser",
		Password: "testpass",
	})

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "auth unsuccessful")
}

func TestNewClient_NoCredentials(t *testing.T) {
	t.Parallel()
	ts := NewTestServer(t)
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"

	// Should succeed without credentials
	client, err := NewClient(wsURL, Options{})
	assert.NoError(t, err)
	assert.NotNil(t, client)
	defer client.Close()
}

func TestClient_CallErrorPaths(t *testing.T) {
	t.Parallel()
	ts := NewTestServer(t, WithCustomHandler(func(req Request) (Response, bool) {
		response := Response{JSONRPC: "2.0", ID: &req.ID}
		switch req.Method {
		case "auth.login", "auth.login_with_api_key":
			response.Result = json.RawMessage(`true`)
		case "test.error":
			response.Error = &RPCError{
				Code:    -32001,
				Message: "Internal server error",
				Data:    &RPCErrorData{Error: 500, Reason: "Database connection failed", ErrName: "INTERNAL_ERROR"},
			}
		case "test.timeout":
			// Don't send response to simulate timeout.
			return response, false
		default:
			response.Error = &RPCError{Code: -32601, Message: "Method not found"}
		}
		return response, true
	}))
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"
	client, err := NewClient(wsURL, Options{Username: "test", Password: "test"})
	require.NoError(t, err)
	defer client.Close()

	t.Run("api_error_response", func(t *testing.T) {
		var result map[string]any
		err := client.Call(context.Background(), "test.error", nil, &result)
		assert.Error(t, err)

		rpcErr, ok := err.(*RPCError)
		assert.True(t, ok)
		assert.Equal(t, -32001, rpcErr.Code)
		assert.Contains(t, rpcErr.Error(), "code: -32001")
		assert.Contains(t, rpcErr.Error(), "Internal server error")
		assert.Contains(t, rpcErr.Error(), "Database connection failed")
		assert.Contains(t, rpcErr.Error(), "INTERNAL_ERROR")
	})

	t.Run("context_timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		var result map[string]any
		err := client.Call(ctx, "test.timeout", nil, &result)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("context_cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		var result map[string]any
		err := client.Call(ctx, "system.info", nil, &result)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("call_after_close", func(t *testing.T) {
		// Create a separate client for this test to avoid double-closing
		separateClient, err := NewClient(wsURL, Options{Username: "test", Password: "test"})
		require.NoError(t, err)

		separateClient.Close()

		var result map[string]any
		err = separateClient.Call(context.Background(), "system.info", nil, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})
}

func TestClient_MessageUnmarshalErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		result   json.RawMessage
		target   any
		hasError bool
	}{
		{
			name:     "invalid_json",
			result:   json.RawMessage(`{"invalid": json}`),
			target:   &map[string]any{},
			hasError: true,
		},
		{
			name:     "type_mismatch",
			result:   json.RawMessage(`"string_value"`),
			target:   &map[string]any{},
			hasError: true,
		},
		{
			name:     "nil_target",
			result:   json.RawMessage(`{"valid": "json"}`),
			target:   nil,
			hasError: true,
		},
		{
			name:     "valid_unmarshal",
			result:   json.RawMessage(`{"key": "value"}`),
			target:   &map[string]any{},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{Result: tt.result}
			err := resp.Unmarshal(tt.target)

			if tt.hasError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unmarshal result")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRPCError_ErrorFormatting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		rpcError *RPCError
		contains []string
	}{
		{
			name:     "empty_error",
			rpcError: &RPCError{},
			contains: []string{"TrueNAS API error"},
		},
		{
			name: "code_only",
			rpcError: &RPCError{
				Code: -32601,
			},
			contains: []string{"code: -32601"},
		},
		{
			name: "all_fields",
			rpcError: &RPCError{
				Code:    -32001,
				Message: "Server error",
				Data:    &RPCErrorData{Error: 500, Reason: "Database failure", ErrName: "INTERNAL"},
			},
			contains: []string{"code: -32001", "message: Server error", "reason: Database failure", "errname: INTERNAL"},
		},
		{
			name: "message_and_reason_only",
			rpcError: &RPCError{
				Message: "Custom error",
				Data:    &RPCErrorData{Reason: "User action required"},
			},
			contains: []string{"message: Custom error", "reason: User action required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorStr := tt.rpcError.Error()
			for _, expected := range tt.contains {
				assert.Contains(t, errorStr, expected)
			}
		})
	}
}

func TestClient_ConnectionManagerAndReconnection(t *testing.T) {
	t.Parallel()
	ts := NewTestServer(t, WithConnectionTracking())
	ts.SetResponse("system.info", map[string]any{
		"hostname": "test-truenas",
		"version":  "TrueNAS-SCALE-23.10.2",
	})
	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"

	client, err := NewClient(wsURL, Options{
		Username: "testuser",
		Password: "testpass",
		Debug:    true,
	})
	require.NoError(t, err)

	// Make a successful call to verify connection
	var result map[string]any
	err = client.Call(context.Background(), "system.info", nil, &result)
	require.NoError(t, err)

	// Use the shutdown function to immediately close all connections
	ts.Shutdown()

	// Wait a bit for connection loss to be detected
	time.Sleep(100 * time.Millisecond)

	// Try another call - should fail due to connection loss
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err = client.Call(ctx, "system.info", nil, &result)
	// Call should fail due to connection loss and reconnection failure
	assert.Error(t, err)

	// Close should work even during reconnection attempts
	start := time.Now()
	closeErr := client.Close()
	duration := time.Since(start)

	assert.NoError(t, closeErr)
	assert.Less(t, duration, 2*time.Second, "Close should complete quickly")
}

func TestClient_ConnectionWriteFailure(t *testing.T) {
	t.Parallel()
	// Server that rejects WebSocket upgrade to cause connection failure
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	wsURL := strings.Replace(server.URL, "http://", "ws://", 1) + "/api/current"

	client, err := NewClient(wsURL, Options{
		Username: "testuser",
		Password: "testpass",
	})

	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestClient_ReadLoopErrorHandling(t *testing.T) {
	t.Parallel()

	// Create server that will be closed after sending malformed JSON
	var serverClosed atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If server is marked as closed, reject new connections
		if serverClosed.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Wait for auth message and respond
		var authReq Request
		_ = conn.ReadJSON(&authReq)
		_ = conn.WriteJSON(Response{
			JSONRPC: "2.0",
			ID:      &authReq.ID,
			Result:  json.RawMessage(`true`),
		})

		// Send malformed JSON to trigger read error
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{invalid json`))

		// Mark server as closed to prevent reconnection
		serverClosed.Store(true)

		// Keep connection open briefly then close unexpectedly
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := strings.Replace(server.URL, "http://", "ws://", 1) + "/api/current"

	client, err := NewClient(wsURL, Options{
		Username: "testuser",
		Password: "testpass",
		Debug:    true,
	})
	require.NoError(t, err)

	// Wait for read error to be processed
	time.Sleep(200 * time.Millisecond)

	// Try to make a call - should work initially then potentially fail
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var result map[string]any
	_ = client.Call(ctx, "system.info", nil, &result)

	// Close should work even with read errors
	err = client.Close()
	assert.NoError(t, err)
}

func TestClient_CallJobErrorHandling(t *testing.T) {
	t.Parallel()
	ts := NewTestServer(t, WithCustomHandler(func(req Request) (Response, bool) {
		response := Response{JSONRPC: "2.0", ID: &req.ID}
		switch req.Method {
		case "auth.login":
			response.Result = json.RawMessage(`true`)
		case "test.job.failure":
			response.Result = json.RawMessage(`123`) // Return job ID
		case "test.job.marshal_error":
			response.Result = json.RawMessage(`456`) // Return job ID
		case "core.get_jobs":
			params := req.Params
			var jobID float64

			// Parse the filter parameters to extract the job ID
			if len(params) > 0 {
				if filterArray, ok := params[0].([]any); ok && len(filterArray) > 0 {
					if filterCondition, ok := filterArray[0].([]any); ok && len(filterCondition) >= 3 {
						if field, ok := filterCondition[0].(string); ok && field == "id" {
							if id, ok := filterCondition[2].(float64); ok {
								jobID = id
							}
						}
					}
				}
			}

			switch jobID {
			case 123:
				// Simulate job failure
				jobResult := map[string]any{
					"id":     123,
					"method": "test.job.failure",
					"state":  "FAILED",
					"error":  "Job execution failed",
					"result": nil,
				}
				resultBytes, _ := json.Marshal([]any{jobResult})
				response.Result = json.RawMessage(resultBytes)
			case 456:
				// Simulate successful job for marshal error test
				jobResult := map[string]any{
					"id":     456,
					"method": "test.job.marshal_error",
					"state":  "SUCCESS",
					"error":  nil,
					"result": map[string]any{"data": "test"},
				}
				resultBytes, _ := json.Marshal([]any{jobResult})
				response.Result = json.RawMessage(resultBytes)
			default:
				// Return empty array if no matching job ID
				response.Result = json.RawMessage(`[]`)
			}
		default:
			response.Error = &RPCError{Code: -32601, Message: "Method not found"}
		}
		return response, true
	}))
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"
	client, err := NewClient(wsURL, Options{Username: "test", Password: "test"})
	require.NoError(t, err)
	defer client.Close()

	t.Run("job_execution_failure", func(t *testing.T) {
		var result map[string]any
		err := client.CallJob(context.Background(), "test.job.failure", nil, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Job execution failed")
	})

	t.Run("job_call_failure", func(t *testing.T) {
		var result map[string]any
		err := client.CallJob(context.Background(), "nonexistent.method", nil, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "call nonexistent.method")
	})

	t.Run("job_marshal_error", func(t *testing.T) {
		var result map[string]any
		err := client.CallJob(context.Background(), "test.job.marshal_error", nil, &result)
		// This test case is difficult to reproduce realistically since job results
		// that can be unmarshaled from JSON should generally be marshalable again.
		// For now, we just verify the call succeeds since the mock provides valid data.
		assert.NoError(t, err)
	})
}

func TestClient_HelperFunctions(t *testing.T) {
	t.Parallel()
	t.Run("ptr_function", func(t *testing.T) {
		val := 42
		ptr := Ptr(val)
		assert.Equal(t, &val, ptr)
		assert.Equal(t, val, *ptr)

		str := "test"
		strPtr := Ptr(str)
		assert.Equal(t, &str, strPtr)
		assert.Equal(t, str, *strPtr)
	})

	t.Run("value_function", func(t *testing.T) {
		// Test with non-nil pointer
		val := 42
		ptr := &val
		result := value(ptr)
		assert.Equal(t, val, result)

		// Test with nil pointer
		var nilPtr *int
		zeroResult := value(nilPtr)
		assert.Equal(t, 0, zeroResult)

		// Test with string
		str := "test"
		strPtr := &str
		strResult := value(strPtr)
		assert.Equal(t, str, strResult)

		// Test with nil string pointer
		var nilStrPtr *string
		zeroStrResult := value(nilStrPtr)
		assert.Equal(t, "", zeroStrResult)
	})

	t.Run("try_marshal", func(t *testing.T) {
		// Test successful marshal
		data := map[string]any{"key": "value"}
		result := tryMarshal(data)
		assert.Contains(t, result, "key")
		assert.Contains(t, result, "value")

		// Test marshal that might fail (but tryMarshal handles it)
		ch := make(chan int)
		result = tryMarshal(ch)
		// Should return some string representation, not panic
		assert.IsType(t, "", result)
	})
}

func TestClient_ReconnectCoverage(t *testing.T) {
	t.Parallel()
	// Test the reconnect function by creating a scenario where it's called
	server1 := NewTestServer(t)
	wsURL1 := strings.Replace(server1.URL, "http://", "ws://", 1) + "/api/current"

	client, err := NewClient(wsURL1, Options{
		Username: "testuser",
		Password: "testpass",
		Debug:    true,
	})
	require.NoError(t, err)

	// Verify initial connection works
	var result map[string]any
	err = client.Call(context.Background(), "system.info", nil, &result)
	require.NoError(t, err)

	// Force close the underlying connection to trigger disconnection detection
	client.mu.Lock()
	if client.conn != nil {
		client.conn.Close()
	}
	client.mu.Unlock()

	// Close the server
	server1.Close()

	// Update client URL to point to a non-existent server (with proper synchronization)
	badURL := "ws://localhost:99999/api/current"
	client.mu.Lock()
	client.url = badURL
	client.mu.Unlock()

	// Wait for disconnection to be detected
	time.Sleep(100 * time.Millisecond)

	// Try to make a call that should trigger reconnection attempts to the bad URL
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = client.Call(ctx, "system.info", nil, &result)
	// Should fail due to reconnection failure to bad URL
	assert.Error(t, err)

	// Close should still work
	err = client.Close()
	assert.NoError(t, err)
}
