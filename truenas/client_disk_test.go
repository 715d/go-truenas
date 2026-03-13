package truenas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiskClient_List(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDisks := []Disk{
		{
			Name:         "sda",
			Devname:      "sda",
			Model:        "Test SSD 1TB",
			Serial:       "TEST123456",
			Size:         1000000000000,
			Type:         DiskTypeSSD,
			Rotationrate: Ptr(0),
			HDDStandby:   "ALWAYS ON",
			AdvPowerMgmt: "DISABLED",
		},
		{
			Name:         "sdb",
			Devname:      "sdb",
			Model:        "Test HDD 2TB",
			Serial:       "TEST789012",
			Size:         2000000000000,
			Type:         DiskTypeHDD,
			Rotationrate: Ptr(7200),
			HDDStandby:   "5",
			AdvPowerMgmt: "LEVEL_128",
		},
	}
	server.SetResponse("disk.query", mockDisks)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	disks, err := client.Disk.List(ctx)
	require.NoError(t, err)
	assert.Len(t, disks, 2)
	assert.Equal(t, "sda", disks[0].Name)
	assert.Equal(t, DiskTypeSSD, disks[0].Type)
	assert.Equal(t, 0, *disks[0].Rotationrate)
	assert.Equal(t, "sdb", disks[1].Name)
	assert.Equal(t, DiskTypeHDD, disks[1].Type)
	assert.Equal(t, 7200, *disks[1].Rotationrate)
}

func TestDiskClient_ListWithOptions(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDisks := []Disk{
		{Name: "sda", Devname: "sda", Type: DiskTypeSSD},
	}
	server.SetResponse("disk.query", mockDisks)

	client := server.CreateTestClient(t)
	defer client.Close()

	opts := &DiskQueryOptions{
		Pools: true,
	}

	ctx := NewTestContext(t)
	disks, err := client.Disk.ListWithOptions(ctx, opts)
	require.NoError(t, err)
	assert.Len(t, disks, 1)
	assert.Equal(t, "sda", disks[0].Name)
}

func TestDiskClient_Get(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDisk := Disk{
		Name:         "sda",
		Devname:      "sda",
		Model:        "Test SSD 1TB",
		Serial:       "TEST123456",
		Size:         1000000000000,
		Type:         DiskTypeSSD,
		Rotationrate: Ptr(0),
	}
	server.SetResponse("disk.query", []Disk{mockDisk})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	disk, err := client.Disk.Get(ctx, "sda")
	require.NoError(t, err)
	require.NotNil(t, disk)
	assert.Equal(t, "sda", disk.Name)
	assert.Equal(t, "Test SSD 1TB", disk.Model)
	assert.Equal(t, DiskTypeSSD, disk.Type)
}

func TestDiskClient_Get_NotFound(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("disk.query", []Disk{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	disk, err := client.Disk.Get(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, disk)

	var notFoundErr *NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
	assert.Equal(t, "disk", notFoundErr.ResourceType)
}

func TestDiskClient_Update(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDisk := Disk{
		Name:         "sda",
		Devname:      "sda",
		HDDStandby:   HDDStandby30,
		AdvPowerMgmt: AdvPowerMgmt192,
	}
	server.SetResponse("disk.update", mockDisk)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &DiskUpdateRequest{
		HDDStandby:    Ptr(HDDStandby30),
		AdvPowerMgmt:  Ptr(AdvPowerMgmt192),
		Description:   Ptr("Updated disk settings"),
		Critical:      Ptr(80),
		Difference:    Ptr(15),
		Informational: Ptr(70),
		Passwd:        Ptr(""),
	}

	ctx := NewTestContext(t)
	disk, err := client.Disk.Update(ctx, "sda", req)
	require.NoError(t, err)
	require.NotNil(t, disk)
	assert.Equal(t, HDDStandby30, disk.HDDStandby)
	assert.Equal(t, AdvPowerMgmt192, disk.AdvPowerMgmt)
}

func TestDiskClient_GetEncrypted(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDevices := []EncryptedDevice{
		{
			Name:      "gptid/12345678-1234-1234-1234-123456789012",
			Status:    "LOCKED",
			Encrypted: true,
		},
		{
			Name:      "gptid/87654321-4321-4321-4321-210987654321",
			Status:    "UNLOCKED",
			Encrypted: true,
		},
	}
	server.SetResponse("disk.get_encrypted", mockDevices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	devices, err := client.Disk.GetEncrypted(ctx, true)
	require.NoError(t, err)
	assert.Len(t, devices, 2)
	assert.Equal(t, "LOCKED", devices[0].Status)
	assert.Equal(t, "UNLOCKED", devices[1].Status)
	assert.True(t, devices[0].Encrypted)
}

func TestDiskClient_Decrypt(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("disk.decrypt", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &DecryptRequest{
		Devices:    []string{"gptid/12345678-1234-1234-1234-123456789012"},
		Passphrase: Ptr("secret123"),
	}

	ctx := NewTestContext(t)
	err := client.Disk.Decrypt(ctx, req)
	assert.NoError(t, err)
}

func TestDiskClient_GetUnused(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDisks := []UnusedDisk{
		{
			Name:    "sdc",
			Devname: "sdc",
			Size:    1000000000000,
			Partitions: []Partition{
				{Name: "sdc1", Size: 500000000000},
				{Name: "sdc2", Size: 500000000000},
			},
		},
		{
			Name:       "sdd",
			Devname:    "sdd",
			Size:       2000000000000,
			Partitions: []Partition{},
		},
	}
	server.SetResponse("disk.get_unused", mockDisks)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	disks, err := client.Disk.GetUnused(ctx, true)
	require.NoError(t, err)
	assert.Len(t, disks, 2)
	assert.Equal(t, "sdc", disks[0].Name)
	assert.Len(t, disks[0].Partitions, 2)
	assert.Equal(t, "sdd", disks[1].Name)
	assert.Len(t, disks[1].Partitions, 0)
}

func TestDiskClient_LabelToDev(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("disk.label_to_dev", "sda")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	dev, err := client.Disk.LabelToDev(ctx, "gptid/12345678-1234-1234-1234-123456789012")
	require.NoError(t, err)
	assert.Equal(t, "sda", dev)
}

func TestDiskClient_GetSmartAttributes(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockAttrs := []SmartAttribute{
		{
			ID:         1,
			Name:       "Raw_Read_Error_Rate",
			Value:      200,
			Worst:      200,
			Threshold:  51,
			Type:       "Pre-fail",
			Updated:    "Always",
			WhenFailed: "",
			RawValue:   0,
		},
		{
			ID:         5,
			Name:       "Reallocated_Sector_Ct",
			Value:      200,
			Worst:      200,
			Threshold:  140,
			Type:       "Pre-fail",
			Updated:    "Always",
			WhenFailed: "",
			RawValue:   0,
		},
	}
	server.SetResponse("disk.smart_attributes", mockAttrs)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	attrs, err := client.Disk.GetSmartAttributes(ctx, "sda")
	require.NoError(t, err)
	assert.Len(t, attrs, 2)
	assert.Equal(t, "Raw_Read_Error_Rate", attrs[0].Name)
	assert.Equal(t, 1, attrs[0].ID)
	assert.Equal(t, "Reallocated_Sector_Ct", attrs[1].Name)
	assert.Equal(t, 5, attrs[1].ID)
}

func TestDiskClient_GetTemperature(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockTemp := &DiskTemperature{
		Name:        "sda",
		Temperature: Ptr(42),
		Unit:        "C",
	}
	server.SetResponse("disk.temperature", mockTemp)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	temp, err := client.Disk.GetTemperature(ctx, "sda", PowerModeNever)
	require.NoError(t, err)
	require.NotNil(t, temp)
	assert.Equal(t, "sda", temp.Name)
	assert.Equal(t, 42, *temp.Temperature)
	assert.Equal(t, "C", temp.Unit)
}

func TestDiskClient_GetTemperatures(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockTemps := []DiskTemperature{
		{Name: "sda", Temperature: Ptr(42), Unit: "C"},
		{Name: "sdb", Temperature: Ptr(38), Unit: "C"},
	}
	server.SetResponse("disk.temperatures", mockTemps)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	temps, err := client.Disk.GetTemperatures(ctx, []string{"sda", "sdb"}, PowerModeNever)
	require.NoError(t, err)
	assert.Len(t, temps, 2)
	assert.Equal(t, "sda", temps[0].Name)
	assert.Equal(t, 42, *temps[0].Temperature)
	assert.Equal(t, "sdb", temps[1].Name)
	assert.Equal(t, 38, *temps[1].Temperature)
}

func TestDiskClient_Spindown(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("disk.spindown", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Disk.Spindown(ctx, "sda")
	assert.NoError(t, err)
}

func TestDiskClient_Overprovision(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("disk.overprovision", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Disk.Overprovision(ctx, "sda", 1073741824) // 1GB
	assert.NoError(t, err)
}

func TestDiskClient_Unoverprovision(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("disk.unoverprovision", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Disk.Unoverprovision(ctx, "sda")
	assert.NoError(t, err)
}

func TestDiskClient_Wipe(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("disk.wipe", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &WipeRequest{
		Device:    "sda",
		Mode:      WipeModeQuick,
		SyncCache: true,
	}

	ctx := NewTestContext(t)
	err := client.Disk.Wipe(ctx, req)
	assert.NoError(t, err)
}

func TestDiskClient_GetSedDevName(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("disk.sed_dev_name", "/dev/da0")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	sedDev, err := client.Disk.GetSedDevName(ctx, "sda")
	require.NoError(t, err)
	assert.Equal(t, "/dev/da0", sedDev)
}

func TestDiskClient_ErrorHandling(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("disk.query", 500, "Disk service unavailable")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Disk.List(ctx)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "Disk service unavailable", apiErr.Message)
}
