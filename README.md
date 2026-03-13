# go-truenas

[![Test](https://github.com/715d/go-truenas/actions/workflows/test.yml/badge.svg)](https://github.com/715d/go-truenas/actions/workflows/test.yml)
[![GoDoc](https://godoc.org/github.com/715d/go-truenas?status.svg)](https://godoc.org/github.com/715d/go-truenas)

A Go client library for interacting with the TrueNAS WebSocket API.

[Official TrueNAS Websocket API Documentation](https://www.truenas.com/docs/api/scale_websocket_api.html)

_**Disclaimer**_: Much of this code was generated from the API documentation. There may be cases where the API behavior does not match the spec. A subset of methods are fully tested with a live TrueNAS instance as part of an integration test suite. See [Contributing](#Contributing) to help increase this coverage. Any workflows using this library should _always_ be tested with a throwaway TrueNAS instance before connecting to one with important data.

Currently tested against `TrueNAS-SCALE-25.10.2`.

## Installation

```bash
go get github.com/715d/go-truenas
```

## Usage

### Basic Example

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "time"
    
    "github.com/715d/go-truenas/truenas"
)

func main() {
    // Connect with username/password
    client, err := truenas.NewClient("ws://your-truenas-host/api/current", truenas.Options{
        Username: "your-username",
        Password: "your-password",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Use context with timeout for operations
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Get system information using type-safe client methods
    info, err := client.System.GetInfo(ctx)
    if err != nil {
        var apiErr *truenas.ErrorMsg
        if errors.As(err, &apiErr) {
            log.Printf("TrueNAS API error: %s (code: %d)", apiErr.Message, apiErr.Code)
        } else {
            log.Printf("Connection error: %v", err)
        }
        return
    }
    
    fmt.Printf("TrueNAS %s on %s (uptime: %s)\n", info.Version, info.Hostname, info.Uptime)
}
```

### Authentication Options

```go
// Username and password
client, err := truenas.NewClient("wss://truenas.local/api/current", truenas.Options{
    Username: "admin",
    Password: "your-password",
})

// API key authentication (recommended for scripts)
client, err := truenas.NewClient("wss://truenas.local/api/current", truenas.Options{
    APIKey: "your-api-key-token",
})
```

### Common Operations

```go
// Create a user
userReq := &truenas.UserCreateRequest{
    Username:    "testuser",
    FullName:    "Test User",
    GroupCreate: truenas.Ptr(true),
    Password:    "securepassword",
    Home:        "/home/testuser",
    Shell:       "/bin/bash",
}
user, err := client.User.Create(ctx, userReq)

// Start a service
err = client.Service.Start(ctx, "nfs")

// Create a dataset
datasetReq := truenas.DatasetCreateRequest{
    Name:   "mypool/mydataset",
    Type:   truenas.DatasetTypeFilesystem,
    Aclmode: truenas.Ptr(truenas.AclModePassthrough),
}
dataset, err := client.Dataset.Create(ctx, datasetReq)
```

### Low-Level API Access

For APIs not yet covered by type-safe methods:

```go
// Direct API call
var result any
err := client.Call(ctx, "system.info", nil, &result)

// Long-running operations (jobs)
var jobResult truenas.JobResult
err := client.CallJob(ctx, "pool.create", poolParams, &jobResult)
```

## Contributing

### Prerequisites

Before contributing, ensure you have the following installed:

- Go 1.24 or later
- Git LFS (required for VM integration tests)
- QEMU (required for VM integration tests)
- Make

### Initial Setup

1. **Clone the repository with Git LFS support:**
   ```bash
   git clone https://github.com/715d/go-truenas.git
   cd go-truenas
   
   # Initialize Git LFS and download VM images
   git lfs install
   git lfs pull
   ```

2. **Install development dependencies:**
   ```bash
   make setup-dev
   ```

### Development Workflow

1. **Make your changes** following the existing code style
2. **Format your code:**
   ```bash
   make fmt
   ```
3. **Run linting:**
   ```bash
   make lint
   ```
4. **Run tests:**
   ```bash
   # Quick checks (unit tests only)
   make check
   
   # Full validation including VM tests
   make check-full
   ```

### Git LFS for VM Tests

The project uses Git LFS to store the TrueNAS VM image (`truenas/truenas.qcow2`) required for integration tests. This keeps the repository lightweight while providing the necessary test infrastructure.

**Important:** New contributors must have Git LFS installed and run `git lfs pull` after cloning to download the VM image. Without this, VM-based integration tests will fail.

### Testing Strategy

- **Unit Tests:** Fast tests that don't require external dependencies (`make test-unit`)
- **VM Tests:** Integration tests against a QEMU-based TrueNAS VM (`make test-vm`)

Always run `make check` before submitting pull requests. For major changes, also run `make check-full` to ensure VM-based tests pass.

### Updating the VM test image

As new versions of TrueNAS Scale are released, we may need to update the QCOW2 image used in the integration tests.

1. Download the [latest TrueNAS Scale installation ISO](https://www.truenas.com/download-truenas-community-edition/).
2. Create a blank disk image

    ```sh
    qemu-img create -f qcow2 truenas.qcow2 16G
    ```
3. Launch a VM using that disk and ISO.

    ```sh
    qemu-system-x86_64 -m 8192 -smp 10 -hda truenas.qcow2 -cdrom truenas.iso -boot d -netdev user,id=net0,hostfwd=tcp::65525-:80,hostfwd=tcp::65526-:443 -device virtio-net,netdev=net0
    ```
4. Go through the installation process and configure the admin credentials to match the integration test (or update accordingly). Do not enable EFI.
5. Configure the GRUB loader to have no wait and to default to booting TrueNAS immediately.

    ```sh
    mount -o remount,rw /
    echo 'GRUB_TIMEOUT=0' > /etc/default/grub.d/zz-custom.cfg
    echo 'GRUB_DEFAULT=0' >> /etc/default/grub.d/zz-custom.cfg
    update-grub
    ```
6. Shutdown the VM.
7. Split the image for Git LFS compatibility:

    ```sh
    # Remove old split files if they exist
    rm -f truenas/truenas.qcow2.part*

    # Split the new image into 512M chunks
    cd truenas
    split -b 512M truenas.qcow2 truenas.qcow2.part

    # Remove the original large file (it will be reassembled automatically during tests)
    rm truenas.qcow2
    ```
8. Commit the split files:

    ```sh
    git add truenas/truenas.qcow2.part*
    git commit -m "Update TrueNAS VM image to [version]"
    ```

The test framework will automatically reassemble the split files (`truenas.qcow2.partaa`, `truenas.qcow2.partab`, etc.) into the full `truenas.qcow2` image before starting the VM.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

### Third-Party Software

This repository includes a TrueNAS SCALE virtual machine image (`truenas/truenas.qcow2`) for integration testing purposes.

**TrueNAS SCALE Attribution:**
- TrueNAS SCALE is developed by iXsystems, Inc.
- Version: TrueNAS-SCALE-25.10.2 (approximate)
- Website: https://www.truenas.com/
- License: Mixed licensing (BSD-3-Clause for middleware/GUI, GPL for kernel components)
- Copyright: © iXsystems, Inc.

**Important Notice:** TrueNAS SCALE software may not be commercially distributed or sold without an addendum license agreement and express written consent from iXsystems. This VM image is included solely for development and testing purposes of this open source client library.

The VM image is used under the terms that allow for non-commercial use and testing. For commercial use or redistribution of TrueNAS components, please contact iXsystems directly at https://www.truenas.com/.

**Open Source Components:** TrueNAS SCALE contains various open source software components, each licensed under their respective open source license agreements. For a complete list of licenses, please refer to the TrueNAS SCALE documentation.
