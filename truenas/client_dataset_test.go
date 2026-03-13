package truenas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatasetClient_List(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDatasets := []Dataset{
		{ID: "tank", Name: "tank", Pool: "tank", Type: "FILESYSTEM"},
		{ID: "tank/test", Name: "tank/test", Pool: "tank", Type: "FILESYSTEM"},
	}
	server.SetResponse("pool.dataset.query", mockDatasets)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	datasets, err := client.Dataset.List(ctx)
	require.NoError(t, err)
	assert.Len(t, datasets, 2)
	assert.Equal(t, "tank", datasets[0].Name)
	assert.Equal(t, "tank/test", datasets[1].Name)
}

func TestDatasetClient_Get(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDataset := Dataset{ID: "tank/test", Name: "tank/test", Pool: "tank", Type: "FILESYSTEM"}
	server.SetResponse("pool.dataset.query", []Dataset{mockDataset})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	dataset, err := client.Dataset.Get(ctx, "tank/test")
	require.NoError(t, err)
	require.NotNil(t, dataset)
	assert.Equal(t, "tank/test", dataset.Name)
	assert.Equal(t, "tank", dataset.Pool)
}

func TestDatasetClient_Get_NotFound(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("pool.dataset.query", []Dataset{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	dataset, err := client.Dataset.Get(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, dataset)

	var notFoundErr *NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
	assert.Equal(t, "dataset", notFoundErr.ResourceType)
}

func TestDatasetClient_GetByName(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDataset := Dataset{ID: "tank/test", Name: "tank/test", Pool: "tank", Type: "FILESYSTEM"}
	server.SetResponse("pool.dataset.query", []Dataset{mockDataset})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	dataset, err := client.Dataset.GetByName(ctx, "tank/test")
	require.NoError(t, err)
	require.NotNil(t, dataset)
	assert.Equal(t, "tank/test", dataset.Name)
}

func TestDatasetClient_Create(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDataset := Dataset{ID: "tank/newds", Name: "tank/newds", Pool: "tank", Type: "FILESYSTEM"}
	server.SetResponse("pool.dataset.create", mockDataset)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &DatasetCreateRequest{
		Name: "tank/newds",
		Type: "FILESYSTEM",
		Properties: map[string]any{
			"comments":        "Test dataset",
			"sync":            "standard",
			"compression":     "off",
			"atime":           "on",
			"exec":            "on",
			"quota":           0,
			"refquota":        0,
			"reservation":     0,
			"refreservation":  0,
			"copies":          1,
			"snapdir":         "hidden",
			"dedup":           "off",
			"readonly":        "off",
			"recordsize":      "128K",
			"casesensitivity": "sensitive",
			"aclmode":         "discard",
			"acltype":         "off",
			"xattr":           "on",
		},
		UserProperties: map[string]string{
			"com.example:backup": "true",
		},
		Encryption: Ptr(false),
		Inherit:    Ptr(true),
	}

	ctx := NewTestContext(t)
	dataset, err := client.Dataset.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, dataset)
	assert.Equal(t, "tank/newds", dataset.Name)
}

func TestDatasetClient_Update(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockDataset := Dataset{ID: "tank/test", Name: "tank/test", Pool: "tank", Type: "FILESYSTEM"}
	server.SetResponse("pool.dataset.update", mockDataset)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := DatasetUpdateRequest{
		Properties: map[string]any{
			"comments":    "Updated comments",
			"compression": "lz4",
			"quota":       1073741824, // 1GB
		},
	}

	ctx := NewTestContext(t)
	dataset, err := client.Dataset.Update(ctx, "tank/test", req)
	require.NoError(t, err)
	require.NotNil(t, dataset)
	assert.Equal(t, "tank/test", dataset.Name)
}

func TestDatasetClient_Delete(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("pool.dataset.delete", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := DatasetDeleteRequest{
		Recursive: Ptr(false),
		Force:     Ptr(false),
	}

	ctx := NewTestContext(t)
	err := client.Dataset.Delete(ctx, "tank/test", req)
	assert.NoError(t, err)
}

func TestDatasetClient_Lock(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("pool.dataset.lock", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := DatasetLockRequest{
		PassPhrase:  "secret123",
		ForceUmount: Ptr(false),
	}

	ctx := NewTestContext(t)
	err := client.Dataset.Lock(ctx, "tank/encrypted", req)
	assert.NoError(t, err)
}

func TestDatasetClient_Unlock(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("pool.dataset.unlock", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := DatasetUnlockRequest{
		Datasets: []DatasetUnlockEntry{
			{
				Name:       "tank/encrypted",
				PassPhrase: "secret123",
			},
		},
		Services: Ptr(true),
	}

	ctx := NewTestContext(t)
	err := client.Dataset.Unlock(ctx, "tank/encrypted", req)
	assert.NoError(t, err)
}

func TestDatasetClient_Mount(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("pool.dataset.mount", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Dataset.Mount(ctx, "tank/test")
	assert.NoError(t, err)
}

func TestDatasetClient_Unmount(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("pool.dataset.umount", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Dataset.Unmount(ctx, "tank/test", false)
	assert.NoError(t, err)
}

func TestDatasetClient_Snapshot(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockSnapshot := map[string]any{
		"name": "tank/test@snap1",
		"id":   "tank/test@snap1",
	}
	server.SetResponse("zfs.snapshot.create", mockSnapshot)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := DatasetSnapshotRequest{
		Dataset:   "tank/test",
		Name:      "snap1",
		Recursive: Ptr(false),
		Properties: map[string]any{
			"com.example:backup": "true",
		},
	}

	ctx := NewTestContext(t)
	result, err := client.Dataset.Snapshot(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDatasetClient_GetSnapshots(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockSnapshots := []map[string]any{
		{
			"name": "tank/test@snap1",
			"id":   "tank/test@snap1",
		},
		{
			"name": "tank/test@snap2",
			"id":   "tank/test@snap2",
		},
	}
	server.SetResponse("zfs.snapshot.query", mockSnapshots)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	snapshots, err := client.Dataset.GetSnapshots(ctx, "tank/test")
	require.NoError(t, err)
	assert.Len(t, snapshots, 2)
}

func TestDatasetClient_Promote(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("pool.dataset.promote", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Dataset.Promote(ctx, "tank/clone")
	assert.NoError(t, err)
}

func TestDatasetClient_GetProcesses(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockProcesses := []map[string]any{
		{"pid": 1234, "name": "test-process", "cmdline": "/usr/bin/test"},
	}
	server.SetResponse("pool.dataset.processes", mockProcesses)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	processes, err := client.Dataset.GetProcesses(ctx, "tank/test")
	require.NoError(t, err)
	assert.NotNil(t, processes)
}

func TestDatasetClient_ErrorHandling(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("pool.dataset.query", 404, "Dataset not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Dataset.List(ctx)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "Dataset not found", apiErr.Message)
}
