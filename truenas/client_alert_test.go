package truenas

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlertClient_List(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockAlerts := []Alert{
		{
			UUID:      "alert-1",
			Source:    "system",
			Klass:     "SystemAlert",
			Level:     "WARNING",
			Formatted: "System warning message",
			Text:      "Warning text",
			OneShot:   false,
			Mail:      true,
			DateTime:  time.Now(),
		},
		{
			UUID:      "alert-2",
			Source:    "disk",
			Klass:     "DiskAlert",
			Level:     "CRITICAL",
			Formatted: "Disk failure detected",
			Text:      "Critical disk error",
			OneShot:   true,
			Mail:      true,
			DateTime:  time.Now(),
		},
	}
	server.SetResponse("alert.list", mockAlerts)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	alerts, err := client.Alert.List(ctx)
	require.NoError(t, err)
	assert.Len(t, alerts, 2)
	assert.Equal(t, "alert-1", alerts[0].UUID)
	assert.Equal(t, "WARNING", alerts[0].Level)
	assert.Equal(t, "CRITICAL", alerts[1].Level)
}

func TestAlertClient_Dismiss(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("alert.dismiss", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Alert.Dismiss(ctx, "alert-1")
	assert.NoError(t, err)
}

func TestAlertClient_Restore(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("alert.restore", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Alert.Restore(ctx, "alert-1")
	assert.NoError(t, err)
}

func TestAlertClient_ListCategories(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockCategories := []AlertCategory{
		{ID: "System", Title: "System Alerts"},
		{ID: "Hardware", Title: "Hardware Alerts"},
		{ID: "Services", Title: "Service Alerts"},
	}
	server.SetResponse("alert.list_categories", mockCategories)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	categories, err := client.Alert.ListCategories(ctx)
	require.NoError(t, err)
	assert.Len(t, categories, 3)
	assert.Equal(t, "System", categories[0].ID)
	assert.Equal(t, "System Alerts", categories[0].Title)
}

func TestAlertClient_ListPolicies(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockPolicies := []AlertPolicy{
		{Name: "IMMEDIATELY", Description: "Immediately"},
		{Name: "HOURLY", Description: "Hourly"},
		{Name: "DAILY", Description: "Daily"},
		{Name: "NEVER", Description: "Never"},
	}
	server.SetResponse("alert.list_policies", mockPolicies)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	policies, err := client.Alert.ListPolicies(ctx)
	require.NoError(t, err)
	assert.Len(t, policies, 4)
	assert.Equal(t, "IMMEDIATELY", policies[0].Name)
	assert.Equal(t, "Immediately", policies[0].Description)
}

func TestAlertClient_GetAlertClassesConfig(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := map[string]any{
		"UPSBatteryLow": map[string]any{
			"policy": "IMMEDIATELY",
			"send":   []string{"EMAIL"},
		},
		"VolumeStatusAlert": map[string]any{
			"policy": "IMMEDIATELY",
			"send":   []string{"EMAIL", "SNMP"},
		},
	}
	server.SetResponse("alertclasses.config", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	config, err := client.Alert.GetAlertClassesConfig(ctx)
	require.NoError(t, err)
	assert.Contains(t, config, "UPSBatteryLow")
	assert.Contains(t, config, "VolumeStatusAlert")
}

func TestAlertClient_UpdateAlertClasses(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockResult := map[string]any{
		"UPSBatteryLow": map[string]any{
			"policy": "HOURLY",
			"send":   []string{"EMAIL"},
		},
	}
	server.SetResponse("alertclasses.update", mockResult)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &AlertClassesUpdateRequest{
		Classes: map[string]any{
			"UPSBatteryLow": map[string]any{
				"policy": "HOURLY",
				"send":   []string{"EMAIL"},
			},
		},
	}

	ctx := NewTestContext(t)
	result, err := client.Alert.UpdateAlertClasses(ctx, req)
	require.NoError(t, err)
	assert.Contains(t, result, "UPSBatteryLow")
}

// AlertServiceClient Tests
func TestAlertServiceClient_List(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockServices := []AlertService{
		{
			ID:      1,
			Name:    "email-alerts",
			Type:    "Mail",
			Level:   "WARNING",
			Enabled: true,
		},
		{
			ID:      2,
			Name:    "slack-alerts",
			Type:    "Slack",
			Level:   "CRITICAL",
			Enabled: false,
		},
	}
	server.SetResponse("alertservice.query", mockServices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	services, err := client.AlertService.List(ctx)
	require.NoError(t, err)
	assert.Len(t, services, 2)
	assert.Equal(t, "email-alerts", services[0].Name)
	assert.Equal(t, "Mail", services[0].Type)
	assert.True(t, services[0].Enabled)
}

func TestAlertServiceClient_Get(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockService := AlertService{
		ID:      1,
		Name:    "email-alerts",
		Type:    "Mail",
		Level:   "WARNING",
		Enabled: true,
	}
	server.SetResponse("alertservice.query", []AlertService{mockService})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	service, err := client.AlertService.Get(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, service)
	assert.Equal(t, "email-alerts", service.Name)
	assert.Equal(t, 1, service.ID)
}

func TestAlertServiceClient_Get_NotFound(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("alertservice.query", []AlertService{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	service, err := client.AlertService.Get(ctx, 999)
	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "not found")
}

func TestAlertServiceClient_Create(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockService := AlertService{
		ID:      1,
		Name:    "new-email-service",
		Type:    "Mail",
		Level:   "INFO",
		Enabled: true,
	}
	server.SetResponse("alertservice.create", mockService)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &AlertServiceCreateRequest{
		Name:    "new-email-service",
		Type:    "Mail",
		Level:   "INFO",
		Enabled: true,
		Attributes: map[string]any{
			"email":    "admin@example.com",
			"subject":  "TrueNAS Alert",
			"fromaddr": "truenas@example.com",
		},
	}

	ctx := NewTestContext(t)
	service, err := client.AlertService.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, service)
	assert.Equal(t, "new-email-service", service.Name)
	assert.Equal(t, "Mail", service.Type)
}

func TestAlertServiceClient_Update(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockService := AlertService{
		ID:      1,
		Name:    "updated-email-service",
		Type:    "Mail",
		Level:   "CRITICAL",
		Enabled: false,
	}
	server.SetResponse("alertservice.update", mockService)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &AlertServiceUpdateRequest{
		Name:    "updated-email-service",
		Level:   "CRITICAL",
		Enabled: false,
		Attributes: map[string]any{
			"email": "newemail@example.com",
		},
	}

	ctx := NewTestContext(t)
	service, err := client.AlertService.Update(ctx, 1, req)
	require.NoError(t, err)
	require.NotNil(t, service)
	assert.Equal(t, "updated-email-service", service.Name)
	assert.Equal(t, "CRITICAL", service.Level)
	assert.False(t, service.Enabled)
}

func TestAlertServiceClient_Delete(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("alertservice.delete", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.AlertService.Delete(ctx, 1)
	assert.NoError(t, err)
}

func TestAlertServiceClient_Test(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("alertservice.test", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &AlertServiceCreateRequest{
		Name:    "test-service",
		Type:    "Mail",
		Level:   "INFO",
		Enabled: true,
		Attributes: map[string]any{
			"email":    "test@example.com",
			"subject":  "Test Alert",
			"fromaddr": "truenas@example.com",
		},
	}

	ctx := NewTestContext(t)
	err := client.AlertService.Test(ctx, req)
	assert.NoError(t, err)
}

func TestAlertServiceClient_ListTypes(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockTypes := map[string]any{
		"Mail": map[string]any{
			"title": "E-Mail",
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"email": map[string]any{
						"type":  "string",
						"title": "E-mail",
					},
				},
			},
		},
		"Slack": map[string]any{
			"title": "Slack",
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":  "string",
						"title": "Webhook URL",
					},
				},
			},
		},
	}
	server.SetResponse("alertservice.list_types", mockTypes)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	types, err := client.AlertService.ListTypes(ctx)
	require.NoError(t, err)
	assert.Contains(t, types, "Mail")
	assert.Contains(t, types, "Slack")
}

func TestAlertClient_ErrorHandling(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("alert.list", 500, "Alert service unavailable")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Alert.List(ctx)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "Alert service unavailable", apiErr.Message)
}
