// Package genie provides a thread-safe in-memory key-value store with automatic
// backup functionality. The store persists data to disk and can automatically
// create backups at regular intervals.
//
// The store is designed to be simple and reliable, with atomic write operations
// for backup files to prevent data corruption. All operations are thread-safe
// and can be used concurrently from multiple goroutines.
//
// Example usage:
//
//	store, err := genie.NewStore()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Set and get values
//	store.Set("key1", "value1")
//	value, exists := store.Get("key1")
//
//	// Start automatic backups every 5 minutes
//	store.StartAutoBackup(5 * time.Minute)
//	defer store.StopAutoBackup()
//
//	// Manual backup
//	if err := store.Backup(); err != nil {
//		log.Printf("Backup failed: %v", err)
//	}
package genie

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	backupFilename     = ".kvstore_backup.json"
	errorChannelBuffer = 10
	backupFileMode     = 0600
)

type Store struct {
	mu       sync.RWMutex
	data     map[string]string
	path     string
	ctx      context.Context
	cancel   context.CancelFunc
	backupWg sync.WaitGroup
	autoMode bool
	errChan  chan error
}

// NewStore creates and initializes a new Store instance. The store will attempt
// to load existing data from a backup file located in the user's home directory.
// If no backup file exists, the store starts with an empty dataset.
//
// The backup file is named ".kvstore_backup.json" and is stored in the user's
// home directory. The store automatically clears the backup file after loading
// to prevent stale data from being loaded on subsequent runs.
//
// Returns an error if:
//   - The user's home directory cannot be determined
//   - The backup file exists but cannot be read
//   - The backup file contains invalid JSON data
//
// Example:
//
//	store, err := NewStore()
//	if err != nil {
//		log.Fatal("Failed to create store:", err)
//	}
func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, backupFilename)

	ctx, cancel := context.WithCancel(context.Background())
	s := &Store{
		data:    make(map[string]string),
		path:    path,
		ctx:     ctx,
		cancel:  cancel,
		errChan: make(chan error, errorChannelBuffer), // buffered to avoid blocking
	}

	if err := s.loadFromBackup(); err != nil {
		return nil, err
	}

	return s, nil
}

// Set stores a key-value pair in the store. If the key already exists,
// its value will be overwritten. This operation is thread-safe and can
// be called concurrently from multiple goroutines.
//
// Parameters:
//   - key: The key to store (can be any non-empty string)
//   - value: The value to associate with the key (can be any string, including empty)
//
// Example:
//
//	store.Set("username", "alice")
//	store.Set("config.timeout", "30s")
func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// Get retrieves the value associated with the given key. This operation
// is thread-safe and can be called concurrently from multiple goroutines.
//
// Parameters:
//   - key: The key to look up
//
// Returns:
//   - string: The value associated with the key (empty string if key doesn't exist)
//   - bool: true if the key exists, false otherwise
//
// Example:
//
//	value, exists := store.Get("username")
//	if exists {
//		fmt.Printf("Username: %s\n", value)
//	} else {
//		fmt.Println("Username not found")
//	}
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// Backup creates a persistent backup of the current store data to disk.
// The backup operation is atomic - it writes to a temporary file first,
// then atomically renames it to the target file to prevent corruption
// if the operation is interrupted.
//
// The backup file is stored as JSON in the user's home directory with
// the filename ".kvstore_backup.json". This operation is thread-safe
// and will not block other read operations, but will block other write
// operations during the data serialization phase.
//
// Returns an error if:
//   - The data cannot be serialized to JSON
//   - A temporary file cannot be created
//   - Writing to the temporary file fails
//   - The atomic rename operation fails
//
// Example:
//
//	if err := store.Backup(); err != nil {
//		log.Printf("Failed to backup store: %v", err)
//	}
func (s *Store) Backup() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Serialize current data
	bytes, err := json.Marshal(s.data)
	if err != nil {
		return err
	}

	// Create a temporary file in the same directory as the backup file
	dir := filepath.Dir(s.path)
	tmpFile, err := os.CreateTemp(dir, "kvstore_backup_*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	// Always ensure temp file is cleaned up if something goes wrong
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	// Write serialized data to the temp file
	if _, err := tmpFile.Write(bytes); err != nil {
		return err
	}

	// Ensure data is flushed to disk
	if err := tmpFile.Sync(); err != nil {
		return err
	}

	// Rename temp file to target file atomically
	if err := os.Rename(tmpPath, s.path); err != nil {
		return err
	}

	return nil
}

// StartAutoBackup begins automatic periodic backups of the store data.
// Backups will occur at the specified interval until StopAutoBackup is called.
// If auto backup is already running, this method does nothing.
//
// The backup operation runs in a separate goroutine and will not block the
// calling goroutine. Any errors that occur during automatic backups are sent
// to the error channel that can be accessed via AutoBackupErrors().
//
// If the error channel buffer becomes full, subsequent errors will be dropped
// to prevent deadlock. The error channel has a buffer size of 10.
//
// Parameters:
//   - interval: How frequently to perform backups (e.g., 5*time.Minute)
//
// Note: You should call StopAutoBackup() when you're done with the store
// to properly clean up the background goroutine.
//
// Example:
//
//	// Start backing up every 10 minutes
//	store.StartAutoBackup(10 * time.Minute)
//	defer store.StopAutoBackup() // Clean up when done
//
//	// Monitor for backup errors
//	go func() {
//		for err := range store.AutoBackupErrors() {
//			log.Printf("Backup error: %v", err)
//		}
//	}()
func (s *Store) StartAutoBackup(interval time.Duration) {
	if s.autoMode {
		return // already running
	}
	s.autoMode = true
	s.backupWg.Add(1)
	go func() {
		defer s.backupWg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.Backup(); err != nil {
					select {
					case s.errChan <- err:
					default: // drop error if buffer is full to avoid deadlock
					}
				}
			case <-s.ctx.Done():
				close(s.errChan)
				return
			}
		}
	}()
}

// StopAutoBackup stops the automatic backup process if it's currently running.
// This method will block until the backup goroutine has fully terminated.
// If auto backup is not currently running, this method does nothing.
//
// After calling this method, the error channel returned by AutoBackupErrors()
// will be closed and no more automatic backups will occur.
//
// It's safe to call this method multiple times. You should always call this
// method when you're finished using the store to properly clean up resources.
//
// Example:
//
//	store.StartAutoBackup(5 * time.Minute)
//	// ... use the store ...
//	store.StopAutoBackup() // Clean shutdown
func (s *Store) StopAutoBackup() {
	if !s.autoMode {
		return
	}
	s.cancel()
	s.backupWg.Wait()
	s.autoMode = false
}

// AutoBackupErrors returns a receive-only channel that delivers errors
// encountered during automatic backup operations. The channel has a buffer
// size of 10 to prevent blocking the backup process.
//
// If the buffer becomes full, subsequent errors will be dropped to avoid
// deadlock in the backup goroutine. The channel will be closed when
// StopAutoBackup() is called.
//
// This channel is only relevant when automatic backups are enabled via
// StartAutoBackup(). If automatic backups are not running, this channel
// will not receive any errors.
//
// Returns:
//   - <-chan error: A receive-only channel for backup errors
//
// Example:
//
//	store.StartAutoBackup(5 * time.Minute)
//
//	// Monitor for backup errors in a separate goroutine
//	go func() {
//		for err := range store.AutoBackupErrors() {
//			log.Printf("Auto backup failed: %v", err)
//			// Optionally trigger manual backup or other recovery
//		}
//		log.Println("Auto backup error monitoring stopped")
//	}()
func (s *Store) AutoBackupErrors() <-chan error {
	return s.errChan
}

func (s *Store) loadFromBackup() error {
	if _, err := os.Stat(s.path); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	bytes, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	if len(bytes) == 0 {
		return nil
	}

	if err := json.Unmarshal(bytes, &s.data); err != nil {
		return err
	}

	return os.WriteFile(s.path, []byte{}, backupFileMode)
}
