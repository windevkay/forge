package genie

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStore_SetGet(t *testing.T) {
	s, err := NewStore()
	require.NoError(t, err)

	s.Set("foo", "bar")
	val, ok := s.Get("foo")
	require.True(t, ok)
	require.Equal(t, "bar", val)
}

func TestStore_BackupAndRestore(t *testing.T) {
	s, err := NewStore()
	require.NoError(t, err)

	s.Set("one", "1")
	s.Set("two", "2")

	err = s.Backup()
	require.NoError(t, err)

	bytes, err := os.ReadFile(getBackupFilePath(t))
	require.NoError(t, err)

	var contents map[string]string
	err = json.Unmarshal(bytes, &contents)
	require.NoError(t, err)
	require.Equal(t, "1", contents["one"])
	require.Equal(t, "2", contents["two"])

	s2, err := NewStore()
	require.NoError(t, err)

	val, ok := s2.Get("one")
	require.True(t, ok)
	require.Equal(t, "1", val)

	bytes, err = os.ReadFile(getBackupFilePath(t))
	require.NoError(t, err)
	require.Equal(t, 0, len(bytes))
}

func TestStore_AutoBackup(t *testing.T) {
	s, err := NewStore()
	require.NoError(t, err)

	s.Set("alpha", "A")
	s.StartAutoBackup(100 * time.Millisecond)

	time.Sleep(300 * time.Millisecond)
	s.StopAutoBackup()

	bytes, err := os.ReadFile(getBackupFilePath(t))
	require.NoError(t, err)

	var contents map[string]string
	err = json.Unmarshal(bytes, &contents)
	require.NoError(t, err)
	require.Equal(t, "A", contents["alpha"])
}

func TestStore_AutoBackup_ErrorHandling(t *testing.T) {
	s, err := NewStore()
	require.NoError(t, err)

	// Set some data
	s.Set("badkey", "badval")
	s.Set("another", "value")

	// Create a second store instead of copying the first one
	s2, err := NewStore()
	require.NoError(t, err)
	s2.Set("key", "val")

	// Modify the path to an invalid location to induce backup errors
	s2.path = "/root/kvstore_illegal_nonexistent.json"

	s2.StartAutoBackup(50 * time.Millisecond)

	// read error channel
	go func() {
		for err := range s2.AutoBackupErrors() {
			t.Logf("Caught error from auto-backup: %v", err)
		}
	}()

	// Let it run long enough to attempt at least one backup
	time.Sleep(200 * time.Millisecond)
	s2.StopAutoBackup()

	// Should not panic or deadlock
}

func getBackupFilePath(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	return filepath.Join(home, ".kvstore_backup.json")
}
