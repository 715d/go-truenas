package truenas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVMClient_List(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockVMs := []VM{
		{ID: 1, Name: "test-vm-1", VCPUs: 2, Memory: 1024, Autostart: false},
		{ID: 2, Name: "test-vm-2", VCPUs: 4, Memory: 2048, Autostart: true},
	}
	server.SetResponse("vm.query", mockVMs)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	vms, err := client.VM.List(ctx)
	require.NoError(t, err)
	assert.Len(t, vms, 2)
	assert.Equal(t, "test-vm-1", vms[0].Name)
	assert.Equal(t, 2, vms[0].VCPUs)
	assert.Equal(t, 1024, vms[0].Memory)
}

func TestVMClient_Get(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockVM := VM{ID: 1, Name: "test-vm", VCPUs: 2, Memory: 1024, Autostart: false}
	server.SetResponse("vm.query", []VM{mockVM})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	vm, err := client.VM.Get(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, vm)
	assert.Equal(t, "test-vm", vm.Name)
	assert.Equal(t, 1, vm.ID)
}

func TestVMClient_Create(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockVM := VM{ID: 1, Name: "new-vm", VCPUs: 2, Memory: 1024, Autostart: false}
	server.SetResponse("vm.create", mockVM)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &VMCreateRequest{
		Name:            "new-vm",
		Description:     "Test VM",
		VCPUs:           2,
		Memory:          1024,
		Bootloader:      "UEFI",
		Autostart:       Ptr(false),
		Time:            "LOCAL",
		ShutdownTimeout: 10,
		Devices:         []VMDevice{},
	}

	ctx := NewTestContext(t)
	vm, err := client.VM.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, vm)
	assert.Equal(t, "new-vm", vm.Name)
	assert.Equal(t, 2, vm.VCPUs)
}

func TestVMClient_Update(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockVM := VM{ID: 1, Name: "updated-vm", VCPUs: 4, Memory: 2048, Autostart: true}
	server.SetResponse("vm.update", mockVM)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &VMUpdateRequest{
		Description: "Updated VM",
		VCPUs:       4,
		Memory:      2048,
		Autostart:   Ptr(true),
	}

	ctx := NewTestContext(t)
	vm, err := client.VM.Update(ctx, 1, req)
	require.NoError(t, err)
	require.NotNil(t, vm)
	assert.Equal(t, 4, vm.VCPUs)
	assert.Equal(t, 2048, vm.Memory)
	assert.True(t, vm.Autostart)
}

func TestVMClient_Delete(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("vm.delete", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &VMDeleteRequest{
		Zvols: Ptr(true),
		Force: Ptr(false),
	}

	ctx := NewTestContext(t)
	err := client.VM.Delete(ctx, 1, req)
	assert.NoError(t, err)
}

func TestVMClient_Clone(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockVM := VM{ID: 2, Name: "cloned-vm", VCPUs: 2, Memory: 1024, Autostart: false}
	server.SetResponse("vm.clone", mockVM)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	vm, err := client.VM.Clone(ctx, 1, "cloned-vm")
	require.NoError(t, err)
	require.NotNil(t, vm)
	assert.Equal(t, "cloned-vm", vm.Name)
	assert.Equal(t, 2, vm.ID)
}

func TestVMClient_Start(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("vm.start", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &VMStartRequest{
		Overcommit: Ptr(false),
	}

	ctx := NewTestContext(t)
	err := client.VM.Start(ctx, 1, req)
	assert.NoError(t, err)
}

func TestVMClient_Stop(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("vm.stop", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &VMStopRequest{
		Force:             Ptr(false),
		ForceAfterTimeout: Ptr(true),
	}

	ctx := NewTestContext(t)
	err := client.VM.Stop(ctx, 1, req)
	assert.NoError(t, err)
}

func TestVMClient_PowerOff(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("vm.poweroff", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.VM.PowerOff(ctx, 1)
	assert.NoError(t, err)
}

func TestVMClient_Restart(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("vm.restart", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.VM.Restart(ctx, 1)
	assert.NoError(t, err)
}

func TestVMClient_GetStatus(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockStatus := &VMStatus{
		State: VMStateRunning,
		PID:   12345,
	}
	server.SetResponse("vm.status", mockStatus)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	status, err := client.VM.GetStatus(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, VMStateRunning, status.State)
	assert.Equal(t, 12345, status.PID)
}

func TestVMClient_GetFlags(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockFlags := map[string]any{
		"intel_vmx": true,
		"amd_svm":   false,
		"rdtscp":    true,
	}
	server.SetResponse("vm.flags", mockFlags)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	flags, err := client.VM.GetFlags(ctx)
	require.NoError(t, err)
	assert.Contains(t, flags, "intel_vmx")
	assert.True(t, flags["intel_vmx"].(bool))
}

func TestVMClient_GetAvailableMemory(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("vm.get_available_memory", 8192)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	memory, err := client.VM.GetAvailableMemory(ctx, false)
	require.NoError(t, err)
	assert.Equal(t, 8192, memory)
}

func TestVMClient_GetMemoryInUse(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInfo := &VMMemoryInfo{
		RNP:  2048,
		PRD:  1024,
		RPRD: 6144,
	}
	server.SetResponse("vm.get_vmemory_in_use", mockInfo)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	info, err := client.VM.GetMemoryInUse(ctx)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, 2048, info.RNP)
	assert.Equal(t, 1024, info.PRD)
	assert.Equal(t, 6144, info.RPRD)
}

func TestVMClient_GetAttachedInterfaces(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInterfaces := []string{"vtnet0", "vtnet1"}
	server.SetResponse("vm.get_attached_iface", mockInterfaces)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	interfaces, err := client.VM.GetAttachedInterfaces(ctx, 1)
	require.NoError(t, err)
	assert.Contains(t, interfaces, "vtnet0")
	assert.Contains(t, interfaces, "vtnet1")
}

func TestVMClient_GetConsole(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("vm.get_console", "/dev/nmdm0A")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	console, err := client.VM.GetConsole(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, "/dev/nmdm0A", console)
}

func TestVMClient_GetVNC(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockVNC := []map[string]any{
		{"port": 5900, "host": "0.0.0.0"},
	}
	server.SetResponse("vm.get_vnc", mockVNC)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	vnc, err := client.VM.GetVNC(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, vnc, 1)
	assert.Equal(t, float64(5900), vnc[0]["port"])
}

func TestVMClient_GetVNCWeb(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockURLs := []string{"http://192.168.1.100:6080/vnc.html?host=192.168.1.100&port=5900"}
	server.SetResponse("vm.get_vnc_web", mockURLs)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	urls, err := client.VM.GetVNCWeb(ctx, 1, "192.168.1.100")
	require.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Contains(t, urls[0], "vnc.html")
}

func TestVMClient_GetVNCIPv4(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockIPs := []string{"192.168.1.100", "10.0.0.100"}
	server.SetResponse("vm.get_vnc_ipv4", mockIPs)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	ips, err := client.VM.GetVNCIPv4(ctx)
	require.NoError(t, err)
	assert.Contains(t, ips, "192.168.1.100")
	assert.Contains(t, ips, "10.0.0.100")
}

func TestVMClient_GetVNCPortWizard(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockWizard := map[string]any{"port": 5901}
	server.SetResponse("vm.vnc_port_wizard", mockWizard)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	wizard, err := client.VM.GetVNCPortWizard(ctx)
	require.NoError(t, err)
	assert.NotNil(t, wizard)
}

func TestVMClient_GenerateRandomMAC(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("vm.random_mac", "00:a0:98:12:34:56")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	mac, err := client.VM.GenerateRandomMAC(ctx)
	require.NoError(t, err)
	assert.Equal(t, "00:a0:98:12:34:56", mac)
}

func TestVMClient_IdentifyHypervisor(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("vm.identify_hypervisor", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	isHypervisor, err := client.VM.IdentifyHypervisor(ctx)
	require.NoError(t, err)
	assert.True(t, isHypervisor)
}

// VMDeviceClient Tests
func TestVMDeviceClient_List(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDevices := []VMDevice{
		{ID: 1, VM: 1, DType: VMDeviceTypeNIC, Order: 1000},
		{ID: 2, VM: 1, DType: VMDeviceTypeDisk, Order: 1001},
	}
	server.SetResponse("vm.device.query", mockDevices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	devices, err := client.VMDevice.List(ctx)
	require.NoError(t, err)
	assert.Len(t, devices, 2)
	assert.Equal(t, VMDeviceTypeNIC, devices[0].DType)
	assert.Equal(t, VMDeviceTypeDisk, devices[1].DType)
}

func TestVMDeviceClient_Get(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDevice := VMDevice{ID: 1, VM: 1, DType: VMDeviceTypeNIC, Order: 1000}
	server.SetResponse("vm.device.query", []VMDevice{mockDevice})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	device, err := client.VMDevice.Get(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, device)
	assert.Equal(t, VMDeviceTypeNIC, device.DType)
	assert.Equal(t, 1, device.ID)
}

func TestVMDeviceClient_Create(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDevice := VMDevice{ID: 1, VM: 1, DType: VMDeviceTypeNIC, Order: 1000}
	server.SetResponse("vm.device.create", mockDevice)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &VMDeviceCreateRequest{
		VM:    1,
		DType: VMDeviceTypeNIC,
		Order: 1000,
		Attributes: map[string]any{
			"type":       "E1000",
			"mac":        "00:a0:98:12:34:56",
			"nic_attach": "br0",
		},
	}

	ctx := NewTestContext(t)
	device, err := client.VMDevice.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, device)
	assert.Equal(t, VMDeviceTypeNIC, device.DType)
}

func TestVMDeviceClient_Update(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDevice := VMDevice{ID: 1, VM: 1, DType: VMDeviceTypeNIC, Order: 1000}
	server.SetResponse("vm.device.update", mockDevice)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &VMDeviceCreateRequest{
		Order: 1001,
		Attributes: map[string]any{
			"type": "VIRTIO",
		},
	}

	ctx := NewTestContext(t)
	device, err := client.VMDevice.Update(ctx, 1, req)
	require.NoError(t, err)
	require.NotNil(t, device)
	assert.Equal(t, VMDeviceTypeNIC, device.DType)
}

func TestVMDeviceClient_Delete(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("vm.device.delete", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &VMDeviceDeleteRequest{
		Zvol:    Ptr(false),
		RawFile: Ptr(false),
	}

	ctx := NewTestContext(t)
	err := client.VMDevice.Delete(ctx, 1, req)
	assert.NoError(t, err)
}

func TestVMDeviceClient_GetNICAttachChoices(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockChoices := map[string]any{
		"br0":    "Bridge (br0)",
		"vtnet0": "Interface (vtnet0)",
	}
	server.SetResponse("vm.device.nic_attach_choices", mockChoices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	choices, err := client.VMDevice.GetNICAttachChoices(ctx)
	require.NoError(t, err)
	assert.Contains(t, choices, "br0")
	assert.Contains(t, choices, "vtnet0")
}

func TestVMDeviceClient_GetPPTDevChoices(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockChoices := map[string]any{
		"ppt0": "Network Controller",
		"ppt1": "Graphics Controller",
	}
	server.SetResponse("vm.device.pptdev_choices", mockChoices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	choices, err := client.VMDevice.GetPPTDevChoices(ctx)
	require.NoError(t, err)
	assert.Contains(t, choices, "ppt0")
	assert.Contains(t, choices, "ppt1")
}

func TestVMDeviceClient_GetVNCBindChoices(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockChoices := map[string]any{
		"0.0.0.0":     "All Interfaces",
		"192.168.1.1": "LAN Interface",
	}
	server.SetResponse("vm.device.vnc_bind_choices", mockChoices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	choices, err := client.VMDevice.GetVNCBindChoices(ctx)
	require.NoError(t, err)
	assert.Contains(t, choices, "0.0.0.0")
	assert.Contains(t, choices, "192.168.1.1")
}

func TestVMClient_ErrorHandling(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("vm.query", 500, "VM service unavailable")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.VM.List(ctx)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "VM service unavailable", apiErr.Message)
}
