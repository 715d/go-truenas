package truenas

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBootClient(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	client := server.CreateTestClient(t)
	defer client.Close()

	bootClient := NewBootClient(client)
	assert.NotNil(t, bootClient)
	assert.Equal(t, client, bootClient.client)
}

func TestBootClient_GetDisks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		mockDisks     []BootDisk
		expectedCount int
		expectedError bool
		errorCode     int
		errorMessage  string
	}{
		{
			name: "successful retrieval with multiple disks",
			mockDisks: []BootDisk{
				{
					Name:      "sda",
					Label:     "boot-disk-1",
					Size:      120000000000,
					Path:      "/dev/sda",
					Status:    "ONLINE",
					Serial:    "WD-12345",
					Model:     "WD Blue",
					Type:      "HDD",
					Available: true,
				},
				{
					Name:      "sdb",
					Label:     "boot-disk-2",
					Size:      120000000000,
					Path:      "/dev/sdb",
					Status:    "ONLINE",
					Serial:    "WD-67890",
					Model:     "WD Red",
					Type:      "HDD",
					Available: false,
				},
			},
			expectedCount: 2,
			expectedError: false,
		},
		{
			name:          "successful retrieval with empty result",
			mockDisks:     []BootDisk{},
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:          "API error response",
			mockDisks:     nil,
			expectedCount: 0,
			expectedError: true,
			errorCode:     500,
			errorMessage:  "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			if tt.expectedError {
				server.SetError("boot.get_disks", tt.errorCode, tt.errorMessage)
			} else {
				server.SetResponse("boot.get_disks", tt.mockDisks)
			}

			client := server.CreateTestClient(t)
			defer client.Close()

			bootClient := NewBootClient(client)
			ctx := NewTestContext(t)

			disks, err := bootClient.GetDisks(ctx)

			if tt.expectedError {
				assert.Error(t, err)
				var apiErr *RPCError
				assert.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.errorCode, apiErr.Data.Error)
				assert.Equal(t, tt.errorMessage, apiErr.Message)
				assert.Nil(t, disks)
			} else {
				assert.NoError(t, err)
				assert.Len(t, disks, tt.expectedCount)

				if tt.expectedCount > 0 {
					assert.Equal(t, tt.mockDisks[0].Name, disks[0].Name)
					assert.Equal(t, tt.mockDisks[0].Label, disks[0].Label)
					assert.Equal(t, tt.mockDisks[0].Size, disks[0].Size)
					assert.Equal(t, tt.mockDisks[0].Path, disks[0].Path)
					assert.Equal(t, tt.mockDisks[0].Status, disks[0].Status)
					assert.Equal(t, tt.mockDisks[0].Serial, disks[0].Serial)
					assert.Equal(t, tt.mockDisks[0].Model, disks[0].Model)
					assert.Equal(t, tt.mockDisks[0].Type, disks[0].Type)
					assert.Equal(t, tt.mockDisks[0].Available, disks[0].Available)
				}
			}
		})
	}
}

func TestBootClient_GetState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		mockState     *BootState
		expectedError bool
		errorCode     int
		errorMessage  string
	}{
		{
			name: "successful state retrieval",
			mockState: &BootState{
				Name:     "boot-pool",
				ID:       "boot-pool-id",
				GUID:     "1234567890",
				Hostname: "truenas.local",
				Status:   "ONLINE",
				Scan:     nil,
				Properties: map[string]any{
					"version":  "5000",
					"readonly": "off",
					"autotrim": "off",
				},
				Groups: []BootVdev{
					{
						Name:   "mirror-0",
						Type:   "mirror",
						Status: "ONLINE",
						Stats:  nil,
						Children: []BootVdev{
							{
								Name:   "sda1",
								Type:   "disk",
								Status: "ONLINE",
								Device: "sda1",
								Disk:   "sda",
								Path:   "/dev/sda1",
							},
							{
								Name:   "sdb1",
								Type:   "disk",
								Status: "ONLINE",
								Device: "sdb1",
								Disk:   "sdb",
								Path:   "/dev/sdb1",
							},
						},
					},
				},
				Topology: BootTopology{
					Data: []BootVdev{
						{
							Name:   "mirror-0",
							Type:   "mirror",
							Status: "ONLINE",
						},
					},
					Log:     []BootVdev{},
					Cache:   []BootVdev{},
					Spare:   []BootVdev{},
					Special: []BootVdev{},
					Dedup:   []BootVdev{},
				},
				Healthy: true,
				Warning: false,
				Unknown: false,
			},
			expectedError: false,
		},
		{
			name:          "API error response",
			mockState:     nil,
			expectedError: true,
			errorCode:     403,
			errorMessage:  "Access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			if tt.expectedError {
				server.SetError("boot.get_state", tt.errorCode, tt.errorMessage)
			} else {
				server.SetResponse("boot.get_state", *tt.mockState)
			}

			client := server.CreateTestClient(t)
			defer client.Close()

			bootClient := NewBootClient(client)
			ctx := NewTestContext(t)

			state, err := bootClient.GetState(ctx)

			if tt.expectedError {
				assert.Error(t, err)
				var apiErr *RPCError
				assert.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.errorCode, apiErr.Data.Error)
				assert.Equal(t, tt.errorMessage, apiErr.Message)
				// Note: GetState returns a pointer to an empty struct on error, not nil
				assert.NotNil(t, state)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, state)
				assert.Equal(t, tt.mockState.Name, state.Name)
				assert.Equal(t, tt.mockState.ID, state.ID)
				assert.Equal(t, tt.mockState.GUID, state.GUID)
				assert.Equal(t, tt.mockState.Hostname, state.Hostname)
				assert.Equal(t, tt.mockState.Status, state.Status)
				assert.Equal(t, tt.mockState.Healthy, state.Healthy)
				assert.Equal(t, tt.mockState.Warning, state.Warning)
				assert.Equal(t, tt.mockState.Unknown, state.Unknown)
				assert.Len(t, state.Groups, len(tt.mockState.Groups))
				assert.Len(t, state.Topology.Data, len(tt.mockState.Topology.Data))
			}
		})
	}
}

func TestBootClient_Attach(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		device        string
		expand        bool
		expectedError bool
		errorCode     int
		errorMessage  string
	}{
		{
			name:          "successful attach without expand",
			device:        "sdc",
			expand:        false,
			expectedError: false,
		},
		{
			name:          "successful attach with expand",
			device:        "sdd",
			expand:        true,
			expectedError: false,
		},
		{
			name:          "attach with empty device name",
			device:        "",
			expand:        false,
			expectedError: false, // API should handle validation
		},
		{
			name:          "API error during attach",
			device:        "sde",
			expand:        false,
			expectedError: true,
			errorCode:     400,
			errorMessage:  "Device not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			if tt.expectedError {
				server.SetJobError("boot.attach", tt.errorMessage)
			} else {
				server.SetJobResponse("boot.attach", nil)
			}

			client := server.CreateTestClient(t)
			defer client.Close()

			bootClient := NewBootClient(client)
			ctx := NewTestContext(t)

			err := bootClient.Attach(ctx, tt.device, tt.expand)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBootClient_Detach(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		device        string
		expectedError bool
		errorCode     int
		errorMessage  string
	}{
		{
			name:          "successful detach",
			device:        "sdc",
			expectedError: false,
		},
		{
			name:          "detach with empty device name",
			device:        "",
			expectedError: false, // API should handle validation
		},
		{
			name:          "API error during detach",
			device:        "sde",
			expectedError: true,
			errorCode:     404,
			errorMessage:  "Device not found in boot pool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			if tt.expectedError {
				server.SetError("boot.detach", tt.errorCode, tt.errorMessage)
			} else {
				server.SetResponse("boot.detach", true)
			}

			client := server.CreateTestClient(t)
			defer client.Close()

			bootClient := NewBootClient(client)
			ctx := NewTestContext(t)

			err := bootClient.Detach(ctx, tt.device)

			if tt.expectedError {
				assert.Error(t, err)
				var apiErr *RPCError
				assert.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.errorCode, apiErr.Data.Error)
				assert.Equal(t, tt.errorMessage, apiErr.Message)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBootClient_Replace(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		label         string
		device        string
		expectedError bool
		errorCode     int
		errorMessage  string
	}{
		{
			name:          "successful replace",
			label:         "boot-disk-1",
			device:        "sdc",
			expectedError: false,
		},
		{
			name:          "replace with empty label",
			label:         "",
			device:        "sdc",
			expectedError: false, // API should handle validation
		},
		{
			name:          "replace with empty device",
			label:         "boot-disk-1",
			device:        "",
			expectedError: false, // API should handle validation
		},
		{
			name:          "API error during replace",
			label:         "nonexistent-label",
			device:        "sdc",
			expectedError: true,
			errorCode:     404,
			errorMessage:  "Label not found in boot pool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			if tt.expectedError {
				server.SetError("boot.replace", tt.errorCode, tt.errorMessage)
			} else {
				server.SetResponse("boot.replace", true)
			}

			client := server.CreateTestClient(t)
			defer client.Close()

			bootClient := NewBootClient(client)
			ctx := NewTestContext(t)

			err := bootClient.Replace(ctx, tt.label, tt.device)

			if tt.expectedError {
				assert.Error(t, err)
				var apiErr *RPCError
				assert.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.errorCode, apiErr.Data.Error)
				assert.Equal(t, tt.errorMessage, apiErr.Message)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBootClient_Scrub(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		expectedError bool
		errorMessage  string
	}{
		{
			name:          "successful scrub start",
			expectedError: false,
		},
		{
			name:          "scrub job failure",
			expectedError: true,
			errorMessage:  "Scrub already in progress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			if tt.expectedError {
				server.SetJobError("boot.scrub", tt.errorMessage)
			} else {
				server.SetJobResponse("boot.scrub", nil)
			}

			client := server.CreateTestClient(t)
			defer client.Close()

			bootClient := NewBootClient(client)
			ctx := NewTestContext(t)

			err := bootClient.Scrub(ctx)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBootClient_GetScrubInterval(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		mockInterval     int
		expectedInterval int
		expectedError    bool
		errorCode        int
		errorMessage     string
	}{
		{
			name:             "successful interval retrieval - 7 days",
			mockInterval:     7,
			expectedInterval: 7,
			expectedError:    false,
		},
		{
			name:             "successful interval retrieval - 30 days",
			mockInterval:     30,
			expectedInterval: 30,
			expectedError:    false,
		},
		{
			name:             "successful interval retrieval - 0 (disabled)",
			mockInterval:     0,
			expectedInterval: 0,
			expectedError:    false,
		},
		{
			name:          "API error response",
			mockInterval:  0,
			expectedError: true,
			errorCode:     500,
			errorMessage:  "Failed to get scrub interval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			if tt.expectedError {
				server.SetError("boot.get_scrub_interval", tt.errorCode, tt.errorMessage)
			} else {
				server.SetResponse("boot.get_scrub_interval", tt.mockInterval)
			}

			client := server.CreateTestClient(t)
			defer client.Close()

			bootClient := NewBootClient(client)
			ctx := NewTestContext(t)

			interval, err := bootClient.GetScrubInterval(ctx)

			if tt.expectedError {
				assert.Error(t, err)
				var apiErr *RPCError
				assert.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.errorCode, apiErr.Data.Error)
				assert.Equal(t, tt.errorMessage, apiErr.Message)
				assert.Equal(t, 0, interval) // Default value for error case
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedInterval, interval)
			}
		})
	}
}

func TestBootClient_SetScrubInterval(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		interval      int
		expectedError bool
		errorCode     int
		errorMessage  string
	}{
		{
			name:          "successful interval set - 7 days",
			interval:      7,
			expectedError: false,
		},
		{
			name:          "successful interval set - 30 days",
			interval:      30,
			expectedError: false,
		},
		{
			name:          "successful interval set - 0 (disabled)",
			interval:      0,
			expectedError: false,
		},
		{
			name:          "successful interval set - negative value",
			interval:      -1,
			expectedError: false, // API should handle validation
		},
		{
			name:          "successful interval set - large value",
			interval:      365,
			expectedError: false,
		},
		{
			name:          "API error response",
			interval:      7,
			expectedError: true,
			errorCode:     400,
			errorMessage:  "Invalid scrub interval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			if tt.expectedError {
				server.SetError("boot.set_scrub_interval", tt.errorCode, tt.errorMessage)
			} else {
				server.SetResponse("boot.set_scrub_interval", true)
			}

			client := server.CreateTestClient(t)
			defer client.Close()

			bootClient := NewBootClient(client)
			ctx := NewTestContext(t)

			err := bootClient.SetScrubInterval(ctx, tt.interval)

			if tt.expectedError {
				assert.Error(t, err)
				var apiErr *RPCError
				assert.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.errorCode, apiErr.Data.Error)
				assert.Equal(t, tt.errorMessage, apiErr.Message)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBootClient_ContextCancellation(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	client := server.CreateTestClient(t)
	defer client.Close()

	bootClient := NewBootClient(client)

	// Test with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// All methods should respect context cancellation
	_, err := bootClient.GetDisks(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	_, err = bootClient.GetState(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	err = bootClient.Attach(ctx, "sdc", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	err = bootClient.Detach(ctx, "sdc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	err = bootClient.Replace(ctx, "label", "device")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	err = bootClient.Scrub(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	_, err = bootClient.GetScrubInterval(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	err = bootClient.SetScrubInterval(ctx, 7)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestBootClient_NilClient(t *testing.T) {
	t.Parallel()
	// Test behavior when client is nil
	bootClient := &BootClient{client: nil}
	ctx := NewTestContext(t)

	// All methods should panic or return appropriate errors when client is nil
	assert.Panics(t, func() {
		_, _ = bootClient.GetDisks(ctx)
	})

	assert.Panics(t, func() {
		_, _ = bootClient.GetState(ctx)
	})

	assert.Panics(t, func() {
		_ = bootClient.Attach(ctx, "sdc", false)
	})

	assert.Panics(t, func() {
		_ = bootClient.Detach(ctx, "sdc")
	})

	assert.Panics(t, func() {
		_ = bootClient.Replace(ctx, "label", "device")
	})

	assert.Panics(t, func() {
		_ = bootClient.Scrub(ctx)
	})

	assert.Panics(t, func() {
		_, _ = bootClient.GetScrubInterval(ctx)
	})

	assert.Panics(t, func() {
		_ = bootClient.SetScrubInterval(ctx, 7)
	})
}

func TestBootAttachRequest_Struct(t *testing.T) {
	t.Parallel()
	// Test BootAttachRequest struct
	req := BootAttachRequest{
		Device: "sdc",
		Expand: true,
	}

	assert.Equal(t, "sdc", req.Device)
	assert.True(t, req.Expand)

	// Test with default values
	req2 := BootAttachRequest{
		Device: "sdd",
	}
	assert.Equal(t, "sdd", req2.Device)
	assert.False(t, req2.Expand) // Should be false by default
}

func TestBootDisk_Struct(t *testing.T) {
	t.Parallel()
	// Test BootDisk struct with all fields
	disk := BootDisk{
		Name:      "sda",
		Label:     "boot-disk-1",
		Size:      120000000000,
		Path:      "/dev/sda",
		Status:    "ONLINE",
		Serial:    "WD-12345",
		Model:     "WD Blue",
		Type:      "HDD",
		Available: true,
	}

	assert.Equal(t, "sda", disk.Name)
	assert.Equal(t, "boot-disk-1", disk.Label)
	assert.Equal(t, int64(120000000000), disk.Size)
	assert.Equal(t, "/dev/sda", disk.Path)
	assert.Equal(t, "ONLINE", disk.Status)
	assert.Equal(t, "WD-12345", disk.Serial)
	assert.Equal(t, "WD Blue", disk.Model)
	assert.Equal(t, "HDD", disk.Type)
	assert.True(t, disk.Available)
}

func TestBootState_Struct(t *testing.T) {
	t.Parallel()
	// Test BootState struct with nested structures
	state := BootState{
		Name:     "boot-pool",
		ID:       "boot-pool-id",
		GUID:     "1234567890",
		Hostname: "truenas.local",
		Status:   "ONLINE",
		Scan:     map[string]any{"function": "scrub", "state": "finished"},
		Properties: map[string]any{
			"version":  "5000",
			"readonly": "off",
		},
		Groups: []BootVdev{
			{
				Name:   "mirror-0",
				Type:   "mirror",
				Status: "ONLINE",
				Children: []BootVdev{
					{
						Name:   "sda1",
						Type:   "disk",
						Status: "ONLINE",
						Device: "sda1",
						Disk:   "sda",
						Path:   "/dev/sda1",
					},
				},
			},
		},
		Topology: BootTopology{
			Data: []BootVdev{
				{
					Name:   "mirror-0",
					Type:   "mirror",
					Status: "ONLINE",
				},
			},
		},
		Healthy: true,
		Warning: false,
		Unknown: false,
	}

	assert.Equal(t, "boot-pool", state.Name)
	assert.Equal(t, "boot-pool-id", state.ID)
	assert.Equal(t, "1234567890", state.GUID)
	assert.Equal(t, "truenas.local", state.Hostname)
	assert.Equal(t, "ONLINE", state.Status)
	assert.NotNil(t, state.Scan)
	assert.Len(t, state.Properties, 2)
	assert.Len(t, state.Groups, 1)
	assert.Equal(t, "mirror-0", state.Groups[0].Name)
	assert.Len(t, state.Groups[0].Children, 1)
	assert.Equal(t, "sda1", state.Groups[0].Children[0].Name)
	assert.Len(t, state.Topology.Data, 1)
	assert.True(t, state.Healthy)
	assert.False(t, state.Warning)
	assert.False(t, state.Unknown)
}

func TestBootVdev_Struct(t *testing.T) {
	t.Parallel()
	// Test BootVdev struct with all fields
	vdev := BootVdev{
		Name:     "sda1",
		Type:     "disk",
		Status:   "ONLINE",
		Stats:    map[string]any{"read_errors": 0, "write_errors": 0},
		Children: []BootVdev{},
		Device:   "sda1",
		Disk:     "sda",
		Path:     "/dev/sda1",
	}

	assert.Equal(t, "sda1", vdev.Name)
	assert.Equal(t, "disk", vdev.Type)
	assert.Equal(t, "ONLINE", vdev.Status)
	assert.NotNil(t, vdev.Stats)
	assert.Empty(t, vdev.Children)
	assert.Equal(t, "sda1", vdev.Device)
	assert.Equal(t, "sda", vdev.Disk)
	assert.Equal(t, "/dev/sda1", vdev.Path)
}

func TestBootTopology_Struct(t *testing.T) {
	t.Parallel()
	// Test BootTopology struct
	topology := BootTopology{
		Data: []BootVdev{
			{Name: "mirror-0", Type: "mirror", Status: "ONLINE"},
		},
		Log: []BootVdev{
			{Name: "log-0", Type: "disk", Status: "ONLINE"},
		},
		Cache: []BootVdev{
			{Name: "cache-0", Type: "disk", Status: "ONLINE"},
		},
		Spare: []BootVdev{
			{Name: "spare-0", Type: "disk", Status: "AVAIL"},
		},
		Special: []BootVdev{
			{Name: "special-0", Type: "disk", Status: "ONLINE"},
		},
		Dedup: []BootVdev{
			{Name: "dedup-0", Type: "disk", Status: "ONLINE"},
		},
	}

	assert.Len(t, topology.Data, 1)
	assert.Equal(t, "mirror-0", topology.Data[0].Name)
	assert.Len(t, topology.Log, 1)
	assert.Equal(t, "log-0", topology.Log[0].Name)
	assert.Len(t, topology.Cache, 1)
	assert.Equal(t, "cache-0", topology.Cache[0].Name)
	assert.Len(t, topology.Spare, 1)
	assert.Equal(t, "spare-0", topology.Spare[0].Name)
	assert.Len(t, topology.Special, 1)
	assert.Equal(t, "special-0", topology.Special[0].Name)
	assert.Len(t, topology.Dedup, 1)
	assert.Equal(t, "dedup-0", topology.Dedup[0].Name)
}
