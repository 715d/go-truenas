package truenas

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFilesystemClient(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	client := server.CreateTestClient(t)
	defer client.Close()

	// Test that the filesystem client is properly initialized
	require.NotNil(t, client.Filesystem)
	require.NotNil(t, client.Filesystem.client)
	assert.Equal(t, client, client.Filesystem.client)
}

func TestFilesystemClient_Stat(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockStat := FilesystemStat{
		Size:       1024,
		Mode:       0755,
		UID:        1000,
		GID:        1000,
		Atime:      time.Now(),
		Mtime:      time.Now(),
		Ctime:      time.Now(),
		Dev:        2049,
		Inode:      12345,
		Nlink:      1,
		User:       "testuser",
		Group:      "testgroup",
		Acl:        false,
		IsFile:     true,
		IsDir:      false,
		IsSymlink:  false,
		IsCharDev:  false,
		IsBlockDev: false,
		IsFIFO:     false,
		IsSocket:   false,
		RealPath:   "/mnt/tank/testfile.txt",
	}
	server.SetResponse("filesystem.stat", mockStat)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	stat, err := client.Filesystem.Stat(ctx, "/mnt/tank/testfile.txt")
	require.NoError(t, err)
	require.NotNil(t, stat)
	assert.Equal(t, int64(1024), stat.Size)
	assert.Equal(t, 0755, stat.Mode)
	assert.Equal(t, 1000, stat.UID)
	assert.Equal(t, 1000, stat.GID)
	assert.Equal(t, "testuser", stat.User)
	assert.Equal(t, "testgroup", stat.Group)
	assert.True(t, stat.IsFile)
	assert.False(t, stat.IsDir)
	assert.Equal(t, "/mnt/tank/testfile.txt", stat.RealPath)
}

func TestFilesystemClient_Stat_Directory(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockStat := FilesystemStat{
		Size:      4096,
		Mode:      0755,
		UID:       0,
		GID:       0,
		User:      "root",
		Group:     "wheel",
		Acl:       true,
		IsFile:    false,
		IsDir:     true,
		IsSymlink: false,
		RealPath:  "/mnt/tank/testdir",
	}
	server.SetResponse("filesystem.stat", mockStat)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	stat, err := client.Filesystem.Stat(ctx, "/mnt/tank/testdir")
	require.NoError(t, err)
	require.NotNil(t, stat)
	assert.Equal(t, int64(4096), stat.Size)
	assert.Equal(t, "root", stat.User)
	assert.Equal(t, "wheel", stat.Group)
	assert.False(t, stat.IsFile)
	assert.True(t, stat.IsDir)
	assert.True(t, stat.Acl)
}

func TestFilesystemClient_Stat_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("filesystem.stat", 404, "Path not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	stat, err := client.Filesystem.Stat(ctx, "/nonexistent")
	require.Error(t, err)
	assert.Nil(t, stat)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "Path not found", apiErr.Message)
}

func TestFilesystemClient_Statfs(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockStatfs := FilesystemStatfs{
		FreeBytes:  1073741824,  // 1GB
		AvailBytes: 1073741824,  // 1GB
		TotalBytes: 10737418240, // 10GB
		TotalFiles: 1000000,
		FreeFiles:  999000,
		NameMax:    255,
		Fstype:     "zfs",
		Flags:      []string{"rw", "relatime"},
	}
	server.SetResponse("filesystem.statfs", mockStatfs)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	statfs, err := client.Filesystem.Statfs(ctx, "/mnt/tank")
	require.NoError(t, err)
	require.NotNil(t, statfs)
	assert.Equal(t, int64(1073741824), statfs.FreeBytes)
	assert.Equal(t, int64(1073741824), statfs.AvailBytes)
	assert.Equal(t, int64(10737418240), statfs.TotalBytes)
	assert.Equal(t, int64(1000000), statfs.TotalFiles)
	assert.Equal(t, int64(999000), statfs.FreeFiles)
	assert.Equal(t, 255, statfs.NameMax)
	assert.Equal(t, "zfs", statfs.Fstype)
	assert.Equal(t, []string{"rw", "relatime"}, statfs.Flags)
}

func TestFilesystemClient_Statfs_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("filesystem.statfs", 500, "Filesystem unavailable")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	statfs, err := client.Filesystem.Statfs(ctx, "/mnt/tank")
	require.Error(t, err)
	assert.Nil(t, statfs)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "Filesystem unavailable", apiErr.Message)
}

func TestFilesystemClient_ListDir(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockEntries := []DirEntry{
		{
			Name:     "file1.txt",
			Path:     "/mnt/tank/file1.txt",
			RealPath: "/mnt/tank/file1.txt",
			Type:     "FILE",
			Size:     1024,
			Mode:     0644,
			UID:      1000,
			GID:      1000,
			Mtime:    time.Now(),
			HasACL:   false,
		},
		{
			Name:     "dir1",
			Path:     "/mnt/tank/dir1",
			RealPath: "/mnt/tank/dir1",
			Type:     "DIRECTORY",
			Size:     4096,
			Mode:     0755,
			UID:      1000,
			GID:      1000,
			Mtime:    time.Now(),
			HasACL:   true,
		},
		{
			Name:     "link1",
			Path:     "/mnt/tank/link1",
			RealPath: "/mnt/tank/file1.txt",
			Type:     "SYMLINK",
			Size:     9,
			Mode:     0777,
			UID:      1000,
			GID:      1000,
			Mtime:    time.Now(),
			HasACL:   false,
		},
	}
	server.SetResponse("filesystem.listdir", mockEntries)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	entries, err := client.Filesystem.ListDir(ctx, "/mnt/tank")
	require.NoError(t, err)
	require.Len(t, entries, 3)

	// Test file entry
	fileEntry := entries[0]
	assert.Equal(t, "file1.txt", fileEntry.Name)
	assert.Equal(t, "/mnt/tank/file1.txt", fileEntry.Path)
	assert.Equal(t, "/mnt/tank/file1.txt", fileEntry.RealPath)
	assert.Equal(t, "FILE", fileEntry.Type)
	assert.Equal(t, int64(1024), fileEntry.Size)
	assert.Equal(t, 0644, fileEntry.Mode)
	assert.False(t, fileEntry.HasACL)

	// Test directory entry
	dirEntry := entries[1]
	assert.Equal(t, "dir1", dirEntry.Name)
	assert.Equal(t, "DIRECTORY", dirEntry.Type)
	assert.Equal(t, int64(4096), dirEntry.Size)
	assert.Equal(t, 0755, dirEntry.Mode)
	assert.True(t, dirEntry.HasACL)

	// Test symlink entry
	linkEntry := entries[2]
	assert.Equal(t, "link1", linkEntry.Name)
	assert.Equal(t, "SYMLINK", linkEntry.Type)
	assert.Equal(t, "/mnt/tank/file1.txt", linkEntry.RealPath)
	assert.Equal(t, 0777, linkEntry.Mode)
}

func TestFilesystemClient_ListDir_Empty(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("filesystem.listdir", []DirEntry{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	entries, err := client.Filesystem.ListDir(ctx, "/mnt/tank/empty")
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestFilesystemClient_ListDir_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("filesystem.listdir", 403, "Permission denied")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	entries, err := client.Filesystem.ListDir(ctx, "/mnt/tank/restricted")
	require.Error(t, err)
	assert.Nil(t, entries)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 403, apiErr.Data.Error)
	assert.Equal(t, "Permission denied", apiErr.Message)
}

func TestFilesystemClient_GetACL(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockACL := ACL{
		ACLType: "NFS4",
		UID:     1000,
		GID:     1000,
		ACL: []ACLEntry{
			{
				Tag:   "owner@",
				Type:  "ALLOW",
				Perms: "rwxpDdaARWcCos",
			},
			{
				Tag:   "group@",
				Type:  "ALLOW",
				Perms: "rxaRc",
			},
			{
				Tag:   "everyone@",
				Type:  "ALLOW",
				Perms: "rxaRc",
			},
		},
		Trivial: false,
		Path:    "/mnt/tank/testfile",
	}
	server.SetResponse("filesystem.getacl", mockACL)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	acl, err := client.Filesystem.GetACL(ctx, "/mnt/tank/testfile", false)
	require.NoError(t, err)
	require.NotNil(t, acl)
	assert.Equal(t, "NFS4", acl.ACLType)
	assert.Equal(t, 1000, acl.UID)
	assert.Equal(t, 1000, acl.GID)
	assert.False(t, acl.Trivial)
	assert.Equal(t, "/mnt/tank/testfile", acl.Path)
	assert.Len(t, acl.ACL, 3)
	assert.Equal(t, "owner@", acl.ACL[0].Tag)
	assert.Equal(t, "ALLOW", acl.ACL[0].Type)
	assert.Equal(t, "rwxpDdaARWcCos", acl.ACL[0].Perms)
}

func TestFilesystemClient_GetACL_Simplified(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockACL := ACL{
		ACLType: "POSIX1E",
		UID:     1000,
		GID:     1000,
		ACL: []ACLEntry{
			{
				Tag:   "USER_OBJ",
				Type:  "ALLOW",
				Perms: "rwx",
			},
			{
				Tag:   "GROUP_OBJ",
				Type:  "ALLOW",
				Perms: "r-x",
			},
			{
				Tag:   "OTHER",
				Type:  "ALLOW",
				Perms: "r-x",
			},
		},
		Trivial: true,
		Path:    "/mnt/tank/simplefile",
	}
	server.SetResponse("filesystem.getacl", mockACL)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	acl, err := client.Filesystem.GetACL(ctx, "/mnt/tank/simplefile", true)
	require.NoError(t, err)
	require.NotNil(t, acl)
	assert.Equal(t, "POSIX1E", acl.ACLType)
	assert.True(t, acl.Trivial)
	assert.Len(t, acl.ACL, 3)
}

func TestFilesystemClient_GetACL_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("filesystem.getacl", 404, "Path not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	acl, err := client.Filesystem.GetACL(ctx, "/nonexistent", false)
	require.Error(t, err)
	assert.Nil(t, acl)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "Path not found", apiErr.Message)
}

func TestFilesystemClient_SetACL(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.setacl", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	uid := 1000
	gid := 1000
	req := &SetACLRequest{
		Path: "/mnt/tank/testfile",
		UID:  &uid,
		GID:  &gid,
		DACL: []ACLEntry{
			{
				Tag:   "owner@",
				Type:  "ALLOW",
				Perms: "full_set",
			},
			{
				Tag:   "group@",
				Type:  "ALLOW",
				Perms: "read_set",
			},
			{
				Tag:   "everyone@",
				Type:  "ALLOW",
				Perms: "read_set",
			},
		},
		NFS41Flags: &NFS41Flags{
			Autoinherit: true,
			Protected:   false,
		},
		ACLType: ACLTypeNFS4,
		Options: SetACLOptions{
			StripACL:     false,
			Recursive:    false,
			Traverse:     false,
			Canonicalize: true,
		},
	}

	ctx := NewTestContext(t)
	err := client.Filesystem.SetACL(ctx, req)
	assert.NoError(t, err)
}

func TestFilesystemClient_SetACL_Recursive(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.setacl", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &SetACLRequest{
		Path: "/mnt/tank/testdir",
		DACL: []ACLEntry{
			{
				Tag:   "owner@",
				Type:  "ALLOW",
				Perms: "full_set",
			},
		},
		ACLType: ACLTypeNFS4,
		Options: SetACLOptions{
			StripACL:  false,
			Recursive: true,
			Traverse:  true,
		},
	}

	ctx := NewTestContext(t)
	err := client.Filesystem.SetACL(ctx, req)
	assert.NoError(t, err)
}

func TestFilesystemClient_SetACL_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobError("filesystem.setacl", "Invalid ACL entry")

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &SetACLRequest{
		Path:    "/mnt/tank/testfile",
		DACL:    []ACLEntry{},
		ACLType: ACLTypeNFS4,
		Options: SetACLOptions{},
	}

	ctx := NewTestContext(t)
	err := client.Filesystem.SetACL(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid ACL entry")
}

func TestFilesystemClient_IsACLTrivial(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "trivial ACL",
			path:     "/mnt/tank/simple",
			expected: true,
		},
		{
			name:     "complex ACL",
			path:     "/mnt/tank/complex",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			server.SetResponse("filesystem.acl_is_trivial", tt.expected)

			client := server.CreateTestClient(t)
			defer client.Close()

			ctx := NewTestContext(t)
			result, err := client.Filesystem.IsACLTrivial(ctx, tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilesystemClient_IsACLTrivial_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("filesystem.acl_is_trivial", 404, "Path not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	result, err := client.Filesystem.IsACLTrivial(ctx, "/nonexistent")
	require.Error(t, err)
	assert.False(t, result)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
}

func TestFilesystemClient_GetDefaultACL(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockACL := ACL{
		ACLType: "NFS4",
		UID:     0,
		GID:     0,
		ACL: []ACLEntry{
			{
				Tag:   "owner@",
				Type:  "ALLOW",
				Perms: "full_set",
			},
			{
				Tag:   "group@",
				Type:  "ALLOW",
				Perms: "modify_set",
			},
			{
				Tag:   "everyone@",
				Type:  "ALLOW",
				Perms: "read_set",
			},
		},
		Trivial: false,
	}
	server.SetResponse("filesystem.get_default_acl", mockACL)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	acl, err := client.Filesystem.GetDefaultACL(ctx, DefaultACLTypeOpen, ShareTypeSMB)
	require.NoError(t, err)
	require.NotNil(t, acl)
	assert.Equal(t, "NFS4", acl.ACLType)
	assert.Len(t, acl.ACL, 3)
	assert.Equal(t, "owner@", acl.ACL[0].Tag)
	assert.Equal(t, "full_set", acl.ACL[0].Perms)
}

func TestFilesystemClient_GetDefaultACL_AllTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		aclType   DefaultACLType
		shareType ShareType
	}{
		{
			name:      "Open/None",
			aclType:   DefaultACLTypeOpen,
			shareType: ShareTypeNone,
		},
		{
			name:      "Restricted/AFP",
			aclType:   DefaultACLTypeRestricted,
			shareType: ShareTypeAFP,
		},
		{
			name:      "Home/SMB",
			aclType:   DefaultACLTypeHome,
			shareType: ShareTypeSMB,
		},
		{
			name:      "DomainHome/NFS",
			aclType:   DefaultACLTypeDomainHome,
			shareType: ShareTypeNFS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			mockACL := ACL{
				ACLType: "NFS4",
				ACL:     []ACLEntry{},
			}
			server.SetResponse("filesystem.get_default_acl", mockACL)

			client := server.CreateTestClient(t)
			defer client.Close()

			ctx := NewTestContext(t)
			acl, err := client.Filesystem.GetDefaultACL(ctx, tt.aclType, tt.shareType)
			require.NoError(t, err)
			require.NotNil(t, acl)
			assert.Equal(t, "NFS4", acl.ACLType)
		})
	}
}

func TestFilesystemClient_GetDefaultACL_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("filesystem.get_default_acl", 400, "Invalid ACL type")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	acl, err := client.Filesystem.GetDefaultACL(ctx, DefaultACLType("INVALID"), ShareTypeNone)
	require.Error(t, err)
	assert.Nil(t, acl)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 400, apiErr.Data.Error)
	assert.Equal(t, "Invalid ACL type", apiErr.Message)
}

func TestFilesystemClient_GetDefaultACLChoices(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockChoices := []string{
		"OPEN",
		"RESTRICTED",
		"HOME",
		"DOMAIN_HOME",
	}
	server.SetResponse("filesystem.default_acl_choices", mockChoices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	choices, err := client.Filesystem.GetDefaultACLChoices(ctx)
	require.NoError(t, err)
	assert.Equal(t, mockChoices, choices)
	assert.Len(t, choices, 4)
	assert.Contains(t, choices, "OPEN")
	assert.Contains(t, choices, "RESTRICTED")
	assert.Contains(t, choices, "HOME")
	assert.Contains(t, choices, "DOMAIN_HOME")
}

func TestFilesystemClient_GetDefaultACLChoices_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("filesystem.default_acl_choices", 500, "Internal server error")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	choices, err := client.Filesystem.GetDefaultACLChoices(ctx)
	require.Error(t, err)
	assert.Nil(t, choices)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
}

func TestFilesystemClient_SetPermissions(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.setperm", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	mode := "755"
	uid := 1000
	gid := 1000
	req := &SetPermRequest{
		Path: "/mnt/tank/testfile",
		Mode: &mode,
		UID:  &uid,
		GID:  &gid,
		Options: SetPermOptions{
			StripACL:  true,
			Recursive: false,
			Traverse:  false,
		},
	}

	ctx := NewTestContext(t)
	err := client.Filesystem.SetPermissions(ctx, req)
	assert.NoError(t, err)
}

func TestFilesystemClient_SetPermissions_Recursive(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.setperm", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	mode := "755"
	req := &SetPermRequest{
		Path: "/mnt/tank/testdir",
		Mode: &mode,
		Options: SetPermOptions{
			StripACL:  false,
			Recursive: true,
			Traverse:  true,
		},
	}

	ctx := NewTestContext(t)
	err := client.Filesystem.SetPermissions(ctx, req)
	assert.NoError(t, err)
}

func TestFilesystemClient_SetPermissions_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobError("filesystem.setperm", "Invalid mode")

	client := server.CreateTestClient(t)
	defer client.Close()

	mode := "999"
	req := &SetPermRequest{
		Path: "/mnt/tank/testfile",
		Mode: &mode,
		Options: SetPermOptions{
			StripACL: true,
		},
	}

	ctx := NewTestContext(t)
	err := client.Filesystem.SetPermissions(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid mode")
}

func TestFilesystemClient_ChangeOwner(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.chown", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	uid := 1000
	gid := 1000
	req := &ChownRequest{
		Path: "/mnt/tank/testfile",
		UID:  &uid,
		GID:  &gid,
		Options: ChownOptions{
			Recursive: false,
			Traverse:  false,
		},
	}

	ctx := NewTestContext(t)
	err := client.Filesystem.ChangeOwner(ctx, req)
	assert.NoError(t, err)
}

func TestFilesystemClient_ChangeOwner_UIDOnly(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.chown", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	uid := 1001
	req := &ChownRequest{
		Path: "/mnt/tank/testfile",
		UID:  &uid,
		GID:  nil, // Only change UID
		Options: ChownOptions{
			Recursive: false,
			Traverse:  false,
		},
	}

	ctx := NewTestContext(t)
	err := client.Filesystem.ChangeOwner(ctx, req)
	assert.NoError(t, err)
}

func TestFilesystemClient_ChangeOwner_Recursive(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.chown", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	uid := 1000
	gid := 1000
	req := &ChownRequest{
		Path: "/mnt/tank/testdir",
		UID:  &uid,
		GID:  &gid,
		Options: ChownOptions{
			Recursive: true,
			Traverse:  true,
		},
	}

	ctx := NewTestContext(t)
	err := client.Filesystem.ChangeOwner(ctx, req)
	assert.NoError(t, err)
}

func TestFilesystemClient_ChangeOwner_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobError("filesystem.chown", "User not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	uid := 99999
	req := &ChownRequest{
		Path: "/mnt/tank/testfile",
		UID:  &uid,
		Options: ChownOptions{
			Recursive: false,
		},
	}

	ctx := NewTestContext(t)
	err := client.Filesystem.ChangeOwner(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "User not found")
}

func TestFilesystemClient_GetFile(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.get", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Filesystem.GetFile(ctx, "/mnt/tank/testfile.txt")
	assert.NoError(t, err)
}

func TestFilesystemClient_GetFile_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobError("filesystem.get", "File not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Filesystem.GetFile(ctx, "/nonexistent/file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "File not found")
}

func TestFilesystemClient_PutFile(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.put", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	mode := 0644
	options := &PutFileOptions{
		Append: false,
		Mode:   &mode,
	}

	ctx := NewTestContext(t)
	err := client.Filesystem.PutFile(ctx, "/mnt/tank/newfile.txt", options)
	assert.NoError(t, err)
}

func TestFilesystemClient_PutFile_Append(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.put", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	options := &PutFileOptions{
		Append: true,
		Mode:   nil,
	}

	ctx := NewTestContext(t)
	err := client.Filesystem.PutFile(ctx, "/mnt/tank/appendfile.txt", options)
	assert.NoError(t, err)
}

func TestFilesystemClient_PutFile_NilOptions(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.put", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Filesystem.PutFile(ctx, "/mnt/tank/defaultfile.txt", nil)
	assert.NoError(t, err)
}

func TestFilesystemClient_PutFile_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobError("filesystem.put", "Permission denied")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Filesystem.PutFile(ctx, "/mnt/tank/restricted/file.txt", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Permission denied")
}

func TestFilesystemClient_CreateDefaultACL(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockACL := ACL{
		ACLType: "NFS4",
		ACL: []ACLEntry{
			{
				Tag:   "owner@",
				Type:  "ALLOW",
				Perms: "full_set",
			},
		},
	}
	server.SetResponse("filesystem.get_default_acl", mockACL)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	acl, err := client.Filesystem.CreateDefaultACL(ctx, DefaultACLTypeHome)
	require.NoError(t, err)
	require.NotNil(t, acl)
	assert.Equal(t, "NFS4", acl.ACLType)
	assert.Len(t, acl.ACL, 1)
}

func TestFilesystemClient_CreateShareACL(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockACL := ACL{
		ACLType: "NFS4",
		ACL: []ACLEntry{
			{
				Tag:   "owner@",
				Type:  "ALLOW",
				Perms: "full_set",
			},
		},
	}
	server.SetResponse("filesystem.get_default_acl", mockACL)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	acl, err := client.Filesystem.CreateShareACL(ctx, ShareTypeSMB)
	require.NoError(t, err)
	require.NotNil(t, acl)
	assert.Equal(t, "NFS4", acl.ACLType)
	assert.Len(t, acl.ACL, 1)
}

func TestFilesystemClient_SetSimplePermissions(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.setperm", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Filesystem.SetSimplePermissions(ctx, "/mnt/tank/testfile", "755", false)
	assert.NoError(t, err)
}

func TestFilesystemClient_SetSimplePermissions_Recursive(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.setperm", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Filesystem.SetSimplePermissions(ctx, "/mnt/tank/testdir", "644", true)
	assert.NoError(t, err)
}

func TestFilesystemClient_SetOwnership(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.chown", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	uid := 1000
	gid := 1000

	ctx := NewTestContext(t)
	err := client.Filesystem.SetOwnership(ctx, "/mnt/tank/testfile", &uid, &gid, false)
	assert.NoError(t, err)
}

func TestFilesystemClient_SetOwnership_UIDOnly(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.chown", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	uid := 1001

	ctx := NewTestContext(t)
	err := client.Filesystem.SetOwnership(ctx, "/mnt/tank/testfile", &uid, nil, false)
	assert.NoError(t, err)
}

func TestFilesystemClient_SetOwnership_GIDOnly(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.chown", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	gid := 1001

	ctx := NewTestContext(t)
	err := client.Filesystem.SetOwnership(ctx, "/mnt/tank/testfile", nil, &gid, true)
	assert.NoError(t, err)
}

func TestFilesystemClient_SetOwnership_Neither(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.chown", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Filesystem.SetOwnership(ctx, "/mnt/tank/testfile", nil, nil, false)
	assert.NoError(t, err)
}

// Test ACL types and constants
func TestACLTypeConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ACLType("NFS4"), ACLTypeNFS4)
	assert.Equal(t, ACLType("POSIX1E"), ACLTypePOSIX1E)
	assert.Equal(t, ACLType("RICH"), ACLTypeRICH)
}

func TestDefaultACLTypeConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, DefaultACLType("OPEN"), DefaultACLTypeOpen)
	assert.Equal(t, DefaultACLType("RESTRICTED"), DefaultACLTypeRestricted)
	assert.Equal(t, DefaultACLType("HOME"), DefaultACLTypeHome)
	assert.Equal(t, DefaultACLType("DOMAIN_HOME"), DefaultACLTypeDomainHome)
}

func TestShareTypeConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ShareType("NONE"), ShareTypeNone)
	assert.Equal(t, ShareType("AFP"), ShareTypeAFP)
	assert.Equal(t, ShareType("SMB"), ShareTypeSMB)
	assert.Equal(t, ShareType("NFS"), ShareTypeNFS)
}

// Test complex scenarios and edge cases
func TestFilesystemClient_ComplexACLEntry(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	id := 1000
	mockACL := ACL{
		ACLType: "NFS4",
		ACL: []ACLEntry{
			{
				Tag:   "USER",
				ID:    &id,
				Type:  "ALLOW",
				Perms: map[string]any{"BASIC": "FULL_CONTROL"},
				Flags: map[string]any{"INHERIT": "INHERIT_ONLY"},
				Who:   "testuser",
			},
		},
	}
	server.SetResponse("filesystem.getacl", mockACL)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	acl, err := client.Filesystem.GetACL(ctx, "/mnt/tank/complex", false)
	require.NoError(t, err)
	require.NotNil(t, acl)
	require.Len(t, acl.ACL, 1)

	entry := acl.ACL[0]
	assert.Equal(t, "USER", entry.Tag)
	assert.Equal(t, &id, entry.ID)
	assert.Equal(t, "ALLOW", entry.Type)
	assert.Equal(t, "testuser", entry.Who)
	assert.NotNil(t, entry.Perms)
	assert.NotNil(t, entry.Flags)
}

func TestFilesystemClient_NFS41Flags(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("filesystem.setacl", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &SetACLRequest{
		Path: "/mnt/tank/testfile",
		DACL: []ACLEntry{
			{
				Tag:   "owner@",
				Type:  "ALLOW",
				Perms: "full_set",
			},
		},
		NFS41Flags: &NFS41Flags{
			Autoinherit: true,
			Protected:   true,
		},
		ACLType: ACLTypeNFS4,
		Options: SetACLOptions{
			Canonicalize: true,
		},
	}

	ctx := NewTestContext(t)
	err := client.Filesystem.SetACL(ctx, req)
	assert.NoError(t, err)
}

// Test error handling for edge cases
func TestFilesystemClient_EmptyPath(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("filesystem.stat", 400, "Empty path")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	stat, err := client.Filesystem.Stat(ctx, "")
	require.Error(t, err)
	assert.Nil(t, stat)
}

func TestFilesystemClient_SpecialCharactersInPath(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockStat := FilesystemStat{
		Size:     1024,
		Mode:     0644,
		IsFile:   true,
		RealPath: "/mnt/tank/file with spaces & special chars!@#$%^&*()_+.txt",
	}
	server.SetResponse("filesystem.stat", mockStat)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	stat, err := client.Filesystem.Stat(ctx, "/mnt/tank/file with spaces & special chars!@#$%^&*()_+.txt")
	require.NoError(t, err)
	assert.Equal(t, "/mnt/tank/file with spaces & special chars!@#$%^&*()_+.txt", stat.RealPath)
}

func TestFilesystemClient_LargeFileStatistics(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockStatfs := FilesystemStatfs{
		FreeBytes:  1099511627776,  // 1TB
		AvailBytes: 1099511627776,  // 1TB
		TotalBytes: 10995116277760, // 10TB
		TotalFiles: 10000000,       // 10M files
		FreeFiles:  9000000,        // 9M files
		NameMax:    255,
		Fstype:     "zfs",
	}
	server.SetResponse("filesystem.statfs", mockStatfs)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	statfs, err := client.Filesystem.Statfs(ctx, "/mnt/tank")
	require.NoError(t, err)
	require.NotNil(t, statfs)
	assert.Equal(t, int64(1099511627776), statfs.FreeBytes)
	assert.Equal(t, int64(10995116277760), statfs.TotalBytes)
	assert.Equal(t, int64(10000000), statfs.TotalFiles)
}
