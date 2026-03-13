// Package testvm provides VM management for TrueNAS integration testing.
// It handles starting and stopping TrueNAS VMs for testing purposes.
package testvm

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

// RunWithVM runs a test function with VM setup and teardown.
// This is the main entry point for running integration tests against a TrueNAS VM.
// If the test fails and debug mode is enabled, it will keep the VM running and wait for user input.
func RunWithVM(t *testing.T, f func(*Manager)) {
	t.Helper()
	m := NewManager(t, DefaultConfig())

	// Setup VM
	t.Log("Starting VM")
	if err := m.Start(); err != nil {
		t.Fatalf("start vm: %v", err)
	}
	t.Log("VM is running")

	t.Cleanup(func() {
		// If test failed and debug mode is enabled, wait for user input
		if t.Failed() && os.Getenv("DEBUG_VM") != "" {
			m.waitForUserInput()
		}
		// Cleanup VM
		if err := m.Stop(); err != nil {
			t.Errorf("Cleanup VM: %v", err)
		}
	})

	// Run the actual test
	f(m)
}

type Config struct {
	MemoryMB     uint
	CPUs         int
	Snapshot     string
	Username     string
	Password     string
	WebPort      int
	SSLPort      int
	NumPCIePorts int
}

func DefaultConfig() *Config {
	ports, err := getRandomAvailablePorts(2)
	webPort, sslPort := 8080, 8443 // fallback defaults
	if err == nil {
		webPort, sslPort = ports[0], ports[1]
	}

	return &Config{
		CPUs:         runtime.NumCPU(),
		MemoryMB:     8192,
		WebPort:      webPort,
		SSLPort:      sslPort,
		NumPCIePorts: 5,
		Snapshot:     "truenas.qcow2",
		// NOTE: these credentials need to match what was used to build the original snapshot
		Username: "truenas_admin",
		Password: "hunter2",
	}
}

type Manager struct {
	*testing.T
	config             *Config
	vmProcess          *os.Process
	vmCmd              *exec.Cmd
	monitor            *monitor
	console            *console
	tmpDirs            []string
	availablePCIePorts []string
}

func NewManager(t *testing.T, c *Config) *Manager {
	if c == nil {
		c = DefaultConfig()
	}
	return &Manager{
		T:       t,
		config:  c,
		monitor: newMonitor(t),
		console: newConsole(t),
	}
}

// ConnectionInfo contains the information needed to connect to the TrueNAS VM
type ConnectionInfo struct {
	WebSocketURL string
	Username     string
	Password     string
}

// GetConnectionInfo returns the connection information needed to create a TrueNAS client
func (m *Manager) GetConnectionInfo() ConnectionInfo {
	return ConnectionInfo{
		WebSocketURL: fmt.Sprintf("ws://localhost:%d/api/current", m.config.WebPort),
		Username:     m.config.Username,
		Password:     m.config.Password,
	}
}

func (m *Manager) Start() error {
	// Check if QEMU is available
	if _, err := exec.LookPath("qemu-system-x86_64"); err != nil {
		return fmt.Errorf("qemu-system-x86_64 not found in PATH. Please install QEMU")
	}

	// Check if snapshot image exists or needs to be reassembled
	if err := m.ensureDiskImage(); err != nil {
		return fmt.Errorf("ensure disk image: %w", err)
	}

	// Verify the disk image is valid
	if err := m.verifyDiskImage(); err != nil {
		return fmt.Errorf("disk image verification failed: %w", err)
	}

	// Build QEMU command
	args := []string{
		"-M", "q35",
		"-m", fmt.Sprintf("%d", m.config.MemoryMB),
		"-smp", fmt.Sprintf("%d", m.config.CPUs),
		"-drive", fmt.Sprintf("file=%s,if=virtio,snapshot=on", m.config.Snapshot),
		"-netdev", fmt.Sprintf("user,id=net0,hostfwd=tcp::%d-:80,hostfwd=tcp::%d-:443",
			m.config.WebPort, m.config.SSLPort),
		"-device", "virtio-net,netdev=net0",
		"-qmp", m.monitor.Addr(),
		"-serial", m.console.Addr(),
		"-no-reboot",
		"-nographic", "-display", "none",
	}

	if runtime.GOOS == "linux" {
		args = append(args, "-enable-kvm")
	}

	// Add PCIe root ports for hotplug support
	for i := range m.config.NumPCIePorts {
		addr, port := i+10, i+1
		args = append(args, []string{
			"-device",
			fmt.Sprintf("pcie-root-port,port=0x%d,chassis=%d,id=root_port_%d,bus=pcie.0,addr=0x%d",
				addr, port, port, addr,
			),
		}...)
		m.availablePCIePorts = append(m.availablePCIePorts, fmt.Sprintf("root_port_%d", port))
	}

	cmd := exec.Command("qemu-system-x86_64", args...)
	cmd.Stdin, cmd.Stderr, cmd.Stdout = os.Stdin, os.Stderr, os.Stdout

	m.Logf("Starting TrueNAS VM: %s", cmd.String())
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start VM: %w", err)
	}

	m.vmProcess = cmd.Process
	m.vmCmd = cmd

	// Wait for VM to be ready
	if err := m.waitForReady(); err != nil {
		_ = m.Stop()
		return fmt.Errorf("TrueNAS not ready: %w", err)
	}
	return nil
}

// AddDisk adds a new (empty) disk to the VM.
// Size must be a value accepted by `qemu-img`; e.g. 20M or 2G.
// Returns the name of the attached disk.
func (m *Manager) AddDisk(size string) (string, error) {
	// Create the disk image
	dir, err := os.MkdirTemp("", "vmtest-device-*")
	if err != nil {
		return "", fmt.Errorf("mkdirtemp: %w", err)
	}
	p := filepath.Join(dir, "img.qcow2")

	cmd := exec.CommandContext(m.Context(),
		"qemu-img", "create", "-f", "qcow2", p, size)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("qemu-img command failed: %s: %s: %w",
			cmd.String(), string(out), err)
	}
	m.tmpDirs = append(m.tmpDirs, dir)
	id := "d" + randStr()

	// Add block device via QMP
	driveID := "drive-" + id
	m.monitor.WriteQMP(map[string]any{
		"execute": "blockdev-add",
		"arguments": map[string]any{
			"driver":    "qcow2",
			"node-name": driveID,
			"file": map[string]any{
				"driver":   "file",
				"filename": p,
			},
		},
	})

	// Find an available PCIe root port
	if len(m.availablePCIePorts) < 1 {
		return "", fmt.Errorf("no available PCIe root ports for hotplug")
	}
	bus := m.availablePCIePorts[0]
	m.availablePCIePorts = m.availablePCIePorts[1:]

	// Attach the drive to the PCIe root port via QMP
	deviceID := "virtio-disk-" + id
	m.monitor.WriteQMP(map[string]any{
		"execute": "device_add",
		"arguments": map[string]any{
			"driver": "virtio-blk-pci",
			"drive":  driveID,
			"id":     deviceID,
			"bus":    bus,
			"addr":   "0x0",
			"serial": randStr(),
		},
	})
	return id, nil
}

func (m *Manager) Stop() error {
	if m.vmProcess != nil {
		if err := m.vmProcess.Kill(); err != nil {
			return fmt.Errorf("kill VM process: %w", err)
		}
		m.vmProcess = nil
	}
	m.console.Close()
	m.monitor.Close()

	for _, d := range m.tmpDirs {
		if err := os.RemoveAll(d); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("cleanup: %s: %w", d, err)
		}
	}
	return nil
}

func (m *Manager) waitForReady() error {
	webURL := fmt.Sprintf("http://localhost:%d", m.config.WebPort)
	timeout := 15 * time.Minute // TrueNAS can take a while to boot
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(m.Context(), timeout)
	defer cancel()

	// Monitor process state
	processDone := make(chan error, 1)
	go func() {
		if m.vmCmd != nil {
			processDone <- m.vmCmd.Wait()
		}
	}()

	m.Log("Waiting for TrueNAS to boot and become ready...")
	m.Logf("This can take several minutes; timeout=%s", timeout.String())

	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for TrueNAS to become ready")
		case err := <-processDone:
			return fmt.Errorf("VM process exited unexpectedly: %w", err)
		case <-ticker.C:
			if m.checkHTTPEndpoint(ctx, webURL) {
				m.Log("TrueNAS is ready!")
				return nil
			}
			m.Logf("TrueNAS not ready yet... (%s)\n",
				time.Since(start).Truncate(time.Second).String())
		}
	}
}

func (m *Manager) checkHTTPEndpoint(ctx context.Context, endpoint string) bool {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, http.NoBody)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Accept any response (including redirects) as "ready"
	// TrueNAS might redirect to login page, which is still "ready"
	return resp.StatusCode < 500
}

// getRandomAvailablePorts returns the requested number of random available ports
func getRandomAvailablePorts(count int) ([]int, error) {
	listeners := make([]net.Listener, count)
	ports := make([]int, count)

	// Open all listeners first to reserve the ports
	for i := range count {
		listener, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			// Close any listeners we've opened so far
			for j := range i {
				listeners[j].Close()
			}
			return nil, err
		}
		listeners[i] = listener
		ports[i] = listener.Addr().(*net.TCPAddr).Port
	}

	// Close all listeners after getting the ports
	for _, listener := range listeners {
		listener.Close()
	}
	return ports, nil
}

// waitForUserInput prompts the user and waits for input before proceeding with cleanup
func (m *Manager) waitForUserInput() {
	connInfo := m.GetConnectionInfo()
	separator := strings.Repeat("=", 80)

	m.Log(separator)
	m.Log("TEST FAILED - VM DEBUGGING MODE ENABLED")
	m.Log(separator)
	m.Log("The TrueNAS VM is still running for debugging purposes.")
	m.Log("Connection Information:")
	m.Logf("  WebSocket URL: %s", connInfo.WebSocketURL)
	m.Logf("  Username:      %s", connInfo.Username)
	m.Logf("  Password:      %s", connInfo.Password)
	m.Logf("  Web UI:        http://localhost:%d", m.config.WebPort)
	m.Logf("  SSL Web UI:    https://localhost:%d", m.config.SSLPort)
	m.Logf("  VM Process ID: %d", m.vmProcess.Pid)
	m.Log("You can now:")
	m.Log("  - Connect to the VM for debugging")
	m.Log("  - Run additional tests manually")
	m.Log("  - Examine logs and system state")
	m.Log("Press Ctrl+C when you're done debugging to stop the VM and cleanup...")
	m.Log(separator)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	m.Log("Proceeding with VM cleanup...")
}

func randStr() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ensureDiskImage checks if the disk image exists, and if not, reassembles it from split parts
func (m *Manager) ensureDiskImage() error {
	diskPath, err := filepath.Abs(m.config.Snapshot)
	if err != nil {
		return fmt.Errorf("make absolute: %s: %w", diskPath, err)
	}
	m.config.Snapshot = diskPath

	// If the complete disk image already exists, we're done
	if _, err := os.Stat(diskPath); err == nil {
		return nil
	}

	// Check if split parts exist
	partPattern := diskPath + ".part*"
	matches, err := filepath.Glob(partPattern)
	if err != nil {
		return fmt.Errorf("glob pattern %s: %w", partPattern, err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("disk image missing and no split parts found: %s", diskPath)
	}

	m.Logf("Reassembling disk image from %d parts...", len(matches))

	// Create output file
	outFile, err := os.Create(diskPath)
	if err != nil {
		return fmt.Errorf("create output file %s: %w", diskPath, err)
	}
	defer outFile.Close()

	// Sort part files to ensure correct assembly order
	slices.Sort(matches)

	// Copy each part file in order
	for _, partPath := range matches {
		partFile, err := os.Open(partPath)
		if err != nil {
			os.Remove(diskPath) // Clean up on error
			return fmt.Errorf("open part file %s: %w", partPath, err)
		}

		if _, err := outFile.ReadFrom(partFile); err != nil {
			partFile.Close()
			os.Remove(diskPath) // Clean up on error
			return fmt.Errorf("copy part file %s: %w", partPath, err)
		}
		partFile.Close()
	}

	m.Logf("Successfully reassembled disk image: %s", diskPath)
	return nil
}

// verifyDiskImage checks if the disk image is valid and readable by QEMU
func (m *Manager) verifyDiskImage() error {
	cmd := exec.Command("qemu-img", "info", m.config.Snapshot)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qemu-img info failed: %s: %w", string(output), err)
	}
	m.Logf("Disk image info:\n%s", string(output))
	return nil
}
