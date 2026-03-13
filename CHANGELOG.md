# Changelog

All notable changes to this project will be documented in this file.

## [0.2.0]

### Breaking Changes
- Upgraded from proprietary WebSocket protocol to standard JSON-RPC 2.0
  - WebSocket endpoint changed from `/websocket` to `/api/current`
  - Removed legacy `connect`/`connected` handshake
  - Message ID type changed from `string` to `int64`
  - Error type changed from `ErrorMsg` to `RPCError` with nested `RPCErrorData`
  - `Message` type replaced by separate `Request` and `Response` types
- Requires TrueNAS SCALE 25.10.2 or later (no backwards compatibility with older versions)

### Added
- Server-sent notification support via `Options.NotificationHandler`
- `RPCError`, `RPCErrorData`, and `RPCErrorTrace` types for structured JSON-RPC 2.0 error handling

### Removed
- `Message` and `ErrorMsg` types (replaced by `Request`, `Response`, `RPCError`)
- WebSocket connection handshake (`connect`/`connected` exchange)

### Changed
- `Call()` and `CallJob()` now use JSON-RPC 2.0 request/response format internally
- Updated test infrastructure to use JSON-RPC 2.0 protocol
- Tested against TrueNAS SCALE 25.10.2

## [0.1.3]

### Fixed
- Fixed data race in WebSocket client between Close() and Call() methods
- Fixed race condition in connectionManager when sending to closed reconnectCh

## [0.1.2]

### Fixed
- Handle unmarshaling PoolScan timestamps

## [0.1.0]

### Added
- Initial implementation of TrueNAS WebSocket API client
- Support for most of the TrueNAS API surface including:
  - Authentication (username/password and API key)
  - Pool management
  - Dataset operations
  - Service management
  - System information
  - Network configuration
  - File sharing (SMB, NFS, AFP, WebDAV)
  - User and group management
  - Certificate management
  - Job management
  - Alert system
  - Boot management
  - Disk operations
  - Filesystem operations
  - VM management
  - Smart monitoring
- Thread-safe WebSocket client with automatic reconnection
- Context-based timeout support
- Comprehensive test suite
- Type-safe API clients for all endpoints

[0.2.0]: https://github.com/715d/go-truenas/compare/v0.1.3...v0.2.0
[0.1.3]: https://github.com/715d/go-truenas/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/715d/go-truenas/compare/v0.1.1...v0.1.2
[0.1.0]: https://github.com/715d/go-truenas/releases/tag/v0.1.0
