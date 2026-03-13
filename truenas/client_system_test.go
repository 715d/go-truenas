package truenas

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemClient_GetInfo(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInfo := &SystemInfo{
		Version:       "TrueNAS-SCALE-23.10.2",
		Hostname:      "truenas.local",
		UptimeSeconds: 3600.5,
		SystemSerial:  "12345",
		SystemProduct: "TrueNAS",
	}
	server.SetResponse("system.info", mockInfo)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	info, err := client.System.GetInfo(ctx)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "TrueNAS-SCALE-23.10.2", info.Version)
	assert.Equal(t, "truenas.local", info.Hostname)
	assert.Equal(t, 3600.5, info.UptimeSeconds)
}

func TestSystemClient_GetGeneralConfig(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := &SystemGeneralConfig{
		UIHTTPSProtocols:    []string{"TLSv1.2", "TLSv1.3"},
		UIHTTPSPort:         443,
		UIHTTPSRedirect:     true,
		UIPort:              80,
		UIAddress:           []string{"0.0.0.0"},
		Language:            "en",
		Timezone:            "America/New_York",
		KBDMap:              "us",
		CrashReporting:      true,
		UsageCollection:     true,
		WizardShown:         true,
		DSAuth:              false,
		Birthday:            "1970-01-01",
		UIConsoleMsgEnabled: true,
	}
	server.SetResponse("system.general.config", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	config, err := client.System.GetGeneralConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, 443, config.UIHTTPSPort)
	assert.Equal(t, "America/New_York", config.Timezone)
	assert.True(t, config.UIHTTPSRedirect)
}

func TestSystemClient_UpdateGeneralConfig(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := &SystemGeneralConfig{
		Timezone:        "America/Los_Angeles",
		UIHTTPSPort:     8443,
		UIHTTPSRedirect: false,
	}
	server.SetResponse("system.general.update", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	updated, err := client.System.UpdateGeneralConfig(ctx, mockConfig)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "America/Los_Angeles", updated.Timezone)
	assert.Equal(t, 8443, updated.UIHTTPSPort)
	assert.False(t, updated.UIHTTPSRedirect)
}

func TestSystemClient_GetVersion(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("system.version", "TrueNAS-SCALE-23.10.2")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	version, err := client.System.GetVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, "TrueNAS-SCALE-23.10.2", version)
}

func TestSystemClient_GetHostname(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("system.hostname", "truenas.local")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	hostname, err := client.System.GetHostname(ctx)
	require.NoError(t, err)
	assert.Equal(t, "truenas.local", hostname)
}

func TestSystemClient_SetHostname(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("system.general.update", map[string]any{"hostname": "new-hostname"})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.System.SetHostname(ctx, "new-hostname")
	assert.NoError(t, err)
}

func TestSystemClient_Ready(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("system.ready", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	ready, err := client.System.Ready(ctx)
	require.NoError(t, err)
	assert.True(t, ready)
}

func TestSystemClient_Reboot(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("system.reboot", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.System.Reboot(ctx, 0)
	assert.NoError(t, err)
}

func TestSystemClient_Shutdown(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("system.shutdown", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.System.Shutdown(ctx, 0)
	assert.NoError(t, err)
}

func TestSystemClient_ListBootEnvs(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	created1, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	created2, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")
	mockBootEnvs := []BootEnv{
		{ID: "default", Name: "default", Active: BootEnvActiveReboot, Mountpoint: "-", Space: "10.2G", Created: TrueNASTime{created1}},
		{ID: "backup", Name: "backup", Active: BootEnvActiveNone, Mountpoint: "-", Space: "5.1G", Created: TrueNASTime{created2}},
	}
	server.SetResponse("bootenv.query", mockBootEnvs)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	bootenvs, err := client.System.ListBootEnvs(ctx)
	require.NoError(t, err)
	assert.Len(t, bootenvs, 2)
	assert.Equal(t, "default", bootenvs[0].Name)
	assert.Equal(t, BootEnvActiveReboot, bootenvs[0].Active)
}

func TestSystemClient_CreateBootEnv(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	created3, _ := time.Parse(time.RFC3339, "2023-01-03T00:00:00Z")
	mockBootEnv := &BootEnv{
		ID:         "test-env",
		Name:       "test-env",
		Active:     "-",
		Mountpoint: "-",
		Space:      "5.0G",
		Created:    TrueNASTime{created3},
	}
	server.SetResponse("bootenv.create", mockBootEnv)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	bootenv, err := client.System.CreateBootEnv(ctx, "test-env", "default")
	require.NoError(t, err)
	require.NotNil(t, bootenv)
	assert.Equal(t, "test-env", bootenv.Name)
}

func TestSystemClient_DeleteBootEnv(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("bootenv.delete", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.System.DeleteBootEnv(ctx, "test-env")
	assert.NoError(t, err)
}

func TestSystemClient_ActivateBootEnv(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("bootenv.activate", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.System.ActivateBootEnv(ctx, "test-env")
	assert.NoError(t, err)
}

func TestSystemClient_SetBootEnvAttr(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("bootenv.set_attribute", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	attrs := map[string]any{
		"keep": true,
	}

	ctx := NewTestContext(t)
	err := client.System.SetBootEnvAttr(ctx, "test-env", attrs)
	assert.NoError(t, err)
}

func TestSystemClient_GetUpdateConfig(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := &UpdateConfig{
		AutoCheck: true,
		Train:     "stable",
	}
	server.SetResponse("update.config", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	config, err := client.System.GetUpdateConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.True(t, config.AutoCheck)
	assert.Equal(t, "stable", config.Train)
}

func TestSystemClient_CheckForUpdate(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInfo := &UpdateInfo{
		Status:     UpdateStatusAvailable,
		Version:    "23.10.3",
		ChangeLog:  "Bug fixes and improvements",
		Notice:     "",
		Notes:      "",
		Available:  true,
		Downloaded: false,
	}
	server.SetResponse("update.check_available", mockInfo)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	info, err := client.System.CheckForUpdate(ctx)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, UpdateStatusAvailable, info.Status)
	assert.Equal(t, "23.10.3", info.Version)
	assert.True(t, info.Available)
}

func TestSystemClient_GetPendingUpdate(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInfo := &UpdateInfo{
		Status:     UpdateStatusAvailable,
		Version:    "23.10.3",
		ChangeLog:  "Bug fixes and improvements",
		Notice:     "",
		Notes:      "",
		Available:  true,
		Downloaded: false,
	}
	server.SetResponse("update.get_pending", mockInfo)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	info, err := client.System.GetPendingUpdate(ctx)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "23.10.3", info.Version)
}

func TestSystemClient_DownloadUpdate(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("update.download", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.System.DownloadUpdate(ctx)
	assert.NoError(t, err)
}

func TestSystemClient_ManualUpdate(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("update.manual", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.System.ManualUpdate(ctx, "/tmp/update.tar", false)
	assert.NoError(t, err)
}

func TestSystemClient_GetTrains(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockTrains := map[string]any{
		"stable": map[string]any{
			"description": "Stable releases",
			"sequence":    "23.10",
		},
		"nightly": map[string]any{
			"description": "Nightly builds",
			"sequence":    "24.04",
		},
	}
	server.SetResponse("update.get_trains", mockTrains)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	trains, err := client.System.GetTrains(ctx)
	require.NoError(t, err)
	assert.Contains(t, trains, "stable")
	assert.Contains(t, trains, "nightly")
}

func TestSystemClient_SetTrain(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("update.set_train", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.System.SetTrain(ctx, "nightly")
	assert.NoError(t, err)
}

func TestSystemClient_ErrorHandling(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("system.info", 500, "System unavailable")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.System.GetInfo(ctx)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "System unavailable", apiErr.Message)
}
