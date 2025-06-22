# Genie

A thread-safe in-memory key-value store with automatic backup functionality for Go applications.

## Features

- **Thread-safe**: All operations are safe for concurrent use from multiple goroutines
- **Automatic backups**: Configurable periodic backups to disk
- **Atomic writes**: Backup operations use atomic file writes to prevent corruption
- **Simple API**: Easy-to-use interface for storing and retrieving key-value pairs
- **Error handling**: Comprehensive error reporting for backup operations

## Installation

```bash
go get github.com/windevkay/forge/genie
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/windevkay/forge/genie"
)

func main() {
    // Create a new store
    store, err := genie.NewStore()
    if err != nil {
        log.Fatal(err)
    }
    
    // Set and get values
    store.Set("username", "alice")
    store.Set("config.timeout", "30s")
    
    value, exists := store.Get("username")
    if exists {
        fmt.Printf("Username: %s\n", value)
    }
    
    // Start automatic backups every 5 minutes
    store.StartAutoBackup(5 * time.Minute)
    defer store.StopAutoBackup()
    
    // Monitor backup errors
    go func() {
        for err := range store.AutoBackupErrors() {
            log.Printf("Backup error: %v", err)
        }
    }()
    
    // Manual backup
    if err := store.Backup(); err != nil {
        log.Printf("Manual backup failed: %v", err)
    }
}
```

## API Documentation

### Store Creation

#### `NewStore() (*Store, error)`

Creates and initializes a new Store instance. The store will attempt to load existing data from a backup file located in the user's home directory.

```go
store, err := genie.NewStore()
if err != nil {
    log.Fatal("Failed to create store:", err)
}
```

### Basic Operations

#### `Set(key, value string)`

Stores a key-value pair in the store. Thread-safe.

```go
store.Set("key", "value")
```

#### `Get(key string) (string, bool)`

Retrieves the value associated with the given key. Returns the value and a boolean indicating if the key exists.

```go
value, exists := store.Get("key")
if exists {
    fmt.Printf("Value: %s\n", value)
}
```

### Backup Operations

#### `Backup() error`

Creates a manual backup of the current store data to disk. The operation is atomic.

```go
if err := store.Backup(); err != nil {
    log.Printf("Backup failed: %v", err)
}
```

#### `StartAutoBackup(interval time.Duration)`

Begins automatic periodic backups at the specified interval.

```go
store.StartAutoBackup(10 * time.Minute)
```

#### `StopAutoBackup()`

Stops the automatic backup process. Should be called when finished with the store.

```go
defer store.StopAutoBackup()
```

#### `AutoBackupErrors() <-chan error`

Returns a channel that delivers errors from automatic backup operations.

```go
go func() {
    for err := range store.AutoBackupErrors() {
        log.Printf("Auto backup error: %v", err)
    }
}()
```

## Backup File Location

Backup files are stored in the user's home directory as `.kvstore_backup.json`. The file is automatically cleared after loading to prevent stale data.

## Thread Safety

All operations are thread-safe and can be called concurrently from multiple goroutines. The store uses read-write mutexes to optimize for concurrent reads while ensuring data consistency.

## Error Handling

- Backup operations return errors if disk operations fail
- Automatic backup errors are delivered via the error channel
- The error channel has a buffer size of 10; additional errors are dropped to prevent deadlock

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
