package truenas

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIKeyClient(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	client := server.CreateTestClient(t)
	defer client.Close()

	apiKeyClient := NewAPIKeyClient(client)
	assert.NotNil(t, apiKeyClient)
	assert.Equal(t, client, apiKeyClient.client)
}

func TestAPIKeyClient_List(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockAPIKeys := []APIKey{
		{
			ID:        1,
			Name:      "test-key-1",
			Key:       "ak_123456789",
			CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			Username:  "admin",
		},
		{
			ID:        2,
			Name:      "test-key-2",
			Key:       "ak_987654321",
			CreatedAt: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
			Username:  "user1",
		},
	}
	server.SetResponse("api_key.query", mockAPIKeys)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	keys, err := client.APIKey.List(ctx)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Equal(t, "test-key-1", keys[0].Name)
	assert.Equal(t, "ak_123456789", keys[0].Key)
	assert.Equal(t, "admin", keys[0].Username)
	assert.Equal(t, "test-key-2", keys[1].Name)
	assert.Equal(t, "ak_987654321", keys[1].Key)
	assert.Equal(t, "user1", keys[1].Username)
}

func TestAPIKeyClient_List_Empty(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("api_key.query", []APIKey{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	keys, err := client.APIKey.List(ctx)
	require.NoError(t, err)
	assert.Len(t, keys, 0)
}

func TestAPIKeyClient_List_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("api_key.query", 500, "Internal server error")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	keys, err := client.APIKey.List(ctx)
	require.Error(t, err)
	assert.Nil(t, keys)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "Internal server error", apiErr.Message)
}

func TestAPIKeyClient_Get(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockAPIKey := APIKey{
		ID:        1,
		Name:      "test-key",
		Key:       "ak_123456789",
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Username:  "admin",
	}
	server.SetResponse("api_key.query", []APIKey{mockAPIKey})

	client := server.CreateTestClient(t)
	defer client.Close()

	tests := []struct {
		name    string
		id      int
		want    *APIKey
		wantErr bool
	}{
		{
			name:    "existing API key",
			id:      1,
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name:    "valid ID zero",
			id:      0,
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name:    "negative ID",
			id:      -1,
			want:    &mockAPIKey,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewTestContext(t)
			key, err := client.APIKey.Get(ctx, tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, key)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, key)
			assert.Equal(t, tt.want.ID, key.ID)
			assert.Equal(t, tt.want.Name, key.Name)
			assert.Equal(t, tt.want.Key, key.Key)
			assert.Equal(t, tt.want.Username, key.Username)
		})
	}
}

func TestAPIKeyClient_Get_NotFound(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("api_key.query", []APIKey{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	key, err := client.APIKey.Get(ctx, 999)
	assert.Error(t, err)
	assert.Nil(t, key)
	var notFoundErr *NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestAPIKeyClient_Get_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("api_key.query", 403, "Permission denied")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	key, err := client.APIKey.Get(ctx, 1)
	require.Error(t, err)
	assert.Nil(t, key)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 403, apiErr.Data.Error)
	assert.Equal(t, "Permission denied", apiErr.Message)
}

func TestAPIKeyClient_Create(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockAPIKey := APIKey{
		ID:        1,
		Name:      "new-api-key",
		Key:       "ak_newkey123",
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Username:  "admin",
	}
	server.SetResponse("api_key.create", mockAPIKey)

	client := server.CreateTestClient(t)
	defer client.Close()

	tests := []struct {
		name    string
		keyName string
		want    *APIKey
		wantErr bool
	}{
		{
			name:    "valid name",
			keyName: "new-api-key",
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name:    "empty name",
			keyName: "",
			want:    &mockAPIKey,
			wantErr: false, // Server handles validation
		},
		{
			name:    "name with spaces",
			keyName: "my api key",
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name:    "name with special characters",
			keyName: "api-key_123!@#",
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name:    "very long name",
			keyName: "very-long-api-key-name-that-might-exceed-normal-limits-1234567890",
			want:    &mockAPIKey,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewTestContext(t)
			key, err := client.APIKey.Create(ctx, tt.keyName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, key)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, key)
			assert.Equal(t, tt.want.ID, key.ID)
			assert.Equal(t, tt.want.Name, key.Name)
			assert.Equal(t, tt.want.Key, key.Key)
			assert.Equal(t, tt.want.Username, key.Username)
		})
	}
}

func TestAPIKeyClient_Create_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("api_key.create", 400, "Invalid API key name")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	key, err := client.APIKey.Create(ctx, "invalid-name")
	require.Error(t, err)
	assert.NotNil(t, key) // API returns empty struct even on error

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 400, apiErr.Data.Error)
	assert.Equal(t, "Invalid API key name", apiErr.Message)
}

func TestAPIKeyClient_Update(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockAPIKey := APIKey{
		ID:        1,
		Name:      "updated-key",
		Key:       "ak_updated123",
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Username:  "admin",
	}
	server.SetResponse("api_key.update", mockAPIKey)

	client := server.CreateTestClient(t)
	defer client.Close()

	tests := []struct {
		name    string
		id      int
		req     *APIKeyUpdateRequest
		want    *APIKey
		wantErr bool
	}{
		{
			name: "update name only",
			id:   1,
			req: &APIKeyUpdateRequest{
				Name: Ptr("updated-key"),
			},
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name: "reset key only",
			id:   1,
			req: &APIKeyUpdateRequest{
				Reset: Ptr(true),
			},
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name: "update name and reset key",
			id:   1,
			req: &APIKeyUpdateRequest{
				Name:  Ptr("updated-key"),
				Reset: Ptr(true),
			},
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name:    "empty update request",
			id:      1,
			req:     &APIKeyUpdateRequest{},
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name: "update with nil values",
			id:   1,
			req: &APIKeyUpdateRequest{
				Name:  nil,
				Reset: nil,
			},
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name: "reset false",
			id:   1,
			req: &APIKeyUpdateRequest{
				Reset: Ptr(false),
			},
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name: "zero ID",
			id:   0,
			req: &APIKeyUpdateRequest{
				Name: Ptr("updated-key"),
			},
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name: "negative ID",
			id:   -1,
			req: &APIKeyUpdateRequest{
				Name: Ptr("updated-key"),
			},
			want:    &mockAPIKey,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewTestContext(t)
			key, err := client.APIKey.Update(ctx, tt.id, tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, key)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, key)
			assert.Equal(t, tt.want.ID, key.ID)
			assert.Equal(t, tt.want.Name, key.Name)
			assert.Equal(t, tt.want.Key, key.Key)
			assert.Equal(t, tt.want.Username, key.Username)
		})
	}
}

func TestAPIKeyClient_Update_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("api_key.update", 404, "API key not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	req := &APIKeyUpdateRequest{Name: Ptr("new-name")}
	key, err := client.APIKey.Update(ctx, 999, req)
	require.Error(t, err)
	assert.NotNil(t, key) // API returns empty struct even on error

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "API key not found", apiErr.Message)
}

func TestAPIKeyClient_UpdateName(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockAPIKey := APIKey{
		ID:        1,
		Name:      "renamed-key",
		Key:       "ak_123456789",
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Username:  "admin",
	}
	server.SetResponse("api_key.update", mockAPIKey)

	client := server.CreateTestClient(t)
	defer client.Close()

	tests := []struct {
		name    string
		id      int
		newName string
		want    *APIKey
		wantErr bool
	}{
		{
			name:    "valid rename",
			id:      1,
			newName: "renamed-key",
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name:    "empty name",
			id:      1,
			newName: "",
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name:    "name with spaces",
			id:      1,
			newName: "my renamed key",
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name:    "zero ID",
			id:      0,
			newName: "renamed-key",
			want:    &mockAPIKey,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewTestContext(t)
			key, err := client.APIKey.UpdateName(ctx, tt.id, tt.newName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, key)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, key)
			assert.Equal(t, tt.want.ID, key.ID)
			assert.Equal(t, tt.want.Name, key.Name)
			assert.Equal(t, tt.want.Key, key.Key)
			assert.Equal(t, tt.want.Username, key.Username)
		})
	}
}

func TestAPIKeyClient_UpdateName_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("api_key.update", 403, "Permission denied")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	key, err := client.APIKey.UpdateName(ctx, 1, "new-name")
	require.Error(t, err)
	assert.NotNil(t, key) // API returns empty struct even on error

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 403, apiErr.Data.Error)
	assert.Equal(t, "Permission denied", apiErr.Message)
}

func TestAPIKeyClient_Reset(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockAPIKey := APIKey{
		ID:        1,
		Name:      "test-key",
		Key:       "ak_newresetkey456",
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Username:  "admin",
	}
	server.SetResponse("api_key.update", mockAPIKey)

	client := server.CreateTestClient(t)
	defer client.Close()

	tests := []struct {
		name    string
		id      int
		want    *APIKey
		wantErr bool
	}{
		{
			name:    "valid reset",
			id:      1,
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name:    "zero ID",
			id:      0,
			want:    &mockAPIKey,
			wantErr: false,
		},
		{
			name:    "negative ID",
			id:      -1,
			want:    &mockAPIKey,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewTestContext(t)
			key, err := client.APIKey.Reset(ctx, tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, key)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, key)
			assert.Equal(t, tt.want.ID, key.ID)
			assert.Equal(t, tt.want.Name, key.Name)
			assert.Equal(t, tt.want.Key, key.Key)
			assert.Equal(t, tt.want.Username, key.Username)
		})
	}
}

func TestAPIKeyClient_Reset_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("api_key.update", 500, "Failed to reset API key")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	key, err := client.APIKey.Reset(ctx, 1)
	require.Error(t, err)
	assert.NotNil(t, key) // API returns empty struct even on error

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "Failed to reset API key", apiErr.Message)
}

func TestAPIKeyClient_Delete(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("api_key.delete", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	tests := []struct {
		name    string
		id      int
		wantErr bool
	}{
		{
			name:    "valid delete",
			id:      1,
			wantErr: false,
		},
		{
			name:    "zero ID",
			id:      0,
			wantErr: false,
		},
		{
			name:    "negative ID",
			id:      -1,
			wantErr: false,
		},
		{
			name:    "large ID",
			id:      999999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewTestContext(t)
			err := client.APIKey.Delete(ctx, tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestAPIKeyClient_Delete_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	tests := []struct {
		name      string
		errorCode int
		errorMsg  string
	}{
		{
			name:      "not found",
			errorCode: 404,
			errorMsg:  "API key not found",
		},
		{
			name:      "permission denied",
			errorCode: 403,
			errorMsg:  "Permission denied",
		},
		{
			name:      "internal error",
			errorCode: 500,
			errorMsg:  "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server.SetError("api_key.delete", tt.errorCode, tt.errorMsg)

			client := server.CreateTestClient(t)
			defer client.Close()

			ctx := NewTestContext(t)
			err := client.APIKey.Delete(ctx, 1)
			require.Error(t, err)

			var apiErr *RPCError
			assert.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.errorCode, apiErr.Data.Error)
			assert.Equal(t, tt.errorMsg, apiErr.Message)
		})
	}
}

// Test struct marshal/unmarshal
func TestAPIKey_JSON(t *testing.T) {
	t.Parallel()
	apiKey := APIKey{
		ID:        1,
		Name:      "test-key",
		Key:       "ak_123456789",
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Username:  "admin",
	}

	// Test marshaling
	data, err := json.Marshal(apiKey)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"id":1`)
	assert.Contains(t, string(data), `"name":"test-key"`)
	assert.Contains(t, string(data), `"key":"ak_123456789"`)
	assert.Contains(t, string(data), `"username":"admin"`)

	// Test unmarshaling
	var decoded APIKey
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, apiKey.ID, decoded.ID)
	assert.Equal(t, apiKey.Name, decoded.Name)
	assert.Equal(t, apiKey.Key, decoded.Key)
	assert.Equal(t, apiKey.Username, decoded.Username)
	assert.True(t, apiKey.CreatedAt.Equal(decoded.CreatedAt))
}

func TestAPIKeyCreateRequest_JSON(t *testing.T) {
	t.Parallel()
	req := APIKeyCreateRequest{
		Name: "test-key",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"name":"test-key"`)

	var decoded APIKeyCreateRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, req.Name, decoded.Name)
}

func TestAPIKeyUpdateRequest_JSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		req  APIKeyUpdateRequest
	}{
		{
			name: "name only",
			req: APIKeyUpdateRequest{
				Name: Ptr("updated-name"),
			},
		},
		{
			name: "reset only",
			req: APIKeyUpdateRequest{
				Reset: Ptr(true),
			},
		},
		{
			name: "both fields",
			req: APIKeyUpdateRequest{
				Name:  Ptr("updated-name"),
				Reset: Ptr(false),
			},
		},
		{
			name: "empty request",
			req:  APIKeyUpdateRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			require.NoError(t, err)

			var decoded APIKeyUpdateRequest
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			if tt.req.Name != nil {
				require.NotNil(t, decoded.Name)
				assert.Equal(t, *tt.req.Name, *decoded.Name)
			} else {
				assert.Nil(t, decoded.Name)
			}

			if tt.req.Reset != nil {
				require.NotNil(t, decoded.Reset)
				assert.Equal(t, *tt.req.Reset, *decoded.Reset)
			} else {
				assert.Nil(t, decoded.Reset)
			}
		})
	}
}
