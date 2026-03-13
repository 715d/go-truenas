package truenas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkClient_ListInterfaces(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInterfaces := []NetworkInterface{
		{
			ID:          1,
			Name:        "eth0",
			Description: "Primary ethernet interface",
			Type:        InterfaceTypePhysical,
			IPV4DHCP:    true,
			IPV6Auto:    false,
			MTU:         1500,
			Disable:     false,
			HasDefault:  true,
			Aliases: []NetworkInterfaceAlias{
				{
					Type:    "INET",
					Address: "192.168.1.100",
					Netmask: 24,
				},
			},
			State: NetworkInterfaceState{
				Name:        "eth0",
				OrigName:    "eth0",
				Description: "Intel Ethernet",
				MTU:         1500,
				LinkState:   "LINK_STATE_UP",
				LinkAddress: "aa:bb:cc:dd:ee:ff",
			},
		},
		{
			ID:            2,
			Name:          "br0",
			Description:   "Bridge interface",
			Type:          InterfaceTypeBridge,
			IPV4DHCP:      false,
			IPV6Auto:      false,
			MTU:           1500,
			Disable:       false,
			HasDefault:    false,
			BridgeMembers: []string{"eth1", "eth2"},
		},
	}
	server.SetResponse("interface.query", mockInterfaces)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	interfaces, err := client.Network.ListInterfaces(ctx)
	require.NoError(t, err)
	assert.Len(t, interfaces, 2)
	assert.Equal(t, "eth0", interfaces[0].Name)
	assert.Equal(t, InterfaceTypePhysical, interfaces[0].Type)
	assert.True(t, interfaces[0].IPV4DHCP)
	assert.Equal(t, "br0", interfaces[1].Name)
	assert.Equal(t, InterfaceTypeBridge, interfaces[1].Type)
	assert.Contains(t, interfaces[1].BridgeMembers, "eth1")
}

func TestNetworkClient_GetInterface(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInterface := NetworkInterface{
		ID:          1,
		Name:        "eth0",
		Description: "Primary ethernet interface",
		Type:        InterfaceTypePhysical,
		IPV4DHCP:    true,
		IPV6Auto:    false,
		MTU:         1500,
		Disable:     false,
		HasDefault:  true,
		Aliases: []NetworkInterfaceAlias{
			{
				Type:    "INET",
				Address: "192.168.1.100",
				Netmask: 24,
			},
		},
	}
	server.SetResponse("interface.query", []NetworkInterface{mockInterface})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	iface, err := client.Network.GetInterface(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, iface)
	assert.Equal(t, 1, iface.ID)
	assert.Equal(t, "eth0", iface.Name)
	assert.Equal(t, InterfaceTypePhysical, iface.Type)
	assert.True(t, iface.IPV4DHCP)
}

func TestNetworkClient_GetInterface_NotFound(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("interface.query", []NetworkInterface{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	iface, err := client.Network.GetInterface(ctx, 999)
	assert.Error(t, err)
	assert.Nil(t, iface)
	var notFoundErr *NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestNetworkClient_GetInterfaceByName(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInterface := NetworkInterface{
		ID:          1,
		Name:        "eth0",
		Description: "Primary ethernet interface",
		Type:        InterfaceTypePhysical,
		IPV4DHCP:    true,
		IPV6Auto:    false,
		MTU:         1500,
	}
	server.SetResponse("interface.query", []NetworkInterface{mockInterface})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	iface, err := client.Network.GetInterfaceByName(ctx, "eth0")
	require.NoError(t, err)
	require.NotNil(t, iface)
	assert.Equal(t, "eth0", iface.Name)
	assert.Equal(t, InterfaceTypePhysical, iface.Type)
	assert.True(t, iface.IPV4DHCP)
}

func TestNetworkClient_GetInterfaceByName_NotFound(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("interface.query", []NetworkInterface{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	iface, err := client.Network.GetInterfaceByName(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, iface)
	var notFoundErr *NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestNetworkClient_CreateInterface(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInterface := NetworkInterface{
		ID:          3,
		Name:        "vlan100",
		Description: "VLAN 100",
		Type:        InterfaceTypeVLAN,
		IPV4DHCP:    false,
		IPV6Auto:    false,
		MTU:         1500,
		VlanParent:  "eth0",
		VlanTag:     100,
		VlanPcp:     0,
	}
	server.SetResponse("interface.create", mockInterface)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &NetworkInterfaceCreateRequest{
		Name:        "vlan100",
		Description: "VLAN 100",
		Type:        InterfaceTypeVLAN,
		IPV4DHCP:    false,
		IPV6Auto:    false,
		MTU:         1500,
		VlanParent:  Ptr("eth0"),
		VlanTag:     Ptr(100),
		VlanPcp:     Ptr(0),
	}

	ctx := NewTestContext(t)
	iface, err := client.Network.CreateInterface(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, iface)
	assert.Equal(t, 3, iface.ID)
	assert.Equal(t, "vlan100", iface.Name)
	assert.Equal(t, InterfaceTypeVLAN, iface.Type)
	assert.Equal(t, "eth0", iface.VlanParent)
	assert.Equal(t, 100, iface.VlanTag)
}

func TestNetworkClient_CreateInterface_Bridge(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInterface := NetworkInterface{
		ID:            4,
		Name:          "br0",
		Description:   "Bridge interface",
		Type:          InterfaceTypeBridge,
		IPV4DHCP:      false,
		IPV6Auto:      false,
		MTU:           1500,
		BridgeMembers: []string{"eth1", "eth2"},
	}
	server.SetResponse("interface.create", mockInterface)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &NetworkInterfaceCreateRequest{
		Name:          "br0",
		Description:   "Bridge interface",
		Type:          InterfaceTypeBridge,
		IPV4DHCP:      false,
		IPV6Auto:      false,
		MTU:           1500,
		BridgeMembers: []string{"eth1", "eth2"},
	}

	ctx := NewTestContext(t)
	iface, err := client.Network.CreateInterface(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, iface)
	assert.Equal(t, "br0", iface.Name)
	assert.Equal(t, InterfaceTypeBridge, iface.Type)
	assert.Contains(t, iface.BridgeMembers, "eth1")
	assert.Contains(t, iface.BridgeMembers, "eth2")
}

func TestNetworkClient_CreateInterface_LinkAggregation(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInterface := NetworkInterface{
		ID:          5,
		Name:        "lagg0",
		Description: "Link aggregation",
		Type:        InterfaceTypeLinkAgg,
		IPV4DHCP:    false,
		IPV6Auto:    false,
		MTU:         1500,
		LagProtocol: LAGProtocolLACP,
		LagPorts:    []string{"eth1", "eth2"},
	}
	server.SetResponse("interface.create", mockInterface)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &NetworkInterfaceCreateRequest{
		Name:        "lagg0",
		Description: "Link aggregation",
		Type:        InterfaceTypeLinkAgg,
		IPV4DHCP:    false,
		IPV6Auto:    false,
		MTU:         1500,
		LagProtocol: Ptr(LAGProtocolLACP),
		LagPorts:    []string{"eth1", "eth2"},
	}

	ctx := NewTestContext(t)
	iface, err := client.Network.CreateInterface(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, iface)
	assert.Equal(t, "lagg0", iface.Name)
	assert.Equal(t, InterfaceTypeLinkAgg, iface.Type)
	assert.Equal(t, LAGProtocolLACP, iface.LagProtocol)
	assert.Contains(t, iface.LagPorts, "eth1")
}

func TestNetworkClient_UpdateInterface(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInterface := NetworkInterface{
		ID:          1,
		Name:        "eth0",
		Description: "Updated description",
		Type:        InterfaceTypePhysical,
		IPV4DHCP:    false,
		IPV6Auto:    true,
		MTU:         9000,
		Aliases: []NetworkInterfaceAlias{
			{
				Type:    "INET",
				Address: "192.168.1.200",
				Netmask: 24,
			},
		},
	}
	server.SetResponse("interface.update", mockInterface)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &NetworkInterfaceUpdateRequest{
		Description: Ptr("Updated description"),
		IPV4DHCP:    Ptr(false),
		IPV6Auto:    Ptr(true),
		MTU:         Ptr(9000),
		Aliases: []NetworkInterfaceAlias{
			{
				Type:    "INET",
				Address: "192.168.1.200",
				Netmask: 24,
			},
		},
	}

	ctx := NewTestContext(t)
	iface, err := client.Network.UpdateInterface(ctx, 1, req)
	require.NoError(t, err)
	require.NotNil(t, iface)
	assert.Equal(t, "Updated description", iface.Description)
	assert.False(t, iface.IPV4DHCP)
	assert.True(t, iface.IPV6Auto)
	assert.Equal(t, 9000, iface.MTU)
	assert.Equal(t, "192.168.1.200", iface.Aliases[0].Address)
}

func TestNetworkClient_DeleteInterface(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("interface.delete", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Network.DeleteInterface(ctx, 1)
	assert.NoError(t, err)
}

func TestNetworkClient_GetConfiguration(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := NetworkConfiguration{
		ID:              1,
		Hostname:        "truenas",
		HostnameB:       "truenas-b",
		HostnameVirtual: "truenas-virtual",
		Domain:          "example.com",
		Domains:         []string{"example.com", "test.com"},
		IPv4Gateway:     "192.168.1.1",
		IPv6Gateway:     "::1",
		Nameserver1:     "8.8.8.8",
		Nameserver2:     "8.8.4.4",
		Nameserver3:     "1.1.1.1",
		HTTPProxy:       "http://proxy.example.com:8080",
		NetwaitEnabled:  true,
		NetwaitIP:       []string{"192.168.1.1", "8.8.8.8"},
		ServiceAnnouncement: &ServiceAnnouncement{
			Netbios: true,
			MDNS:    true,
			WSD:     false,
		},
		ActivityType:     "PING",
		ActivityInterval: 60,
	}
	server.SetResponse("network.configuration.config", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	config, err := client.Network.GetConfiguration(ctx)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "truenas", config.Hostname)
	assert.Equal(t, "example.com", config.Domain)
	assert.Equal(t, "192.168.1.1", config.IPv4Gateway)
	assert.Equal(t, "8.8.8.8", config.Nameserver1)
	assert.True(t, config.NetwaitEnabled)
	assert.True(t, config.ServiceAnnouncement.Netbios)
	assert.True(t, config.ServiceAnnouncement.MDNS)
	assert.False(t, config.ServiceAnnouncement.WSD)
}

func TestNetworkClient_UpdateConfiguration(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := NetworkConfiguration{
		ID:          1,
		Hostname:    "new-truenas",
		Domain:      "newdomain.com",
		IPv4Gateway: "192.168.2.1",
		Nameserver1: "1.1.1.1",
		Nameserver2: "1.0.0.1",
	}
	server.SetResponse("network.configuration.update", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	updateConfig := &NetworkConfiguration{
		Hostname:    "new-truenas",
		Domain:      "newdomain.com",
		IPv4Gateway: "192.168.2.1",
		Nameserver1: "1.1.1.1",
		Nameserver2: "1.0.0.1",
	}

	ctx := NewTestContext(t)
	config, err := client.Network.UpdateConfiguration(ctx, updateConfig)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "new-truenas", config.Hostname)
	assert.Equal(t, "newdomain.com", config.Domain)
	assert.Equal(t, "192.168.2.1", config.IPv4Gateway)
	assert.Equal(t, "1.1.1.1", config.Nameserver1)
}

func TestNetworkClient_ListStaticRoutes(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockRoutes := []StaticRoute{
		{
			ID:          1,
			Destination: "10.0.0.0/8",
			Gateway:     "192.168.1.1",
			Description: "Private network route",
		},
		{
			ID:          2,
			Destination: "172.16.0.0/12",
			Gateway:     "192.168.1.2",
			Description: "Another private network",
		},
	}
	server.SetResponse("staticroute.query", mockRoutes)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	routes, err := client.Network.ListStaticRoutes(ctx)
	require.NoError(t, err)
	assert.Len(t, routes, 2)
	assert.Equal(t, "10.0.0.0/8", routes[0].Destination)
	assert.Equal(t, "192.168.1.1", routes[0].Gateway)
	assert.Equal(t, "Private network route", routes[0].Description)
}

func TestNetworkClient_GetStaticRoute(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockRoute := StaticRoute{
		ID:          1,
		Destination: "10.0.0.0/8",
		Gateway:     "192.168.1.1",
		Description: "Private network route",
	}
	server.SetResponse("staticroute.query", []StaticRoute{mockRoute})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	route, err := client.Network.GetStaticRoute(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, route)
	assert.Equal(t, 1, route.ID)
	assert.Equal(t, "10.0.0.0/8", route.Destination)
	assert.Equal(t, "192.168.1.1", route.Gateway)
	assert.Equal(t, "Private network route", route.Description)
}

func TestNetworkClient_GetStaticRoute_NotFound(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("staticroute.query", []StaticRoute{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	route, err := client.Network.GetStaticRoute(ctx, 999)
	assert.Error(t, err)
	assert.Nil(t, route)
	var notFoundErr *NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestNetworkClient_CreateStaticRoute(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockRoute := StaticRoute{
		ID:          3,
		Destination: "192.168.100.0/24",
		Gateway:     "192.168.1.10",
		Description: "Test network route",
	}
	server.SetResponse("staticroute.create", mockRoute)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := StaticRouteCreateRequest{
		Destination: "192.168.100.0/24",
		Gateway:     "192.168.1.10",
		Description: "Test network route",
	}

	ctx := NewTestContext(t)
	route, err := client.Network.CreateStaticRoute(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, route)
	assert.Equal(t, 3, route.ID)
	assert.Equal(t, "192.168.100.0/24", route.Destination)
	assert.Equal(t, "192.168.1.10", route.Gateway)
	assert.Equal(t, "Test network route", route.Description)
}

func TestNetworkClient_UpdateStaticRoute(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockRoute := StaticRoute{
		ID:          1,
		Destination: "192.168.100.0/24",
		Gateway:     "192.168.1.20",
		Description: "Updated test route",
	}
	server.SetResponse("staticroute.update", mockRoute)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := StaticRouteCreateRequest{
		Destination: "192.168.100.0/24",
		Gateway:     "192.168.1.20",
		Description: "Updated test route",
	}

	ctx := NewTestContext(t)
	route, err := client.Network.UpdateStaticRoute(ctx, 1, req)
	require.NoError(t, err)
	require.NotNil(t, route)
	assert.Equal(t, 1, route.ID)
	assert.Equal(t, "192.168.100.0/24", route.Destination)
	assert.Equal(t, "192.168.1.20", route.Gateway)
	assert.Equal(t, "Updated test route", route.Description)
}

func TestNetworkClient_DeleteStaticRoute(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("staticroute.delete", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Network.DeleteStaticRoute(ctx, 1)
	assert.NoError(t, err)
}

func TestNetworkClient_GetInterfaceChoices(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockChoices := map[string]string{
		"eth0": "eth0 (Intel Ethernet)",
		"eth1": "eth1 (Realtek Ethernet)",
		"br0":  "br0 (Bridge)",
	}
	server.SetResponse("interface.choices", mockChoices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	choices, err := client.Network.GetInterfaceChoices(ctx)
	require.NoError(t, err)
	assert.Contains(t, choices, "eth0")
	assert.Contains(t, choices, "eth1")
	assert.Contains(t, choices, "br0")
	assert.Equal(t, "eth0 (Intel Ethernet)", choices["eth0"])
	assert.Equal(t, "br0 (Bridge)", choices["br0"])
}

func TestNetworkClient_HasPendingChanges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		response bool
		expected bool
	}{
		{
			name:     "has pending changes",
			response: true,
			expected: true,
		},
		{
			name:     "no pending changes",
			response: false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			server.SetResponse("interface.has_pending_changes", tt.response)

			client := server.CreateTestClient(t)
			defer client.Close()

			ctx := NewTestContext(t)
			hasPending, err := client.Network.HasPendingChanges(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, hasPending)
		})
	}
}

func TestNetworkClient_CommitPendingChanges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		rollback bool
	}{
		{
			name:     "commit without rollback",
			rollback: false,
		},
		{
			name:     "commit with rollback",
			rollback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			server.SetResponse("interface.commit", true)

			client := server.CreateTestClient(t)
			defer client.Close()

			ctx := NewTestContext(t)
			err := client.Network.CommitPendingChanges(ctx, tt.rollback)
			assert.NoError(t, err)
		})
	}
}

func TestNetworkClient_CheckinWaiting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		response bool
		expected bool
	}{
		{
			name:     "checkin waiting",
			response: true,
			expected: true,
		},
		{
			name:     "not waiting for checkin",
			response: false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			server.SetResponse("interface.checkin_waiting", tt.response)

			client := server.CreateTestClient(t)
			defer client.Close()

			ctx := NewTestContext(t)
			waiting, err := client.Network.CheckinWaiting(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, waiting)
		})
	}
}

func TestNetworkClient_Checkin(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("interface.checkin", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Network.Checkin(ctx)
	assert.NoError(t, err)
}

func TestNetworkClient_RollbackPendingChanges(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("interface.rollback", true)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Network.RollbackPendingChanges(ctx)
	assert.NoError(t, err)
}

// Error handling tests

func TestNetworkClient_ListInterfaces_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("interface.query", 500, "Network service unavailable")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Network.ListInterfaces(ctx)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "Network service unavailable", apiErr.Message)
}

func TestNetworkClient_CreateInterface_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("interface.create", 400, "Invalid interface configuration")

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &NetworkInterfaceCreateRequest{
		Name: "invalid",
		Type: InterfaceTypeVLAN,
		// Missing required VLAN parameters
	}

	ctx := NewTestContext(t)
	_, err := client.Network.CreateInterface(ctx, req)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 400, apiErr.Data.Error)
	assert.Equal(t, "Invalid interface configuration", apiErr.Message)
}

func TestNetworkClient_GetConfiguration_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("network.configuration.config", 403, "Access denied")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Network.GetConfiguration(ctx)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 403, apiErr.Data.Error)
	assert.Equal(t, "Access denied", apiErr.Message)
}

func TestNetworkClient_CreateStaticRoute_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("staticroute.create", 422, "Invalid route destination")

	client := server.CreateTestClient(t)
	defer client.Close()

	req := StaticRouteCreateRequest{
		Destination: "invalid-destination",
		Gateway:     "192.168.1.1",
		Description: "Invalid route",
	}

	ctx := NewTestContext(t)
	_, err := client.Network.CreateStaticRoute(ctx, req)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 422, apiErr.Data.Error)
	assert.Equal(t, "Invalid route destination", apiErr.Message)
}

// Test interface with failover configuration

func TestNetworkClient_CreateInterface_Failover(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInterface := NetworkInterface{
		ID:                     6,
		Name:                   "failover0",
		Description:            "Failover interface",
		Type:                   InterfaceTypePhysical,
		IPV4DHCP:               false,
		IPV6Auto:               false,
		MTU:                    1500,
		FailoverCritical:       true,
		FailoverGroup:          1,
		FailoverVhid:           10,
		FailoverAliases:        []string{"192.168.1.50/24"},
		FailoverVirtualAliases: []string{"192.168.1.51/24"},
	}
	server.SetResponse("interface.create", mockInterface)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &NetworkInterfaceCreateRequest{
		Name:                   "failover0",
		Description:            "Failover interface",
		Type:                   InterfaceTypePhysical,
		IPV4DHCP:               false,
		IPV6Auto:               false,
		MTU:                    1500,
		FailoverCritical:       Ptr(true),
		FailoverGroup:          Ptr(1),
		FailoverVhid:           Ptr(10),
		FailoverAliases:        []string{"192.168.1.50/24"},
		FailoverVirtualAliases: []string{"192.168.1.51/24"},
	}

	ctx := NewTestContext(t)
	iface, err := client.Network.CreateInterface(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, iface)
	assert.Equal(t, "failover0", iface.Name)
	assert.True(t, iface.FailoverCritical)
	assert.Equal(t, 1, iface.FailoverGroup)
	assert.Equal(t, 10, iface.FailoverVhid)
	assert.Contains(t, iface.FailoverAliases, "192.168.1.50/24")
	assert.Contains(t, iface.FailoverVirtualAliases, "192.168.1.51/24")
}

// Test update with partial fields

func TestNetworkClient_UpdateInterface_PartialUpdate(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockInterface := NetworkInterface{
		ID:          1,
		Name:        "eth0",
		Description: "Partially updated interface",
		Type:        InterfaceTypePhysical,
		IPV4DHCP:    true,
		IPV6Auto:    false,
		MTU:         1500,
	}
	server.SetResponse("interface.update", mockInterface)

	client := server.CreateTestClient(t)
	defer client.Close()

	// Only update description, leave other fields unchanged
	req := &NetworkInterfaceUpdateRequest{
		Description: Ptr("Partially updated interface"),
	}

	ctx := NewTestContext(t)
	iface, err := client.Network.UpdateInterface(ctx, 1, req)
	require.NoError(t, err)
	require.NotNil(t, iface)
	assert.Equal(t, "Partially updated interface", iface.Description)
	assert.True(t, iface.IPV4DHCP) // Should remain unchanged
}

// Test different LAG protocols

func TestNetworkClient_CreateInterface_LAGProtocols(t *testing.T) {
	t.Parallel()
	protocols := []LAGProtocol{
		LAGProtocolLACP,
		LAGProtocolFailover,
		LAGProtocolLoadBalance,
		LAGProtocolRoundRobin,
		LAGProtocolNone,
	}

	for i, protocol := range protocols {
		t.Run(string(protocol), func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			mockInterface := NetworkInterface{
				ID:          i + 10,
				Name:        "lagg" + string(rune('0'+i)),
				Description: "LAG with " + string(protocol),
				Type:        InterfaceTypeLinkAgg,
				LagProtocol: protocol,
				LagPorts:    []string{"eth1", "eth2"},
			}
			server.SetResponse("interface.create", mockInterface)

			client := server.CreateTestClient(t)
			defer client.Close()

			req := &NetworkInterfaceCreateRequest{
				Name:        "lagg" + string(rune('0'+i)),
				Description: "LAG with " + string(protocol),
				Type:        InterfaceTypeLinkAgg,
				LagProtocol: Ptr(protocol),
				LagPorts:    []string{"eth1", "eth2"},
			}

			ctx := NewTestContext(t)
			iface, err := client.Network.CreateInterface(ctx, req)
			require.NoError(t, err)
			require.NotNil(t, iface)
			assert.Equal(t, protocol, iface.LagProtocol)
		})
	}
}

// Test network configuration with all service announcement options

func TestNetworkClient_UpdateConfiguration_ServiceAnnouncement(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockConfig := NetworkConfiguration{
		ID:       1,
		Hostname: "truenas-with-services",
		ServiceAnnouncement: &ServiceAnnouncement{
			Netbios: true,
			MDNS:    false,
			WSD:     true,
		},
	}
	server.SetResponse("network.configuration.update", mockConfig)

	client := server.CreateTestClient(t)
	defer client.Close()

	updateConfig := &NetworkConfiguration{
		Hostname: "truenas-with-services",
		ServiceAnnouncement: &ServiceAnnouncement{
			Netbios: true,
			MDNS:    false,
			WSD:     true,
		},
	}

	ctx := NewTestContext(t)
	config, err := client.Network.UpdateConfiguration(ctx, updateConfig)
	require.NoError(t, err)
	require.NotNil(t, config)
	require.NotNil(t, config.ServiceAnnouncement)
	assert.True(t, config.ServiceAnnouncement.Netbios)
	assert.False(t, config.ServiceAnnouncement.MDNS)
	assert.True(t, config.ServiceAnnouncement.WSD)
}
