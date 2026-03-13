package truenas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmartClient_GetConfig(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := &SmartConfig{
		Interval:      30,
		Powermode:     string(SmartPowerModeStandby),
		Difference:    8,
		Informational: 194,
		Critical:      0,
	}
	server.SetResponse("smart.config", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	config, err := client.Smart.GetConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, 30, config.Interval)
	assert.Equal(t, string(SmartPowerModeStandby), config.Powermode)
	assert.Equal(t, 8, config.Difference)
	assert.Equal(t, 194, config.Informational)
	assert.Equal(t, 0, config.Critical)
}

func TestSmartClient_GetConfig_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("smart.config", 500, "Unable to retrieve SMART configuration")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Smart.GetConfig(ctx)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "Unable to retrieve SMART configuration", apiErr.Message)
}

func TestSmartClient_UpdateConfig(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	inputConfig := &SmartConfig{
		Interval:      60,
		Powermode:     string(SmartPowerModeIdle),
		Difference:    10,
		Informational: 200,
		Critical:      1,
	}

	mockConfig := &SmartConfig{
		Interval:      60,
		Powermode:     string(SmartPowerModeIdle),
		Difference:    10,
		Informational: 200,
		Critical:      1,
	}
	server.SetResponse("smart.update", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	updated, err := client.Smart.UpdateConfig(ctx, inputConfig)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, 60, updated.Interval)
	assert.Equal(t, string(SmartPowerModeIdle), updated.Powermode)
	assert.Equal(t, 10, updated.Difference)
	assert.Equal(t, 200, updated.Informational)
	assert.Equal(t, 1, updated.Critical)
}

func TestSmartClient_UpdateConfig_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("smart.update", 400, "Invalid SMART configuration")

	client := server.CreateTestClient(t)
	defer client.Close()

	config := &SmartConfig{
		Interval:  -1, // Invalid value
		Powermode: "INVALID",
	}

	ctx := NewTestContext(t)
	_, err := client.Smart.UpdateConfig(ctx, config)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 400, apiErr.Data.Error)
	assert.Equal(t, "Invalid SMART configuration", apiErr.Message)
}

func TestSmartClient_ListTests(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockTests := []SmartTest{
		{
			ID: 1,
			Schedule: SmartTestSchedule{
				Minute: "0",
				Hour:   "0",
				DOM:    "*",
				Month:  "*",
				DOW:    "0",
			},
			Desc:     "Weekly long test",
			AllDisks: true,
			Disks:    []string{},
			Type:     string(SmartTestTypeLong),
		},
		{
			ID: 2,
			Schedule: SmartTestSchedule{
				Minute: "0",
				Hour:   "12",
				DOM:    "*",
				Month:  "*",
				DOW:    "*",
			},
			Desc:     "Daily short test on specific disks",
			AllDisks: false,
			Disks:    []string{"sda", "sdb"},
			Type:     string(SmartTestTypeShort),
		},
	}
	server.SetResponse("smart.test.query", mockTests)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	tests, err := client.Smart.ListTests(ctx)
	require.NoError(t, err)
	assert.Len(t, tests, 2)
	assert.Equal(t, 1, tests[0].ID)
	assert.Equal(t, "Weekly long test", tests[0].Desc)
	assert.True(t, tests[0].AllDisks)
	assert.Equal(t, string(SmartTestTypeLong), tests[0].Type)
	assert.Equal(t, 2, tests[1].ID)
	assert.False(t, tests[1].AllDisks)
	assert.Equal(t, []string{"sda", "sdb"}, tests[1].Disks)
}

func TestSmartClient_ListTests_Empty(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("smart.test.query", []SmartTest{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	tests, err := client.Smart.ListTests(ctx)
	require.NoError(t, err)
	assert.Empty(t, tests)
}

func TestSmartClient_GetTest(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockTest := SmartTest{
		ID: 1,
		Schedule: SmartTestSchedule{
			Minute: "0",
			Hour:   "2",
			DOM:    "*",
			Month:  "*",
			DOW:    "1",
		},
		Desc:     "Weekly conveyance test",
		AllDisks: false,
		Disks:    []string{"sda"},
		Type:     string(SmartTestTypeConveyance),
	}
	server.SetResponse("smart.test.query", []SmartTest{mockTest})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	test, err := client.Smart.GetTest(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, test)
	assert.Equal(t, 1, test.ID)
	assert.Equal(t, "Weekly conveyance test", test.Desc)
	assert.False(t, test.AllDisks)
	assert.Equal(t, []string{"sda"}, test.Disks)
	assert.Equal(t, string(SmartTestTypeConveyance), test.Type)
}

func TestSmartClient_GetTest_NotFound(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("smart.test.query", []SmartTest{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	test, err := client.Smart.GetTest(ctx, 999)
	assert.Error(t, err)
	assert.Nil(t, test)
	var notFoundErr *NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestSmartClient_GetTest_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("smart.test.query", 404, "Test not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Smart.GetTest(ctx, 1)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "Test not found", apiErr.Message)
}

func TestSmartClient_CreateTest(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	req := &SmartTestCreateRequest{
		Schedule: SmartTestSchedule{
			Minute: "30",
			Hour:   "3",
			DOM:    "*",
			Month:  "*",
			DOW:    "6",
		},
		Desc:     "Saturday offline test",
		AllDisks: true,
		Disks:    []string{},
		Type:     string(SmartTestTypeOffline),
	}

	mockTest := &SmartTest{
		ID: 3,
		Schedule: SmartTestSchedule{
			Minute: "30",
			Hour:   "3",
			DOM:    "*",
			Month:  "*",
			DOW:    "6",
		},
		Desc:     "Saturday offline test",
		AllDisks: true,
		Disks:    []string{},
		Type:     string(SmartTestTypeOffline),
	}
	server.SetResponse("smart.test.create", mockTest)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	test, err := client.Smart.CreateTest(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, test)
	assert.Equal(t, 3, test.ID)
	assert.Equal(t, "Saturday offline test", test.Desc)
	assert.True(t, test.AllDisks)
	assert.Equal(t, string(SmartTestTypeOffline), test.Type)
	assert.Equal(t, "30", test.Schedule.Minute)
	assert.Equal(t, "6", test.Schedule.DOW)
}

func TestSmartClient_CreateTest_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("smart.test.create", 400, "Invalid test configuration")

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &SmartTestCreateRequest{
		Schedule: SmartTestSchedule{
			Minute: "invalid",
			Hour:   "25", // Invalid hour
		},
		Type: "INVALID_TYPE",
	}

	ctx := NewTestContext(t)
	_, err := client.Smart.CreateTest(ctx, req)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 400, apiErr.Data.Error)
	assert.Equal(t, "Invalid test configuration", apiErr.Message)
}

func TestSmartClient_UpdateTest(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	req := &SmartTestCreateRequest{
		Schedule: SmartTestSchedule{
			Minute: "15",
			Hour:   "4",
			DOM:    "1",
			Month:  "*",
			DOW:    "*",
		},
		Desc:     "Monthly extended test",
		AllDisks: false,
		Disks:    []string{"sda", "sdb", "sdc"},
		Type:     string(SmartTestTypeLong),
	}

	mockTest := &SmartTest{
		ID: 1,
		Schedule: SmartTestSchedule{
			Minute: "15",
			Hour:   "4",
			DOM:    "1",
			Month:  "*",
			DOW:    "*",
		},
		Desc:     "Monthly extended test",
		AllDisks: false,
		Disks:    []string{"sda", "sdb", "sdc"},
		Type:     string(SmartTestTypeLong),
	}
	server.SetResponse("smart.test.update", mockTest)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	test, err := client.Smart.UpdateTest(ctx, 1, req)
	require.NoError(t, err)
	require.NotNil(t, test)
	assert.Equal(t, 1, test.ID)
	assert.Equal(t, "Monthly extended test", test.Desc)
	assert.False(t, test.AllDisks)
	assert.Equal(t, []string{"sda", "sdb", "sdc"}, test.Disks)
	assert.Equal(t, "15", test.Schedule.Minute)
	assert.Equal(t, "1", test.Schedule.DOM)
}

func TestSmartClient_UpdateTest_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("smart.test.update", 404, "Test not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &SmartTestCreateRequest{
		Type: string(SmartTestTypeShort),
	}

	ctx := NewTestContext(t)
	_, err := client.Smart.UpdateTest(ctx, 999, req)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "Test not found", apiErr.Message)
}

func TestSmartClient_DeleteTest(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("smart.test.delete", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Smart.DeleteTest(ctx, 1)
	assert.NoError(t, err)
}

func TestSmartClient_DeleteTest_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("smart.test.delete", 404, "Test not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Smart.DeleteTest(ctx, 999)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "Test not found", apiErr.Message)
}

func TestSmartClient_GetDiskChoices(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockChoices := map[string]any{
		"sda": "sda (SAMSUNG SSD 980 PRO 1TB)",
		"sdb": "sdb (WDC WD40EFRX-68N32N0 4TB)",
		"sdc": "sdc (ST8000DM004-2CX188 8TB)",
	}
	server.SetResponse("smart.test.disk_choices", mockChoices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	choices, err := client.Smart.GetDiskChoices(ctx, false)
	require.NoError(t, err)
	require.NotNil(t, choices)

	choicesMap, ok := choices.(map[string]any)
	require.True(t, ok)
	assert.Contains(t, choicesMap, "sda")
	assert.Contains(t, choicesMap, "sdb")
	assert.Contains(t, choicesMap, "sdc")
}

func TestSmartClient_GetDiskChoices_FullDisk(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockChoices := []string{"/dev/sda", "/dev/sdb", "/dev/sdc"}
	server.SetResponse("smart.test.disk_choices", mockChoices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	choices, err := client.Smart.GetDiskChoices(ctx, true)
	require.NoError(t, err)
	require.NotNil(t, choices)

	choicesList, ok := choices.([]any)
	require.True(t, ok)
	assert.Len(t, choicesList, 3)
}

func TestSmartClient_GetDiskChoices_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("smart.test.disk_choices", 500, "Unable to retrieve disk choices")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Smart.GetDiskChoices(ctx, false)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "Unable to retrieve disk choices", apiErr.Message)
}

func TestSmartClient_RunManualTest(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("smart.test.manual_test", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	tests := []SmartManualTestRequest{
		{
			Disk: "sda",
			Type: string(SmartTestTypeShort),
		},
		{
			Disk: "sdb",
			Type: string(SmartTestTypeLong),
		},
	}

	ctx := NewTestContext(t)
	err := client.Smart.RunManualTest(ctx, tests)
	assert.NoError(t, err)
}

func TestSmartClient_RunManualTest_SingleDisk(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("smart.test.manual_test", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	tests := []SmartManualTestRequest{
		{
			Disk: "sda",
			Type: string(SmartTestTypeConveyance),
		},
	}

	ctx := NewTestContext(t)
	err := client.Smart.RunManualTest(ctx, tests)
	assert.NoError(t, err)
}

func TestSmartClient_RunManualTest_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("smart.test.manual_test", 400, "Invalid test parameters")

	client := server.CreateTestClient(t)
	defer client.Close()

	tests := []SmartManualTestRequest{
		{
			Disk: "invalid_disk",
			Type: "INVALID_TYPE",
		},
	}

	ctx := NewTestContext(t)
	err := client.Smart.RunManualTest(ctx, tests)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 400, apiErr.Data.Error)
	assert.Equal(t, "Invalid test parameters", apiErr.Message)
}

func TestSmartClient_GetAllTestResults(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockResults := []SmartTestResult{
		{
			Disk: "sda",
			Tests: []SmartTest{
				{
					ID:   1,
					Desc: "Short self-test",
					Type: string(SmartTestTypeShort),
				},
			},
		},
		{
			Disk: "sdb",
			Tests: []SmartTest{
				{
					ID:   2,
					Desc: "Extended self-test",
					Type: string(SmartTestTypeLong),
				},
			},
		},
	}
	server.SetResponse("smart.test.results", mockResults)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	results, err := client.Smart.GetAllTestResults(ctx)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "sda", results[0].Disk)
	assert.Equal(t, "sdb", results[1].Disk)
	assert.Len(t, results[0].Tests, 1)
	assert.Equal(t, "Short self-test", results[0].Tests[0].Desc)
}

func TestSmartClient_GetAllTestResults_Empty(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("smart.test.results", []SmartTestResult{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	results, err := client.Smart.GetAllTestResults(ctx)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSmartClient_GetAllTestResults_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("smart.test.results", 500, "Unable to retrieve test results")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Smart.GetAllTestResults(ctx)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "Unable to retrieve test results", apiErr.Message)
}

func TestSmartClient_GetDiskTestResults(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockResult := &SmartTestResult{
		Disk: "sda",
		Tests: []SmartTest{
			{
				ID:   1,
				Desc: "Short self-test",
				Type: string(SmartTestTypeShort),
			},
			{
				ID:   2,
				Desc: "Extended self-test",
				Type: string(SmartTestTypeLong),
			},
		},
	}
	server.SetResponse("smart.test.results", mockResult)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	result, err := client.Smart.GetDiskTestResults(ctx, "sda")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "sda", result.Disk)
	assert.Len(t, result.Tests, 2)
	assert.Equal(t, "Short self-test", result.Tests[0].Desc)
	assert.Equal(t, "Extended self-test", result.Tests[1].Desc)
}

func TestSmartClient_GetDiskTestResults_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("smart.test.results", 404, "Disk not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Smart.GetDiskTestResults(ctx, "nonexistent")
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "Disk not found", apiErr.Message)
}

func TestSmartClient_GetDiskAttributes(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockAttributes := []SmartAttributes{
		{
			ID:         1,
			Name:       "Raw_Read_Error_Rate",
			Value:      200,
			Worst:      200,
			Threshold:  51,
			Type:       "Pre-fail",
			Updated:    "Always",
			WhenFailed: "-",
			RawValue:   0,
		},
		{
			ID:         5,
			Name:       "Reallocated_Sector_Ct",
			Value:      100,
			Worst:      100,
			Threshold:  36,
			Type:       "Pre-fail",
			Updated:    "Always",
			WhenFailed: "-",
			RawValue:   0,
		},
		{
			ID:         9,
			Name:       "Power_On_Hours",
			Value:      99,
			Worst:      99,
			Threshold:  0,
			Type:       "Old_age",
			Updated:    "Always",
			WhenFailed: "-",
			RawValue:   1234.0,
		},
	}
	server.SetResponse("disk.smart_attributes", mockAttributes)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	attributes, err := client.Smart.GetDiskAttributes(ctx, "sda")
	require.NoError(t, err)
	assert.Len(t, attributes, 3)
	assert.Equal(t, 1, attributes[0].ID)
	assert.Equal(t, "Raw_Read_Error_Rate", attributes[0].Name)
	assert.Equal(t, 200, attributes[0].Value)
	assert.Equal(t, 51, attributes[0].Threshold)
	assert.Equal(t, "Pre-fail", attributes[0].Type)
	assert.Equal(t, 5, attributes[1].ID)
	assert.Equal(t, 9, attributes[2].ID)
	assert.Equal(t, 1234.0, attributes[2].RawValue)
}

func TestSmartClient_GetDiskAttributes_Empty(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("disk.smart_attributes", []SmartAttributes{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	attributes, err := client.Smart.GetDiskAttributes(ctx, "sda")
	require.NoError(t, err)
	assert.Empty(t, attributes)
}

func TestSmartClient_GetDiskAttributes_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("disk.smart_attributes", 404, "Disk not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Smart.GetDiskAttributes(ctx, "nonexistent")
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "Disk not found", apiErr.Message)
}

// Test constants and types
func TestSmartTestTypes(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "LONG", string(SmartTestTypeLong))
	assert.Equal(t, "SHORT", string(SmartTestTypeShort))
	assert.Equal(t, "CONVEYANCE", string(SmartTestTypeConveyance))
	assert.Equal(t, "OFFLINE", string(SmartTestTypeOffline))
}

func TestSmartPowerModes(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "NEVER", string(SmartPowerModeNever))
	assert.Equal(t, "SLEEP", string(SmartPowerModeSleep))
	assert.Equal(t, "STANDBY", string(SmartPowerModeStandby))
	assert.Equal(t, "IDLE", string(SmartPowerModeIdle))
}

// Test edge cases and complex scenarios
func TestSmartClient_CreateTest_AllDisksVsSpecificDisks(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	// Test case 1: AllDisks = true, Disks should be empty
	req1 := &SmartTestCreateRequest{
		Schedule: SmartTestSchedule{
			Minute: "0",
			Hour:   "2",
			DOM:    "*",
			Month:  "*",
			DOW:    "*",
		},
		Desc:     "All disks test",
		AllDisks: true,
		Disks:    []string{}, // Should be empty when AllDisks is true
		Type:     string(SmartTestTypeShort),
	}

	mockTest1 := &SmartTest{
		ID:       1,
		Schedule: req1.Schedule,
		Desc:     req1.Desc,
		AllDisks: true,
		Disks:    []string{},
		Type:     req1.Type,
	}
	server.SetResponse("smart.test.create", mockTest1)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	test1, err := client.Smart.CreateTest(ctx, req1)
	require.NoError(t, err)
	assert.True(t, test1.AllDisks)
	assert.Empty(t, test1.Disks)

	// Test case 2: AllDisks = false, specific disks should be specified
	req2 := &SmartTestCreateRequest{
		Schedule: SmartTestSchedule{
			Minute: "0",
			Hour:   "4",
			DOM:    "*",
			Month:  "*",
			DOW:    "*",
		},
		Desc:     "Specific disks test",
		AllDisks: false,
		Disks:    []string{"sda", "sdb"},
		Type:     string(SmartTestTypeLong),
	}

	mockTest2 := &SmartTest{
		ID:       2,
		Schedule: req2.Schedule,
		Desc:     req2.Desc,
		AllDisks: false,
		Disks:    []string{"sda", "sdb"},
		Type:     req2.Type,
	}
	server.SetResponse("smart.test.create", mockTest2)

	test2, err := client.Smart.CreateTest(ctx, req2)
	require.NoError(t, err)
	assert.False(t, test2.AllDisks)
	assert.Equal(t, []string{"sda", "sdb"}, test2.Disks)
}

func TestSmartClient_RunManualTest_EmptySlice(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("smart.test.manual_test", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Smart.RunManualTest(ctx, []SmartManualTestRequest{})
	assert.NoError(t, err)
}

func TestSmartClient_ComplexSchedule(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	// Test complex cron schedule
	req := &SmartTestCreateRequest{
		Schedule: SmartTestSchedule{
			Minute: "0,30",         // Every 30 minutes
			Hour:   "9-17",         // Business hours
			DOM:    "1,15",         // 1st and 15th of month
			Month:  "1,3,5,7,9,11", // Odd months
			DOW:    "1-5",          // Weekdays
		},
		Desc:     "Complex schedule test",
		AllDisks: true,
		Type:     string(SmartTestTypeShort),
	}

	mockTest := &SmartTest{
		ID:       1,
		Schedule: req.Schedule,
		Desc:     req.Desc,
		AllDisks: req.AllDisks,
		Disks:    []string{},
		Type:     req.Type,
	}
	server.SetResponse("smart.test.create", mockTest)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	test, err := client.Smart.CreateTest(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "0,30", test.Schedule.Minute)
	assert.Equal(t, "9-17", test.Schedule.Hour)
	assert.Equal(t, "1,15", test.Schedule.DOM)
	assert.Equal(t, "1,3,5,7,9,11", test.Schedule.Month)
	assert.Equal(t, "1-5", test.Schedule.DOW)
}
