package truenas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// TestServerOption configures a TestServer
type TestServerOption func(*TestServer)

// WithConnectionTracking enables connection tracking and immediate shutdown capability
func WithConnectionTracking() TestServerOption {
	return func(ts *TestServer) {
		ts.trackConnections = true
		ts.connections = make(map[*websocket.Conn]bool)
	}
}

// WithAuthSuccess configures whether authentication should succeed or fail
func WithAuthSuccess(success bool) TestServerOption {
	return func(ts *TestServer) {
		ts.authSuccess = success
	}
}

// WithCustomHandler sets a custom message handler for the server
func WithCustomHandler(handler func(Request) (Response, bool)) TestServerOption {
	return func(ts *TestServer) {
		ts.customHandler = handler
	}
}

// WithDebug enables debug logging for the server
func WithDebug(debug bool) TestServerOption {
	return func(ts *TestServer) {
		ts.debug = debug
	}
}

// TestServer provides a mock TrueNAS WebSocket server for unit testing
type TestServer struct {
	*httptest.Server
	responses map[string]any
	errors    map[string]*RPCError
	nextJobID int // Auto-incrementing job ID counter

	// Connection tracking
	connections      map[*websocket.Conn]bool
	connMutex        sync.Mutex
	trackConnections bool

	// Behavior configuration
	customHandler func(Request) (Response, bool)
	authSuccess   bool
	debug         bool
}

// NewTestServer creates a new mock TrueNAS server for testing
func NewTestServer(t *testing.T, opts ...TestServerOption) *TestServer {
	ts := &TestServer{
		responses:   make(map[string]any),
		errors:      make(map[string]*RPCError),
		nextJobID:   100,  // Start at 100 to avoid conflicts
		authSuccess: true, // Default to successful auth
	}

	// Apply options
	for _, opt := range opts {
		opt(ts)
	}

	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		// Track connection if enabled
		if ts.trackConnections {
			ts.connMutex.Lock()
			ts.connections[conn] = true
			ts.connMutex.Unlock()
		}

		defer func() {
			if ts.trackConnections {
				ts.connMutex.Lock()
				delete(ts.connections, conn)
				ts.connMutex.Unlock()
			}
			conn.Close()
		}()

		for {
			var req Request
			err := conn.ReadJSON(&req)
			if err != nil {
				break
			}

			// Use custom handler if provided
			if ts.customHandler != nil {
				response, shouldSend := ts.customHandler(req)
				if shouldSend {
					_ = conn.WriteJSON(response)
				}
				continue
			}

			response := Response{
				JSONRPC: "2.0",
				ID:      &req.ID,
			}

			// Check for error responses first
			if errResp, hasError := ts.errors[req.Method]; hasError {
				response.Error = errResp
			} else if req.Method == "auth.login" || req.Method == "auth.login_with_api_key" {
				if ts.authSuccess {
					response.Result = json.RawMessage(`true`)
				} else {
					response.Error = &RPCError{
						Code:    -32001,
						Message: "Authentication failed",
						Data:    &RPCErrorData{Error: 401, Reason: "Authentication failed"},
					}
				}
			} else if mockResp, hasResponse := ts.responses[req.Method]; hasResponse {
				result, _ := json.Marshal(mockResp)
				response.Result = json.RawMessage(result)
			} else {
				// Provide default responses for common methods
				switch req.Method {
				case "system.info":
					defaultSystemInfo := map[string]any{
						"hostname": "test-truenas",
						"version":  "TrueNAS-SCALE-23.10.2",
					}
					result, _ := json.Marshal(defaultSystemInfo)
					response.Result = json.RawMessage(result)
				default:
					// Default success response
					response.Result = json.RawMessage(`true`)
				}
			}

			_ = conn.WriteJSON(response)
		}
	}))

	return ts
}

// Shutdown gracefully shuts down the test server and immediately closes all tracked connections
func (ts *TestServer) Shutdown() {
	ts.Close()
	if ts.trackConnections {
		ts.connMutex.Lock()
		for conn := range ts.connections {
			conn.Close()
		}
		ts.connMutex.Unlock()
	}
}

// SetResponse sets a mock response for a specific method
func (ts *TestServer) SetResponse(method string, response any) {
	ts.responses[method] = response
}

// SetError sets a mock error response for a specific method
func (ts *TestServer) SetError(method string, code int, message string) {
	ts.errors[method] = &RPCError{
		Code:    -32001,
		Message: message,
		Data:    &RPCErrorData{Error: code, Reason: message},
	}
}

// GetWebSocketURL returns the WebSocket URL for this test server
func (ts *TestServer) GetWebSocketURL() string {
	return strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/current"
}

// CreateTestClient creates a test client connected to this server
func (ts *TestServer) CreateTestClient(t *testing.T) *Client {
	client, err := NewClient(ts.GetWebSocketURL(), Options{
		Username: "testuser",
		Password: "testpass",
	})
	require.NoError(t, err)
	return client
}

// CreateTestClientWithoutAuth creates a test client without auto-authentication
func (ts *TestServer) CreateTestClientWithoutAuth(t *testing.T) *Client {
	client, err := NewClient(ts.GetWebSocketURL(), Options{})
	require.NoError(t, err)
	return client
}

// NewTestContext creates a test context with timeout
func NewTestContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// MethodCall is a helper to verify a method was called with expected parameters
type MethodCall struct {
	Method string
	Params []any
}

// CallTracker tracks method calls for verification
type CallTracker struct {
	calls []MethodCall
}

// NewCallTracker creates a new call tracker
func NewCallTracker() *CallTracker {
	return &CallTracker{
		calls: make([]MethodCall, 0),
	}
}

// AddCall records a method call
func (ct *CallTracker) AddCall(method string, params []any) {
	ct.calls = append(ct.calls, MethodCall{
		Method: method,
		Params: params,
	})
}

// GetCalls returns all recorded calls
func (ct *CallTracker) GetCalls() []MethodCall {
	return ct.calls
}

// HasCall checks if a specific method was called
func (ct *CallTracker) HasCall(method string) bool {
	for _, call := range ct.calls {
		if call.Method == method {
			return true
		}
	}
	return false
}

// Common test data structures
var (
	TestPool = Pool{
		ID:   1,
		Name: "tank",
		Path: "/mnt/tank",
	}

	TestDataset = Dataset{
		ID:   "tank/test",
		Name: "tank/test",
		Pool: "tank",
	}

	TestUser = User{
		ID:       1000,
		UID:      1000,
		Username: "testuser",
		FullName: "Test User",
		Home:     "/home/testuser",
		Shell:    "/bin/bash",
	}

	TestGroup = Group{
		ID:      1000,
		GID:     1000,
		Name:    "testgroup",
		Builtin: false,
	}

	TestVM = VM{
		ID:          1,
		Name:        "test-vm",
		Description: "Test VM",
		VCPUs:       2,
		Memory:      1024,
		Autostart:   false,
		Status:      VMStatus{State: "STOPPED"},
	}

	TestCertificate = Certificate{
		ID:      1,
		Name:    "test-cert",
		Subject: map[string]any{"CN": "test.example.com"},
		Issuer:  map[string]any{"CN": "test.example.com"},
	}

	TestAlert = Alert{
		UUID:           "test-uuid-123",
		Source:         "test",
		Klass:          "TestAlert",
		Args:           []any{},
		Node:           "testnode",
		Level:          "INFO",
		Formatted:      "Test alert message",
		OneShot:        false,
		Mail:           false,
		Text:           "Test alert",
		DateTime:       time.Now(),
		LastOccurrence: time.Now(),
	}
)

// SetJobResponse sets up a mock job response that returns a job ID and
// configures core.get_jobs to return a completed job with the specified result
func (ts *TestServer) SetJobResponse(method string, result any) {
	ts.nextJobID++
	jobID := ts.nextJobID

	// Set the method to return the job ID
	ts.SetResponse(method, jobID)

	// Set core.get_jobs to return completed job
	mockJob := Job{
		ID:     jobID,
		Method: method,
		State:  "SUCCESS",
		Result: result,
	}
	ts.SetResponse("core.get_jobs", []Job{mockJob})
}

// SetJobError sets up a mock job response that returns a job ID and
// configures core.get_jobs to return a failed job with the specified error
func (ts *TestServer) SetJobError(method string, errorMsg string) {
	ts.nextJobID++
	jobID := ts.nextJobID

	ts.SetResponse(method, jobID)

	mockJob := Job{
		ID:     jobID,
		Method: method,
		State:  "FAILED",
		Error:  &errorMsg,
	}
	ts.SetResponse("core.get_jobs", []Job{mockJob})
}

// upgrader is the WebSocket upgrader used by test servers
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
