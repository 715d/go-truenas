package truenas

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/715d/go-truenas/truenas/testvm"
)

// TestClientWithVM runs comprehensive tests against a real TrueNAS VM
func TestClientWithVM(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping VM integration test in short mode")
	}

	testvm.RunWithVM(t, func(helper *testvm.Manager) {
		// Create client with retries — the WebSocket API may not be ready
		// immediately after the HTTP endpoint starts responding.
		connInfo := helper.GetConnectionInfo()
		client := createClientWithRetry(t, connInfo)
		defer client.Close()

		// Wait for system to be ready
		if err := waitForSystemReady(t, client); err != nil {
			t.Fatalf("System readiness check failed: %v", err)
		}

		t.Run("ConnectionAndAuthentication", func(t *testing.T) {
			testConnectionAndAuthentication(t, client)
		})

		t.Run("SystemOperations", func(t *testing.T) {
			testSystemOperations(t, client)
		})

		t.Run("UserManagement", func(t *testing.T) {
			testUserManagement(t, client)
		})

		t.Run("ErrorHandling", func(t *testing.T) {
			testAPIErrorHandling(t, client)
		})

		t.Run("ConcurrentOperations", func(t *testing.T) {
			testConcurrentOperations(t, client)
		})

		t.Run("LongRunningOperations", func(t *testing.T) {
			testLongRunningOperations(t, client)
		})

		t.Run("PoolCreation", func(t *testing.T) {
			testPoolCreation(t, client, helper)
		})

		t.Run("RAIDZ1PoolWithDatasetAndNFS", func(t *testing.T) {
			testRAIDZ1PoolWithDatasetAndNFS(t, client, helper)
		})
	})
}

func testConnectionAndAuthentication(t *testing.T, client *Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test that we can make a basic API call
	var info SystemInfo
	err := client.Call(ctx, "system.info", nil, &info)
	require.NoError(t, err)

	// Verify the result contains expected fields
	assert.NotEmpty(t, info.Version)
	assert.NotEmpty(t, info.Hostname)
}

func testSystemOperations(t *testing.T, client *Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("SystemInfo", func(t *testing.T) {
		var info SystemInfo
		err := client.Call(ctx, "system.info", nil, &info)
		require.NoError(t, err)

		// Check for required fields
		assert.NotEmpty(t, info.Version)
		assert.NotEmpty(t, info.Hostname)
		assert.NotEmpty(t, info.Uptime)
		assert.Greater(t, info.UptimeSeconds, 0.0)
	})

	t.Run("SystemVersion", func(t *testing.T) {
		var version string
		err := client.Call(ctx, "system.version", nil, &version)
		require.NoError(t, err)
		assert.NotEmpty(t, version)
	})

	t.Run("SystemHostname", func(t *testing.T) {
		var hostname string
		err := client.Call(ctx, "system.hostname", nil, &hostname)
		require.NoError(t, err)
		assert.NotEmpty(t, hostname)
	})

	t.Run("SystemUptime", func(t *testing.T) {
		var info SystemInfo
		err := client.Call(ctx, "system.info", nil, &info)
		require.NoError(t, err)

		// Uptime should be a positive number
		assert.Greater(t, info.UptimeSeconds, 0.0)
		assert.NotEmpty(t, info.Uptime)
	})
}

func testUserManagement(t *testing.T, client *Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	testUsername := "testuser123"
	testPassword := "testpass456"

	// Clean up user if it exists from previous test
	_ = deleteTestUser(ctx, client, testUsername) // Ignore errors

	t.Run("CreateUser", func(t *testing.T) {
		req := &UserCreateRequest{
			Username:    testUsername,
			FullName:    fmt.Sprintf("Test User %s", testUsername),
			GroupCreate: Ptr(true),
			Password:    testPassword,
			Home:        "/var/empty",
			Shell:       "/usr/sbin/nologin",
			SMB:         Ptr(false),
		}
		_, err := client.User.Create(ctx, req)
		require.NoError(t, err)
	})

	t.Run("QueryUser", func(t *testing.T) {
		var users []User
		err := client.Call(ctx, "user.query", []any{
			[]any{
				[]any{"username", "=", testUsername},
			},
		}, &users)
		require.NoError(t, err)
		require.Len(t, users, 1, "Should find exactly one user")
		assert.Equal(t, testUsername, users[0].Username)
	})

	t.Run("UpdateUser", func(t *testing.T) {
		// First get the user ID
		var users []User
		err := client.Call(ctx, "user.query", []any{
			[]any{
				[]any{"username", "=", testUsername},
			},
		}, &users)
		require.NoError(t, err)
		require.Len(t, users, 1)

		userID := users[0].ID

		// Update the user's full name
		newFullName := "Updated Test User"
		var updateResult User
		err = client.Call(ctx, "user.update", []any{
			userID,
			map[string]any{
				"full_name": newFullName,
			},
		}, &updateResult)
		require.NoError(t, err)
		assert.Equal(t, newFullName, updateResult.FullName)

		// Verify the update
		var updatedUsers []User
		err = client.Call(ctx, "user.query", []any{
			[]any{
				[]any{"username", "=", testUsername},
			},
		}, &updatedUsers)
		require.NoError(t, err)
		require.Len(t, updatedUsers, 1)
		assert.Equal(t, newFullName, updatedUsers[0].FullName)
	})

	t.Run("DeleteUser", func(t *testing.T) {
		err := deleteTestUser(ctx, client, testUsername)
		require.NoError(t, err)

		// Verify user is deleted
		var users []User
		err = client.Call(ctx, "user.query", []any{
			[]any{
				[]any{"username", "=", testUsername},
			},
		}, &users)
		require.NoError(t, err)
		assert.Len(t, users, 0, "User should be deleted")
	})
}

func testAPIErrorHandling(t *testing.T, client *Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("NonExistentMethod", func(t *testing.T) {
		var result any
		err := client.Call(ctx, "nonexistent.method", nil, &result)
		assert.Error(t, err)

		var apiErr *RPCError
		assert.ErrorAs(t, err, &apiErr)
		// Note: TrueNAS returns error with code 0 and empty message for unknown methods
	})

	t.Run("InvalidParameters", func(t *testing.T) {
		var result any
		err := client.Call(ctx, "user.create", []any{
			"invalid-parameter-format",
		}, &result)
		assert.Error(t, err)
	})
}

func testConcurrentOperations(t *testing.T, client *Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const numOperations = 10
	results := make(chan error, numOperations)

	// Launch concurrent system.info calls
	for range numOperations {
		go func() {
			opCtx, opCancel := context.WithTimeout(ctx, 15*time.Second)
			defer opCancel()

			err := client.Call(opCtx, "system.info", nil, nil)
			results <- err
		}()
	}

	// Wait for all operations to complete
	for i := range numOperations {
		select {
		case err := <-results:
			assert.NoError(t, err, "Concurrent operation %d should succeed", i)
		case <-ctx.Done():
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}
}

func testLongRunningOperations(t *testing.T, client *Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	t.Run("SequentialOperations", func(t *testing.T) {
		// Perform a series of operations that might take some time
		operations := []string{
			"system.info",
			"system.version",
			"system.hostname",
			"user.query",
		}

		for _, method := range operations {
			opCtx, opCancel := context.WithTimeout(ctx, 30*time.Second)
			var result any
			err := client.Call(opCtx, method, nil, &result)
			opCancel()

			require.NoError(t, err, "Operation %s should succeed", method)
			assert.NotNil(t, result, "Operation %s should return a result", method)
		}
	})

	t.Run("OperationWithTimeout", func(t *testing.T) {
		// Test that operations respect context timeouts
		shortCtx, shortCancel := context.WithTimeout(ctx, 1*time.Millisecond)
		defer shortCancel()

		var result any
		err := client.Call(shortCtx, "system.info", nil, &result)
		// This should either succeed very quickly or timeout
		if err != nil {
			assert.ErrorIs(t, err, context.DeadlineExceeded)
		}
		_ = result
	})
}

func deleteTestUser(ctx context.Context, client *Client, username string) error {
	user, err := client.User.GetByUsername(ctx, username)
	if err != nil {
		return err
	}
	return client.User.Delete(ctx, user.ID, nil)
}

// createClientWithRetry attempts to connect and authenticate to TrueNAS with retries.
// After the HTTP endpoint is available, the WebSocket API may need additional time to
// become fully ready, so we retry connection attempts.
func createClientWithRetry(t *testing.T, connInfo testvm.ConnectionInfo) *Client {
	t.Helper()
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		client, err := NewClient(connInfo.WebSocketURL, Options{
			Username: connInfo.Username,
			Password: connInfo.Password,
			Debug:    true,
		})
		if err == nil {
			return client
		}
		t.Logf("Client connection attempt failed (will retry): %v", err)
		time.Sleep(10 * time.Second)
	}
	t.Fatal("could not connect to TrueNAS WebSocket API within 3 minutes")
	return nil
}

// waitForSystemReady waits for the TrueNAS system to become ready by polling system.info
func waitForSystemReady(t *testing.T, client *Client) error {
	timeout := time.Now().Add(2 * time.Minute)
	for time.Now().Before(timeout) {
		select {
		case <-t.Context().Done():
			return fmt.Errorf("timeout waiting for system to be ready")
		case <-time.After(5 * time.Second):
			err := client.Call(t.Context(), "system.info", nil, nil)
			if err == nil {
				t.Log("TrueNAS system is ready")
				return nil
			}
			t.Logf("System not ready yet: %v", err)
		}
	}
	return fmt.Errorf("system never became ready")
}

func testPoolCreation(t *testing.T, client *Client, helper *testvm.Manager) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// Add a disk to the VM
	diskID, err := helper.AddDisk("512M")
	require.NoError(t, err, "should add disk to VM")
	t.Logf("Added disk with ID: %s", diskID)

	// Wait for disk to be recognized by TrueNAS
	disks, err := waitForUnusedDisks(ctx, t, client, 1)
	require.NoError(t, err, "should wait for 1 unused disk")
	t.Logf("Found %d unused disks", len(disks))

	// Use the first available unused disk
	diskName := disks[0].Name
	t.Logf("Using disk: %s", diskName)

	// Create a pool using the disk
	poolName := "testpool"
	poolReq := PoolCreateRequest{
		Name:       poolName,
		Encryption: false,
		Topology: PoolTopologyCreate{
			Data: []VDevCreate{{
				Type:  VDevTypeStripe,
				Disks: []string{diskName},
			}},
		},
	}

	pool, err := client.Pool.Create(ctx, poolReq)
	require.NoError(t, err, "should create pool")
	require.NotNil(t, pool, "pool should not be nil")
	assert.Equal(t, poolName, pool.Name, "pool name should match")

	t.Logf("Created pool: %s (ID: %d)", pool.Name, pool.ID)

	// Verify the pool exists
	pools, err := client.Pool.List(ctx)
	require.NoError(t, err, "should list pools")

	var found bool
	for _, p := range pools {
		if p.Name == poolName {
			found = true
			break
		}
	}
	assert.True(t, found, "should find created pool in list")

	// Clean up: delete the pool
	err = client.Pool.Delete(ctx, pool.ID, false)
	require.NoError(t, err, "should delete pool")

	t.Logf("Successfully deleted pool: %s", poolName)
}

func testRAIDZ1PoolWithDatasetAndNFS(t *testing.T, client *Client, helper *testvm.Manager) {
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	// Add 3 disks to the VM for RAIDZ1
	for i := range 3 {
		diskID, err := helper.AddDisk("512M")
		require.NoError(t, err, "should add disk %d to VM", i+1)
		t.Logf("Added disk %d with ID: %s", i+1, diskID)
	}

	// Wait for disks to be recognized by TrueNAS
	disks, err := waitForUnusedDisks(ctx, t, client, 3)
	require.NoError(t, err, "should wait for 3 unused disks")
	t.Logf("Found %d unused disks", len(disks))

	// Use the first 3 available unused disks
	diskNames := make([]string, 3)
	for i := range 3 {
		diskNames[i] = disks[i].Name
	}
	t.Logf("Using disks for RAIDZ1: %v", diskNames)

	// Create a RAIDZ1 pool using the 3 disks
	poolName := "testpool3"
	poolReq := PoolCreateRequest{
		Name:       poolName,
		Encryption: false,
		Topology: PoolTopologyCreate{
			Data: []VDevCreate{{
				Type:  VDevTypeRaidz1,
				Disks: diskNames,
			}},
		},
	}

	pool, err := client.Pool.Create(ctx, poolReq)
	require.NoError(t, err, "should create RAIDZ1 pool")
	require.NotNil(t, pool, "pool should not be nil")
	assert.Equal(t, poolName, pool.Name, "pool name should match")
	t.Logf("Created RAIDZ1 pool: %s (ID: %d)", pool.Name, pool.ID)

	// Verify the pool exists
	pools, err := client.Pool.List(ctx)
	require.NoError(t, err, "should list pools")

	var foundPool bool
	for _, p := range pools {
		if p.Name == poolName {
			foundPool = true
			assert.Equal(t, PoolStatusOnline, p.Status, "pool should be online")
			break
		}
	}
	assert.True(t, foundPool, "should find created RAIDZ1 pool in list")

	// Create a dataset on the pool
	datasetName := poolName + "/testdata"
	datasetReq := &DatasetCreateRequest{
		Name: datasetName,
		Type: DatasetTypeFilesystem,
	}

	dataset, err := client.Dataset.Create(ctx, datasetReq)
	require.NoError(t, err, "should create dataset")
	require.NotNil(t, dataset, "dataset should not be nil")
	assert.Equal(t, datasetName, dataset.Name, "dataset name should match")
	t.Logf("Created dataset: %s", dataset.Name)

	// Verify the dataset exists
	datasets, err := client.Dataset.List(ctx)
	require.NoError(t, err, "should list datasets")

	var foundDataset bool
	for _, d := range datasets {
		if d.Name == datasetName {
			foundDataset = true
			assert.Equal(t, DatasetTypeFilesystem, d.Type, "dataset should be a filesystem")
			break
		}
	}
	assert.True(t, foundDataset, "should find created dataset in list")

	// Enable NFS service
	nfsService, err := client.Service.GetByName(ctx, "nfs")
	require.NoError(t, err, "should get NFS service")
	require.NotNil(t, nfsService, "NFS service should exist")

	if !nfsService.Enable {
		_, err = client.Service.Update(ctx, nfsService.ID, ServiceUpdateRequest{Enable: true})
		require.NoError(t, err, "should enable NFS service")
		t.Log("Enabled NFS service")
	}

	// Start NFS service if not running
	isRunning, err := client.Service.Started(ctx, "nfs")
	require.NoError(t, err, "should check if NFS service is running")
	if !isRunning {
		err = client.Service.Start(ctx, "nfs")
		require.NoError(t, err, "should start NFS service")
		t.Log("Started NFS service")
	}

	// Create an NFS share for the dataset
	nfsShareReq := &NFSShareRequest{
		Path:     "/mnt/" + datasetName,
		Comment:  "Test NFS share for integration test",
		Enabled:  true,
		RO:       false,
		Security: []string{"SYS"},
	}

	nfsShare, err := client.Sharing.NFS.Create(ctx, nfsShareReq)
	require.NoError(t, err, "should create NFS share")
	require.NotNil(t, nfsShare, "NFS share should not be nil")
	require.NotEmpty(t, nfsShare.Path, "NFS share should have a path")
	t.Logf("Created NFS share (ID: %d) for path: %s", nfsShare.ID, nfsShare.Path)

	// Verify the NFS share exists
	shares, err := client.Sharing.NFS.List(ctx)
	require.NoError(t, err, "should list NFS shares")

	var foundShare bool
	for _, share := range shares {
		if share.ID == nfsShare.ID {
			foundShare = true
			break
		}
	}
	assert.True(t, foundShare, "should find created NFS share in list")

	// Trigger a scrub on the pool using the new RunScrubAsync method
	jobID, err := client.Pool.RunScrubAsync(ctx, poolName, "START")
	require.NoError(t, err, "should start pool scrub")
	t.Logf("Started scrub on pool: %s (Job ID: %d)", poolName, jobID)

	// Monitor the scrub job until completion (or for a reasonable time)
	scrubCtx, scrubCancel := context.WithTimeout(ctx, 120*time.Second)
	defer scrubCancel()

	scrubJob, err := waitForJobCompletion(scrubCtx, t, client, jobID)
	require.NoError(t, err)
	t.Logf("Scrub job completed: %s", scrubJob.State)
	if scrubJob.Progress != nil {
		t.Logf("Final progress: %.2f%% - %s", scrubJob.Progress.Percent, scrubJob.Progress.Description)
	}

	// Clean up: delete NFS share
	err = client.Sharing.NFS.Delete(ctx, nfsShare.ID)
	require.NoError(t, err, "should delete NFS share")
	t.Logf("Deleted NFS share (ID: %d)", nfsShare.ID)

	// Clean up: delete dataset
	err = client.Dataset.Delete(ctx, dataset.ID, DatasetDeleteRequest{Recursive: Ptr(true), Force: Ptr(true)})
	require.NoError(t, err, "should delete dataset")
	t.Logf("Deleted dataset: %s", dataset.Name)

	// Clean up: delete pool
	err = client.Pool.Delete(ctx, pool.ID, false)
	require.NoError(t, err, "should delete RAIDZ1 pool")
	t.Logf("Successfully deleted RAIDZ1 pool: %s", poolName)
}

// waitForUnusedDisks polls for unused disks until the expected count is available
func waitForUnusedDisks(ctx context.Context, t *testing.T, client *Client, expectedCount int) ([]UnusedDisk, error) {
	timeout := time.Now().Add(2 * time.Minute)
	t.Logf("Waiting for %d unused disks to be recognized by TrueNAS...", expectedCount)

	for time.Now().Before(timeout) {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled while waiting for unused disks")
		case <-time.After(5 * time.Second):
			disks, err := client.Disk.GetUnused(ctx, false)
			if err != nil {
				t.Logf("Error querying unused disks: %v", err)
				continue
			}

			var usableDisks []UnusedDisk
			for _, d := range disks {
				if d.Driver != "floppy" {
					usableDisks = append(usableDisks, d)
				}
			}
			disks = usableDisks

			if len(disks) >= expectedCount {
				t.Logf("Found %d unused disks (needed %d)", len(disks), expectedCount)
				if len(disks) > expectedCount {
					disks = disks[len(disks)-expectedCount:]
				}
				// TrueNAS seems to error out if you try to use the disk too soon.
				time.Sleep(30 * time.Second)
				return disks, nil
			}

			t.Logf("Found %d unused disks, waiting for %d...", len(disks), expectedCount)
		}
	}

	return nil, fmt.Errorf("timeout waiting for %d unused disks", expectedCount)
}

// waitForJobCompletion polls until a job completes (success or failure)
func waitForJobCompletion(ctx context.Context, t *testing.T, client *Client, jobID int) (*Job, error) {
	timeout := time.Now().Add(2 * time.Minute)
	t.Logf("Waiting for job %d to complete...", jobID)

	for time.Now().Before(timeout) {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled while waiting for job completion")
		case <-time.After(5 * time.Second):
			job, err := client.Job.Get(ctx, jobID)
			if err != nil {
				t.Logf("Error querying job %d: %v", jobID, err)
				continue
			}
			if job == nil {
				return nil, fmt.Errorf("job %d not found", jobID)
			}

			t.Logf("Job %d state: %s", jobID, job.State)
			if job.Progress != nil {
				t.Logf("Job %d progress: %.2f%% - %s", jobID, job.Progress.Percent, job.Progress.Description)
			}

			if job.IsCompleted() {
				if job.IsSuccessful() {
					t.Logf("Job %d completed successfully", jobID)
					return job, nil
				}
				if job.IsFailed() {
					errorMsg := "unknown error"
					if job.Error != nil {
						errorMsg = *job.Error
					}
					return job, fmt.Errorf("job %d failed: %s", jobID, errorMsg)
				}
			}
		}
	}

	return nil, fmt.Errorf("timeout waiting for job %d to complete", jobID)
}
