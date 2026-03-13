package truenas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceClient_List(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockServices := []Service{
		{ID: 1, Service: "ssh", Enable: true, State: "RUNNING"},
		{ID: 2, Service: "nfs", Enable: false, State: "STOPPED"},
	}
	server.SetResponse("service.query", mockServices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	services, err := client.Service.List(ctx)
	require.NoError(t, err)
	assert.Len(t, services, 2)
	assert.Equal(t, "ssh", services[0].Service)
	assert.Equal(t, "RUNNING", services[0].State)
	assert.True(t, services[0].Enable)
}

func TestServiceClient_Get(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockService := Service{ID: 1, Service: "ssh", Enable: true, State: "RUNNING"}
	server.SetResponse("service.query", []Service{mockService})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	service, err := client.Service.Get(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, service)
	assert.Equal(t, "ssh", service.Service)
	assert.Equal(t, "RUNNING", service.State)
}

func TestServiceClient_Get_NotFound(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("service.query", []Service{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	service, err := client.Service.Get(ctx, 999)
	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "not found")
}

func TestServiceClient_GetByName(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockService := Service{ID: 1, Service: "ssh", Enable: true, State: "RUNNING"}
	server.SetResponse("service.query", []Service{mockService})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	service, err := client.Service.GetByName(ctx, "ssh")
	require.NoError(t, err)
	require.NotNil(t, service)
	assert.Equal(t, "ssh", service.Service)
	assert.Equal(t, 1, service.ID)
}

func TestServiceClient_Update(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockService := Service{ID: 1, Service: "ssh", Enable: false, State: "STOPPED"}
	// service.update returns just the ID
	server.SetResponse("service.update", 1)
	// The subsequent Get call returns the full service
	server.SetResponse("service.query", []Service{mockService})

	client := server.CreateTestClient(t)
	defer client.Close()

	req := ServiceUpdateRequest{
		Enable: false,
	}

	ctx := NewTestContext(t)
	service, err := client.Service.Update(ctx, 1, req)
	require.NoError(t, err)
	require.NotNil(t, service)
	assert.False(t, service.Enable)
}

func TestServiceClient_Start(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("service.start", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Service.Start(ctx, "ssh")
	assert.NoError(t, err)
}

func TestServiceClient_Start_WithOptions(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("service.start", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	options := map[string]any{
		"silent": true,
	}

	ctx := NewTestContext(t)
	err := client.Service.Start(ctx, "ssh", options)
	assert.NoError(t, err)
}

func TestServiceClient_Stop(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("service.stop", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Service.Stop(ctx, "ssh")
	assert.NoError(t, err)
}

func TestServiceClient_Restart(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("service.restart", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Service.Restart(ctx, "ssh")
	assert.NoError(t, err)
}

func TestServiceClient_Reload(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("service.reload", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Service.Reload(ctx, "ssh")
	assert.NoError(t, err)
}

func TestServiceClient_Started(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("service.started", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	started, err := client.Service.Started(ctx, "ssh")
	require.NoError(t, err)
	assert.True(t, started)
}

// SMBClient Tests
func TestSMBClient_GetConfig(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := &SMBConfig{
		NetBIOSName:     "TRUENAS",
		Workgroup:       "WORKGROUP",
		Description:     "TrueNAS Server",
		UnixCharset:     "UTF-8",
		LogLevel:        "MINIMUM",
		LocalMaster:     true,
		DomainLogons:    false,
		UseSendfile:     true,
		AAAPLExtensions: true,
		EASupport:       "auto",
	}
	server.SetResponse("smb.config", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	config, err := client.SMB.GetConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "TRUENAS", config.NetBIOSName)
	assert.Equal(t, "WORKGROUP", config.Workgroup)
	assert.True(t, config.UseSendfile)
}

func TestSMBClient_UpdateConfig(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := &SMBConfig{
		NetBIOSName: "UPDATED-TRUENAS",
		Workgroup:   "NEWGROUP",
		Description: "Updated TrueNAS Server",
		UseSendfile: false,
	}
	server.SetResponse("smb.update", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	updated, err := client.SMB.UpdateConfig(ctx, mockConfig)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "UPDATED-TRUENAS", updated.NetBIOSName)
	assert.Equal(t, "NEWGROUP", updated.Workgroup)
	assert.False(t, updated.UseSendfile)
}

// NFSClient Tests
func TestNFSClient_GetConfig(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := &NFSConfig{
		V4:             true,
		V4V3Owner:      false,
		V4KrbEnabled:   false,
		V4Domain:       "",
		BindIP:         []string{},
		MountdPort:     618,
		RpcstatdPort:   662,
		RpclockdPort:   32803,
		Servers:        16,
		UDPEnabled:     false,
		RPCGSSEnabled:  false,
		UserdMaxGroups: 16,
	}
	server.SetResponse("nfs.config", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	config, err := client.NFS.GetConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.True(t, config.V4)
	assert.False(t, config.V4V3Owner)
	assert.Equal(t, 16, config.Servers)
	assert.False(t, config.UDPEnabled)
}

func TestNFSClient_UpdateConfig(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := &NFSConfig{
		V4:         true,
		Servers:    32,
		UDPEnabled: true,
		V4V3Owner:  true,
	}
	server.SetResponse("nfs.update", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	updated, err := client.NFS.UpdateConfig(ctx, mockConfig)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.True(t, updated.V4)
	assert.Equal(t, 32, updated.Servers)
	assert.True(t, updated.UDPEnabled)
}

// SSHClient Tests
func TestSSHClient_GetConfig(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := &SSHConfig{
		BindIface:       []string{},
		TCPPort:         []int{22},
		RootLogin:       true,
		PasswordAuth:    true,
		KerberosAuth:    false,
		TCPForwarding:   true,
		Compression:     false,
		SFTPLogLevel:    "",
		SFTPLogFacility: "",
		WeakCiphers:     []string{},
		AuxParam:        "",
	}
	server.SetResponse("ssh.config", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	config, err := client.SSH.GetConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, []int{22}, config.TCPPort)
	assert.True(t, config.RootLogin)
	assert.True(t, config.PasswordAuth)
	assert.False(t, config.KerberosAuth)
}

func TestSSHClient_UpdateConfig(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := &SSHConfig{
		TCPPort:       []int{2222},
		RootLogin:     false,
		PasswordAuth:  false,
		TCPForwarding: false,
	}
	server.SetResponse("ssh.update", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	updated, err := client.SSH.UpdateConfig(ctx, mockConfig)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, []int{2222}, updated.TCPPort)
	assert.False(t, updated.RootLogin)
	assert.False(t, updated.PasswordAuth)
	assert.False(t, updated.TCPForwarding)
}

func TestServiceClient_ErrorHandling(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("service.query", 404, "Service not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Service.List(ctx)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "Service not found", apiErr.Message)
}
