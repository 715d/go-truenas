package truenas

import (
	"context"
	"fmt"
)

// DatasetType represents the type of a ZFS dataset
type DatasetType string

const (
	DatasetTypeFilesystem DatasetType = "FILESYSTEM"
	DatasetTypeVolume     DatasetType = "VOLUME"
)

// DatasetShareType represents the share type for a dataset
type DatasetShareType string

const (
	DatasetShareTypeGeneric DatasetShareType = "GENERIC"
	DatasetShareTypeSMB     DatasetShareType = "SMB"
)

// DatasetCaseSensitivity represents case sensitivity options
type DatasetCaseSensitivity string

const (
	DatasetCaseSensitivitySensitive   DatasetCaseSensitivity = "SENSITIVE"
	DatasetCaseSensitivityInsensitive DatasetCaseSensitivity = "INSENSITIVE"
	DatasetCaseSensitivityMixed       DatasetCaseSensitivity = "MIXED"
)

// DatasetSync represents sync options
type DatasetSync string

const (
	DatasetSyncStandard DatasetSync = "STANDARD"
	DatasetSyncAlways   DatasetSync = "ALWAYS"
	DatasetSyncDisabled DatasetSync = "DISABLED"
	DatasetSyncInherit  DatasetSync = "INHERIT"
)

// DatasetOnOff represents on/off/inherit options
type DatasetOnOff string

const (
	DatasetOnOffOn      DatasetOnOff = "ON"
	DatasetOnOffOff     DatasetOnOff = "OFF"
	DatasetOnOffInherit DatasetOnOff = "INHERIT"
)

// DatasetVolBlockSize represents volume block size options
type DatasetVolBlockSize string

const (
	DatasetVolBlockSize512  DatasetVolBlockSize = "512"
	DatasetVolBlockSize1K   DatasetVolBlockSize = "1K"
	DatasetVolBlockSize2K   DatasetVolBlockSize = "2K"
	DatasetVolBlockSize4K   DatasetVolBlockSize = "4K"
	DatasetVolBlockSize8K   DatasetVolBlockSize = "8K"
	DatasetVolBlockSize16K  DatasetVolBlockSize = "16K"
	DatasetVolBlockSize32K  DatasetVolBlockSize = "32K"
	DatasetVolBlockSize64K  DatasetVolBlockSize = "64K"
	DatasetVolBlockSize128K DatasetVolBlockSize = "128K"
)

// DatasetClient provides methods for dataset management
type DatasetClient struct {
	client *Client
}

// NewDatasetClient creates a new dataset client
func NewDatasetClient(client *Client) *DatasetClient {
	return &DatasetClient{client: client}
}

// Dataset represents a ZFS dataset
type Dataset struct {
	ID             string                      `json:"id"`
	Type           DatasetType                 `json:"type"`
	Name           string                      `json:"name"`
	Pool           string                      `json:"pool"`
	Encrypted      bool                        `json:"encrypted"`
	EncryptionRoot string                      `json:"encryption_root"`
	KeyLoaded      bool                        `json:"key_loaded"`
	Children       []Dataset                   `json:"children"`
	Mountpoint     any                         `json:"mountpoint"`
	Locked         bool                        `json:"locked"`
	UserProperties map[string]*DatasetProperty `json:"user_properties"`

	// ZFS Properties
	ManagedBy             *DatasetProperty `json:"managedby,omitempty"`
	Deduplication         *DatasetProperty `json:"deduplication,omitempty"`
	AclMode               *DatasetProperty `json:"aclmode,omitempty"`
	AclType               *DatasetProperty `json:"acltype,omitempty"`
	Xattr                 *DatasetProperty `json:"xattr,omitempty"`
	Atime                 *DatasetProperty `json:"atime,omitempty"`
	CaseSensitivity       *DatasetProperty `json:"casesensitivity,omitempty"`
	Checksum              *DatasetProperty `json:"checksum,omitempty"`
	Exec                  *DatasetProperty `json:"exec,omitempty"`
	Sync                  *DatasetProperty `json:"sync,omitempty"`
	Compression           *DatasetProperty `json:"compression,omitempty"`
	CompressRatio         *DatasetProperty `json:"compressratio,omitempty"`
	Origin                *DatasetProperty `json:"origin,omitempty"`
	Quota                 *DatasetProperty `json:"quota,omitempty"`
	RefQuota              *DatasetProperty `json:"refquota,omitempty"`
	Reservation           *DatasetProperty `json:"reservation,omitempty"`
	RefReservation        *DatasetProperty `json:"refreservation,omitempty"`
	Copies                *DatasetProperty `json:"copies,omitempty"`
	SnapDir               *DatasetProperty `json:"snapdir,omitempty"`
	ReadOnly              *DatasetProperty `json:"readonly,omitempty"`
	RecordSize            *DatasetProperty `json:"recordsize,omitempty"`
	KeyFormat             *DatasetProperty `json:"key_format,omitempty"`
	EncryptionAlgorithm   *DatasetProperty `json:"encryption_algorithm,omitempty"`
	Used                  *DatasetProperty `json:"used,omitempty"`
	UsedByChildren        *DatasetProperty `json:"usedbychildren,omitempty"`
	UsedByDataset         *DatasetProperty `json:"usedbydataset,omitempty"`
	UsedByRefReservation  *DatasetProperty `json:"usedbyrefreservation,omitempty"`
	UsedBySnapshots       *DatasetProperty `json:"usedbysnapshots,omitempty"`
	Available             *DatasetProperty `json:"available,omitempty"`
	SpecialSmallBlockSize *DatasetProperty `json:"special_small_block_size,omitempty"`
	PBKDF2Iters           *DatasetProperty `json:"pbkdf2iters,omitempty"`
	Creation              *DatasetProperty `json:"creation,omitempty"`
	SnapDev               *DatasetProperty `json:"snapdev,omitempty"`
	VolSize               *DatasetProperty `json:"volsize,omitempty"`
	VolBlockSize          *DatasetProperty `json:"volblocksize,omitempty"`
}

// DatasetProperty represents a ZFS property with its value and metadata
type DatasetProperty struct {
	Parsed     any    `json:"parsed"`
	RawValue   string `json:"rawvalue"`
	Value      string `json:"value"`
	Source     string `json:"source"`
	SourceInfo any    `json:"source_info"`
}

// DatasetCreateRequest represents parameters for pool.dataset.create
type DatasetCreateRequest struct {
	Name              string            `json:"name"`
	Type              DatasetType       `json:"type,omitempty"`
	Properties        map[string]any    `json:"properties,omitempty"`
	UserProperties    map[string]string `json:"user_properties,omitempty"`
	Encryption        *bool             `json:"encryption,omitempty"`
	EncryptionOptions any               `json:"encryption_options,omitempty"`
	Inherit           *bool             `json:"inherit_encryption,omitempty"`

	// Volume-specific fields (required for VOLUME type)
	Volsize      *int64               `json:"volsize,omitempty"`
	Volblocksize *DatasetVolBlockSize `json:"volblocksize,omitempty"`
	Sparse       *bool                `json:"sparse,omitempty"`

	// Optional fields
	Comments        *string                 `json:"comments,omitempty"`
	ShareType       *DatasetShareType       `json:"share_type,omitempty"`
	CaseSensitivity *DatasetCaseSensitivity `json:"casesensitivity,omitempty"`
	ForceSize       *bool                   `json:"force_size,omitempty"`

	// Common ZFS properties
	Sync           *DatasetSync  `json:"sync,omitempty"`
	Compression    *string       `json:"compression,omitempty"`
	Atime          *DatasetOnOff `json:"atime,omitempty"`
	Exec           *DatasetOnOff `json:"exec,omitempty"`
	Quota          *int64        `json:"quota,omitempty"`
	Refquota       *int64        `json:"refquota,omitempty"`
	Reservation    *int64        `json:"reservation,omitempty"`
	Refreservation *int64        `json:"refreservation,omitempty"`
}

// DatasetUpdateRequest represents parameters for pool.dataset.update
type DatasetUpdateRequest struct {
	Properties     map[string]any    `json:"properties,omitempty"`
	UserProperties map[string]string `json:"user_properties,omitempty"`

	// Volume-specific fields
	Volsize   *int64 `json:"volsize,omitempty"`
	ForceSize *bool  `json:"force_size,omitempty"`

	// Optional fields
	Comments *string `json:"comments,omitempty"`

	// ZFS properties that can be updated
	Sync                  *DatasetSync  `json:"sync,omitempty"`
	Compression           *string       `json:"compression,omitempty"`
	Atime                 *DatasetOnOff `json:"atime,omitempty"`
	Exec                  *DatasetOnOff `json:"exec,omitempty"`
	Quota                 *int64        `json:"quota,omitempty"`
	QuotaWarning          *int64        `json:"quota_warning,omitempty"`
	QuotaCritical         *int64        `json:"quota_critical,omitempty"`
	Refquota              *int64        `json:"refquota,omitempty"`
	RefquotaWarning       *int64        `json:"refquota_warning,omitempty"`
	RefquotaCritical      *int64        `json:"refquota_critical,omitempty"`
	Reservation           *int64        `json:"reservation,omitempty"`
	Refreservation        *int64        `json:"refreservation,omitempty"`
	Copies                *int          `json:"copies,omitempty"`
	Readonly              *DatasetOnOff `json:"readonly,omitempty"`
	Recordsize            *string       `json:"recordsize,omitempty"`
	Snapdir               *DatasetOnOff `json:"snapdir,omitempty"`
	Deduplication         *DatasetOnOff `json:"deduplication,omitempty"`
	Checksum              *string       `json:"checksum,omitempty"`
	Managedby             *string       `json:"managedby,omitempty"`
	SpecialSmallBlockSize *int64        `json:"special_small_block_size,omitempty"`
}

// DatasetDeleteRequest represents parameters for pool.dataset.delete
type DatasetDeleteRequest struct {
	Recursive *bool `json:"recursive,omitempty"`
	Force     *bool `json:"force,omitempty"`
}

// DatasetLockRequest represents parameters for pool.dataset.lock
type DatasetLockRequest struct {
	PassPhrase  string `json:"passphrase,omitempty"`
	KeyFile     string `json:"key_file,omitempty"`
	ForceUmount *bool  `json:"force_umount,omitempty"`
}

// DatasetUnlockRequest represents parameters for pool.dataset.unlock
type DatasetUnlockRequest struct {
	Datasets          []DatasetUnlockEntry `json:"datasets"`
	Services          *bool                `json:"services,omitempty"`
	KeyFile           *bool                `json:"key_file,omitempty"`
	Recursive         *bool                `json:"recursive,omitempty"`
	ToggleAttachments *bool                `json:"toggle_attachments,omitempty"`
}

// DatasetUnlockEntry represents a single dataset unlock entry
type DatasetUnlockEntry struct {
	Name       string `json:"name"`
	PassPhrase string `json:"passphrase,omitempty"`
	KeyFile    string `json:"key_file,omitempty"`
}

// DatasetSnapshotRequest represents parameters for pool.dataset.snapshot
type DatasetSnapshotRequest struct {
	Dataset    string         `json:"dataset"`
	Name       string         `json:"name"`
	Naming     string         `json:"naming_schema,omitempty"`
	VmwareSync *bool          `json:"vmware_sync,omitempty"`
	Recursive  *bool          `json:"recursive,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

// List returns all datasets
func (d *DatasetClient) List(ctx context.Context) ([]Dataset, error) {
	var result []Dataset
	err := d.client.Call(ctx, "pool.dataset.query", []any{}, &result)
	return result, err
}

// Get returns a specific dataset by ID
func (d *DatasetClient) Get(ctx context.Context, id string) (*Dataset, error) {
	var result []Dataset
	err := d.client.Call(ctx, "pool.dataset.query", []any{[]any{[]any{"id", "=", id}}}, &result)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, NewNotFoundError("dataset", fmt.Sprintf("ID %s", id))
	}
	return &result[0], nil
}

// GetByName returns a specific dataset by name
func (d *DatasetClient) GetByName(ctx context.Context, name string) (*Dataset, error) {
	var result []Dataset
	err := d.client.Call(ctx, "pool.dataset.query", []any{[]any{[]any{"name", "=", name}}}, &result)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, NewNotFoundError("dataset", fmt.Sprintf("name %s", name))
	}
	return &result[0], nil
}

// Create creates a new dataset
func (d *DatasetClient) Create(ctx context.Context, req *DatasetCreateRequest) (*Dataset, error) {
	var result Dataset
	err := d.client.Call(ctx, "pool.dataset.create", []any{*req}, &result)
	return &result, err
}

// Update updates an existing dataset
func (d *DatasetClient) Update(ctx context.Context, id string, req DatasetUpdateRequest) (*Dataset, error) {
	var result Dataset
	err := d.client.Call(ctx, "pool.dataset.update", []any{id, req}, &result)
	return &result, err
}

// Delete deletes a dataset
func (d *DatasetClient) Delete(ctx context.Context, id string, req DatasetDeleteRequest) error {
	return d.client.Call(ctx, "pool.dataset.delete", []any{id, req}, nil)
}

// Lock locks an encrypted dataset
func (d *DatasetClient) Lock(ctx context.Context, id string, req DatasetLockRequest) error {
	params := []any{id}
	if req.PassPhrase != "" || req.KeyFile != "" || value(req.ForceUmount) {
		params = append(params, req)
	}
	return d.client.CallJob(ctx, "pool.dataset.lock", params, nil)
}

// Unlock unlocks encrypted datasets
func (d *DatasetClient) Unlock(ctx context.Context, id string, req DatasetUnlockRequest) error {
	return d.client.CallJob(ctx, "pool.dataset.unlock", []any{id, req}, nil)
}

// Mount mounts a dataset
func (d *DatasetClient) Mount(ctx context.Context, id string) error {
	return d.client.Call(ctx, "pool.dataset.mount", []any{id}, nil)
}

// Unmount unmounts a dataset
func (d *DatasetClient) Unmount(ctx context.Context, id string, force bool) error {
	params := []any{id}
	if force {
		params = append(params, map[string]any{"force": true})
	}
	return d.client.Call(ctx, "pool.dataset.umount", params, nil)
}

// Snapshot creates a snapshot of a dataset
func (d *DatasetClient) Snapshot(ctx context.Context, req DatasetSnapshotRequest) (any, error) {
	var result any
	err := d.client.Call(ctx, "zfs.snapshot.create", []any{req}, &result)
	return result, err
}

// GetSnapshots returns snapshots for a dataset
func (d *DatasetClient) GetSnapshots(ctx context.Context, datasetName string) ([]any, error) {
	var result []any
	err := d.client.Call(ctx, "zfs.snapshot.query", []any{[]any{[]any{"dataset", "=", datasetName}}}, &result)
	return result, err
}

// Promote promotes a clone to become the origin
func (d *DatasetClient) Promote(ctx context.Context, id string) error {
	return d.client.Call(ctx, "pool.dataset.promote", []any{id}, nil)
}

// GetProcesses returns processes using the dataset
func (d *DatasetClient) GetProcesses(ctx context.Context, id string) (any, error) {
	var result any
	err := d.client.Call(ctx, "pool.dataset.processes", []any{id}, &result)
	return result, err
}
