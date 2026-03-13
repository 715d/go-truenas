package truenas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthClient_Login(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	client := server.CreateTestClient(t)
	defer client.Close()

	tests := []struct {
		name     string
		username string
		password string
		want     bool
		wantErr  bool
	}{
		{
			name:     "successful login",
			username: "admin",
			password: "password",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			password: "password",
			want:     true, // Server handles validation
			wantErr:  false,
		},
		{
			name:     "empty password",
			username: "admin",
			password: "",
			want:     true, // Server handles validation
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewTestContext(t)
			got, err := client.Auth.Login(ctx, tt.username, tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAuthClient_LoginWithAPIKey(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	client := server.CreateTestClient(t)
	defer client.Close()

	tests := []struct {
		name    string
		apiKey  string
		want    bool
		wantErr bool
	}{
		{
			name:    "successful API key login",
			apiKey:  "test-api-key-123",
			want:    true,
			wantErr: false,
		},
		{
			name:    "empty API key",
			apiKey:  "",
			want:    true, // Server handles validation
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewTestContext(t)
			got, err := client.Auth.LoginWithAPIKey(ctx, tt.apiKey)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAuthClient_Logout(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Auth.Logout(ctx)
	assert.NoError(t, err)
}

func TestAuthClient_CheckPassword(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("auth.check_password", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	tests := []struct {
		name     string
		username string
		password string
		want     bool
		wantErr  bool
	}{
		{
			name:     "valid password",
			username: "admin",
			password: "correct",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			password: "password",
			want:     true, // Mock always returns true
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewTestContext(t)
			got, err := client.Auth.CheckPassword(ctx, tt.username, tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAuthClient_GenerateToken(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockResponse := &TokenResponse{
		Token: "generated-token-123",
	}
	server.SetResponse("auth.generate_token", mockResponse)

	client := server.CreateTestClient(t)
	defer client.Close()

	tests := []struct {
		name    string
		req     GenerateTokenRequest
		want    *TokenResponse
		wantErr bool
	}{
		{
			name: "successful token generation",
			req: GenerateTokenRequest{
				TTL:        3600,
				Attributes: map[string]any{"test": "value"},
			},
			want:    mockResponse,
			wantErr: false,
		},
		{
			name: "zero TTL",
			req: GenerateTokenRequest{
				TTL:        0,
				Attributes: map[string]any{},
			},
			want:    mockResponse,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewTestContext(t)
			got, err := client.Auth.GenerateToken(ctx, tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAuthClient_ErrorHandling(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	// Set up error response
	server.SetError("auth.login", 401, "Authentication failed")

	client := server.CreateTestClientWithoutAuth(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Auth.Login(ctx, "baduser", "badpass")
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 401, apiErr.Data.Error)
	assert.Equal(t, "Authentication failed", apiErr.Message)
}
