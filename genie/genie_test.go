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

	s.Set("baz", 123)
	val, ok = s.Get("baz")
	require.True(t, ok)
	require.Equal(t, 123, val)

	s.Set("enabled", true)
	val, ok = s.Get("enabled")
	require.True(t, ok)
	require.Equal(t, true, val)

	s.Set("data", map[string]int{"count": 42})
	val, ok = s.Get("data")
	expectedMap := map[string]int{"count": 42}
	require.True(t, ok)
	require.Equal(t, expectedMap, val)
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

	var contents map[string]any
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

	var contents map[string]any
	err = json.Unmarshal(bytes, &contents)
	require.NoError(t, err)
	require.Equal(t, "A", contents["alpha"])
}

func TestStore_BackupRestoreWithMixedTypes(t *testing.T) {
	s, err := NewStore()
	require.NoError(t, err)

	// Set different types of data
	s.Set("string", "hello")
	s.Set("number", 42)
	s.Set("float", 3.14)
	s.Set("boolean", true)
	s.Set("slice", []string{"a", "b", "c"})
	s.Set("map", map[string]any{"nested": "value", "count": 10})

	err = s.Backup()
	require.NoError(t, err)

	// Create a new store to test restoration
	s2, err := NewStore()
	require.NoError(t, err)

	// Test string value
	val, ok := s2.Get("string")
	require.True(t, ok)
	require.Equal(t, "hello", val)

	// Test number value (JSON unmarshals numbers as float64)
	val, ok = s2.Get("number")
	require.True(t, ok)
	require.Equal(t, float64(42), val)

	// Test float value
	val, ok = s2.Get("float")
	require.True(t, ok)
	require.Equal(t, 3.14, val)

	// Test boolean value
	val, ok = s2.Get("boolean")
	require.True(t, ok)
	require.Equal(t, true, val)

	// Test slice value
	val, ok = s2.Get("slice")
	require.True(t, ok)
	// JSON unmarshals []string as []interface{}
	expectedSlice := []interface{}{"a", "b", "c"}
	require.Equal(t, expectedSlice, val)

	// Test map value
	val, ok = s2.Get("map")
	require.True(t, ok)
	// JSON unmarshals map[string]any as map[string]interface{}
	expectedMap := map[string]interface{}{"nested": "value", "count": float64(10)}
	require.Equal(t, expectedMap, val)
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
