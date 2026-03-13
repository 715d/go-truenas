package truenas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolClient_List(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockPools := []Pool{
		{ID: 1, Name: "tank", Path: "/mnt/tank"},
		{ID: 2, Name: "backup", Path: "/mnt/backup"},
	}
	server.SetResponse("pool.query", mockPools)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	pools, err := client.Pool.List(ctx)
	require.NoError(t, err)
	assert.Len(t, pools, 2)
	assert.Equal(t, "tank", pools[0].Name)
	assert.Equal(t, "backup", pools[1].Name)
}

func TestPoolClient_Get(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockPool := Pool{ID: 1, Name: "tank", Path: "/mnt/tank"}
	server.SetResponse("pool.query", []Pool{mockPool})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	pool, err := client.Pool.Get(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, pool)
	assert.Equal(t, "tank", pool.Name)
	assert.Equal(t, 1, pool.ID)
}

func TestPoolClient_Get_NotFound(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("pool.query", []Pool{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	pool, err := client.Pool.Get(ctx, 999)
	assert.Error(t, err)
	assert.Nil(t, pool)

	var notFoundErr *NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
	assert.Equal(t, "pool", notFoundErr.ResourceType)
}

func TestPoolClient_GetByName(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockPool := Pool{ID: 1, Name: "tank", Path: "/mnt/tank"}
	server.SetResponse("pool.query", []Pool{mockPool})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	pool, err := client.Pool.GetByName(ctx, "tank")
	require.NoError(t, err)
	require.NotNil(t, pool)
	assert.Equal(t, "tank", pool.Name)
	assert.Equal(t, 1, pool.ID)
}

func TestPoolClient_Create(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockPool := Pool{ID: 1, Name: "newpool", Path: "/mnt/newpool"}
	server.SetJobResponse("pool.create", mockPool)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := PoolCreateRequest{
		Name:       "newpool",
		Encryption: false,
		Topology: PoolTopologyCreate{
			Data: []VDevCreate{{
				Type:  VDevTypeRaidz,
				Disks: []string{"sda", "sdb", "sdc"},
			}},
		},
	}

	ctx := NewTestContext(t)
	pool, err := client.Pool.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, pool)
	assert.Equal(t, "newpool", pool.Name)
	assert.Equal(t, 1, pool.ID)
}

func TestPoolClient_Update(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockPool := Pool{ID: 1, Name: "tank", Path: "/mnt/tank"}
	server.SetJobResponse("pool.update", mockPool)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := PoolUpdateRequest{
		Topology: &PoolTopology{
			Data: []VDev{{
				Type: VDevTypeRaidz1,
				Children: []VDev{
					{Type: VDevTypeDisk, Disk: Ptr("sda")},
					{Type: VDevTypeDisk, Disk: Ptr("sdb")},
					{Type: VDevTypeDisk, Disk: Ptr("sdc")},
					{Type: VDevTypeDisk, Disk: Ptr("sdd")},
				},
			}},
		},
	}

	ctx := NewTestContext(t)
	pool, err := client.Pool.Update(ctx, 1, req)
	require.NoError(t, err)
	require.NotNil(t, pool)
	assert.Equal(t, "tank", pool.Name)
}

func TestPoolClient_Delete(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("pool.export", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Pool.Delete(ctx, 1, false)
	assert.NoError(t, err)
}

func TestPoolClient_Export(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("pool.export", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := PoolExportRequest{
		Cascade:        false,
		RestartService: false,
	}

	ctx := NewTestContext(t)
	err := client.Pool.Export(ctx, 1, req)
	assert.NoError(t, err)
}

func TestPoolClient_Import(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockPool := Pool{ID: 1, Name: "imported", Path: "/mnt/imported"}
	server.SetJobResponse("pool.import_pool", mockPool)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := PoolImportRequest{
		GUID: "12345678901234567890",
		Name: "imported",
	}

	ctx := NewTestContext(t)
	pool, err := client.Pool.Import(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, pool)
	assert.Equal(t, "imported", pool.Name)
}

func TestPoolClient_Scrub(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("pool.scrub", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Pool.Scrub(ctx, 1, PoolScrubActionStart)
	assert.NoError(t, err)
}

func TestPoolClient_GetProcesses(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockProcesses := []map[string]any{
		{"pid": 1234, "name": "test-process"},
	}
	server.SetResponse("pool.processes", mockProcesses)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	processes, err := client.Pool.GetProcesses(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, processes)
}

func TestPoolClient_FindImportablePools(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockPools := []PoolImportFindResult{
		{GUID: "12345", Name: "tank", Status: "ONLINE"},
	}
	server.SetJobResponse("pool.import_find", mockPools)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	pools, err := client.Pool.FindImportablePools(ctx)
	require.NoError(t, err)
	assert.Len(t, pools, 1)
	assert.Equal(t, "tank", pools[0].Name)
}

func TestPoolClient_ErrorHandling(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("pool.query", 500, "Internal server error")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Pool.List(ctx)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "Internal server error", apiErr.Message)
}

func TestPoolClient_ListScrubTasks(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockTasks := []PoolScrubTask{
		{
			ID:          1,
			Pool:        1,
			Threshold:   35,
			Description: "Weekly scrub",
			Schedule: CronSchedule{
				Minute: "0",
				Hour:   "3",
				Dom:    "*",
				Month:  "*",
				Dow:    "7",
			},
			Enabled: true,
		},
	}
	server.SetResponse("pool.scrub.query", mockTasks)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	tasks, err := client.Pool.ListScrubTasks(ctx)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Weekly scrub", tasks[0].Description)
	assert.Equal(t, 35, tasks[0].Threshold)
}

func TestPoolClient_CreateScrubTask(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockTask := PoolScrubTask{
		ID:          2,
		Pool:        1,
		Threshold:   35,
		Description: "Monthly scrub",
		Schedule: CronSchedule{
			Minute: "0",
			Hour:   "3",
			Dom:    "1",
			Month:  "*",
			Dow:    "*",
		},
		Enabled: true,
	}
	server.SetResponse("pool.scrub.create", mockTask)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := PoolScrubTaskRequest{
		Pool:        1,
		Threshold:   35,
		Description: "Monthly scrub",
		Schedule: CronSchedule{
			Minute: "0",
			Hour:   "3",
			Dom:    "1",
			Month:  "*",
			Dow:    "*",
		},
		Enabled: true,
	}

	ctx := NewTestContext(t)
	task, err := client.Pool.CreateScrubTask(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "Monthly scrub", task.Description)
	assert.Equal(t, 2, task.ID)
}

func TestPoolClient_RunScrub(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("pool.scrub.scrub", 123)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	jobID, err := client.Pool.RunScrubAsync(ctx, "tank", "START")
	require.NoError(t, err)
	assert.Equal(t, 123, jobID)
}

func TestPoolClient_GetScrubTask(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockTask := PoolScrubTask{
		ID:          1,
		Pool:        1,
		Threshold:   35,
		Description: "Weekly scrub",
		Schedule: CronSchedule{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "7",
		},
		Enabled: true,
	}
	server.SetResponse("pool.scrub.query", []PoolScrubTask{mockTask})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	task, err := client.Pool.GetScrubTask(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, 1, task.ID)
	assert.Equal(t, 1, task.Pool)
	assert.Equal(t, "Weekly scrub", task.Description)
	assert.Equal(t, 35, task.Threshold)
	assert.True(t, task.Enabled)
	assert.Equal(t, "0", task.Schedule.Minute)
	assert.Equal(t, "3", task.Schedule.Hour)
	assert.Equal(t, "7", task.Schedule.Dow)
}

func TestPoolClient_GetScrubTask_NotFound(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("pool.scrub.query", []PoolScrubTask{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	task, err := client.Pool.GetScrubTask(ctx, 999)
	assert.Error(t, err)
	assert.Nil(t, task)

	var notFoundErr *NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
	assert.Equal(t, "scrub_task", notFoundErr.ResourceType)
	assert.Equal(t, "ID 999", notFoundErr.Identifier)
}

func TestPoolClient_GetScrubTask_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("pool.scrub.query", 500, "Internal server error")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	task, err := client.Pool.GetScrubTask(ctx, 1)
	require.Error(t, err)
	assert.Nil(t, task)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "Internal server error", apiErr.Message)
}

func TestPoolClient_GetScrubTasksByPool(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockTasks := []PoolScrubTask{
		{
			ID:          1,
			Pool:        1,
			Threshold:   35,
			Description: "Weekly scrub for tank",
			Schedule: CronSchedule{
				Minute: "0",
				Hour:   "3",
				Dom:    "*",
				Month:  "*",
				Dow:    "7",
			},
			Enabled: true,
		},
		{
			ID:          2,
			Pool:        1,
			Threshold:   40,
			Description: "Monthly scrub for tank",
			Schedule: CronSchedule{
				Minute: "0",
				Hour:   "4",
				Dom:    "1",
				Month:  "*",
				Dow:    "*",
			},
			Enabled: false,
		},
	}
	server.SetResponse("pool.scrub.query", mockTasks)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	tasks, err := client.Pool.GetScrubTasksByPool(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "Weekly scrub for tank", tasks[0].Description)
	assert.Equal(t, "Monthly scrub for tank", tasks[1].Description)
	assert.Equal(t, 1, tasks[0].Pool)
	assert.Equal(t, 1, tasks[1].Pool)
	assert.True(t, tasks[0].Enabled)
	assert.False(t, tasks[1].Enabled)
}

func TestPoolClient_GetScrubTasksByPool_Empty(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("pool.scrub.query", []PoolScrubTask{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	tasks, err := client.Pool.GetScrubTasksByPool(ctx, 999)
	require.NoError(t, err)
	assert.Len(t, tasks, 0)
}

func TestPoolClient_GetScrubTasksByPool_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("pool.scrub.query", 404, "Pool not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	tasks, err := client.Pool.GetScrubTasksByPool(ctx, 999)
	require.Error(t, err)
	assert.Nil(t, tasks)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "Pool not found", apiErr.Message)
}

func TestPoolClient_UpdateScrubTask(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockTask := PoolScrubTask{
		ID:          1,
		Pool:        1,
		Threshold:   40,
		Description: "Updated weekly scrub",
		Schedule: CronSchedule{
			Minute: "0",
			Hour:   "4",
			Dom:    "*",
			Month:  "*",
			Dow:    "1",
		},
		Enabled: false,
	}
	server.SetResponse("pool.scrub.update", mockTask)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := PoolScrubTaskRequest{
		Pool:        1,
		Threshold:   40,
		Description: "Updated weekly scrub",
		Schedule: CronSchedule{
			Minute: "0",
			Hour:   "4",
			Dom:    "*",
			Month:  "*",
			Dow:    "1",
		},
		Enabled: false,
	}

	ctx := NewTestContext(t)
	task, err := client.Pool.UpdateScrubTask(ctx, 1, req)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, 1, task.ID)
	assert.Equal(t, 1, task.Pool)
	assert.Equal(t, "Updated weekly scrub", task.Description)
	assert.Equal(t, 40, task.Threshold)
	assert.False(t, task.Enabled)
	assert.Equal(t, "0", task.Schedule.Minute)
	assert.Equal(t, "4", task.Schedule.Hour)
	assert.Equal(t, "1", task.Schedule.Dow)
}

func TestPoolClient_UpdateScrubTask_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("pool.scrub.update", 404, "Scrub task not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	req := PoolScrubTaskRequest{
		Pool:        1,
		Threshold:   35,
		Description: "Updated scrub",
		Schedule: CronSchedule{
			Minute: "0",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "7",
		},
		Enabled: true,
	}

	ctx := NewTestContext(t)
	task, err := client.Pool.UpdateScrubTask(ctx, 999, req)
	require.Error(t, err)
	require.NotNil(t, task)
	assert.Equal(t, 0, task.ID) // Zero-valued struct on error

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "Scrub task not found", apiErr.Message)
}

func TestPoolClient_DeleteScrubTask(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("pool.scrub.delete", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Pool.DeleteScrubTask(ctx, 1)
	assert.NoError(t, err)
}

func TestPoolClient_DeleteScrubTask_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("pool.scrub.delete", 404, "Scrub task not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Pool.DeleteScrubTask(ctx, 999)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "Scrub task not found", apiErr.Message)
}

func TestPoolClient_DeleteScrubTask_MultipleCalls(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("pool.scrub.delete", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)

	// First deletion should succeed
	err := client.Pool.DeleteScrubTask(ctx, 1)
	assert.NoError(t, err)

	// Set up error for second call (simulating task already deleted)
	server.SetError("pool.scrub.delete", 404, "Scrub task not found")

	// Second deletion should fail
	err = client.Pool.DeleteScrubTask(ctx, 1)
	require.Error(t, err)
	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
}
