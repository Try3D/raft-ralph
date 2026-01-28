package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
)

// ErrCorrupted indicates that the stored data is corrupted
type ErrCorrupted struct {
	Reason string
}

func (e *ErrCorrupted) Error() string {
	return "storage corrupted: " + e.Reason
}

// ErrIO indicates an I/O error during storage operations
type ErrIO struct {
	Reason string
}

func (e *ErrIO) Error() string {
	return "storage I/O error: " + e.Reason
}

// Storage defines the interface for storing and retrieving Raft persistent state
type Storage interface {
	// SaveHardState saves the current hard state (term and vote) to storage
	SaveHardState(ctx context.Context, term int, votedFor int) error

	// LoadHardState loads the previously saved hard state from storage
	LoadHardState(ctx context.Context) (term int, votedFor int, err error)
}

// HardState represents the persistent state that must survive crashes
type HardState struct {
	Term    int `json:"term"`
	VotedFor int `json:"voted_for"`
	Checksum uint32 `json:"checksum"`  // CRC32 checksum for corruption detection
}

// FileStorage implements the Storage interface using file-based persistence
type FileStorage struct {
	path string
}

// NewFileStorage creates a new FileStorage instance
func NewFileStorage(path string) *FileStorage {
	return &FileStorage{
		path: path,
	}
}

// calculateChecksum calculates a CRC32 checksum for the given data
func (fs *FileStorage) calculateChecksum(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

// SaveHardState saves the current hard state (term and vote) to storage
// It uses atomic writes and fsync to ensure durability
func (fs *FileStorage) SaveHardState(ctx context.Context, term int, votedFor int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create the hard state object
	hs := HardState{
		Term:     term,
		VotedFor: votedFor,
	}

	// Serialize the hard state to JSON
	data, err := json.Marshal(hs)
	if err != nil {
		return fmt.Errorf("failed to marshal hard state: %w", &ErrIO{Reason: err.Error()})
	}

	// Calculate checksum
	checksum := fs.calculateChecksum(data)
	hs.Checksum = checksum

	// Re-marshal with checksum included
	data, err = json.Marshal(hs)
	if err != nil {
		return fmt.Errorf("failed to marshal hard state with checksum: %w", &ErrIO{Reason: err.Error()})
	}

	// Write to a temporary file first (for atomic replacement)
	tempPath := fs.path + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", &ErrIO{Reason: err.Error()})
	}
	defer file.Close()

	// Write the data to the temporary file
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", &ErrIO{Reason: err.Error()})
	}

	// Sync the file to ensure it's written to disk
	err = file.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", &ErrIO{Reason: err.Error()})
	}

	// Close the file before renaming
	err = file.Close()
	if err != nil {
		return fmt.Errorf("failed to close temporary file: %w", &ErrIO{Reason: err.Error()})
	}

	// Atomically rename the temporary file to the actual file
	err = os.Rename(tempPath, fs.path)
	if err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", &ErrIO{Reason: err.Error()})
	}

	// Also sync the directory to ensure the rename is persisted
	dirFile, err := os.Open(filepath.Dir(fs.path))
	if err != nil {
		return fmt.Errorf("failed to open directory: %w", &ErrIO{Reason: err.Error()})
	}
	defer dirFile.Close()

	err = dirFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync directory: %w", &ErrIO{Reason: err.Error()})
	}

	return nil
}

// LoadHardState loads the previously saved hard state from storage
// It validates the checksum to detect corruption
func (fs *FileStorage) LoadHardState(ctx context.Context) (int, int, error) {
	select {
	case <-ctx.Done():
		return 0, 0, ctx.Err()
	default:
	}

	// Read the file
	data, err := os.ReadFile(fs.path)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, return zero values (first startup)
			return 0, -1, nil
		}
		return 0, 0, fmt.Errorf("failed to read file: %w", &ErrIO{Reason: err.Error()})
	}

	// Parse the JSON
	var hs HardState
	err = json.Unmarshal(data, &hs)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to unmarshal hard state: %w", &ErrCorrupted{Reason: err.Error()})
	}

	// Verify the checksum to detect corruption
	// Temporarily remove the checksum field from the data for verification
	tempHS := HardState{
		Term:     hs.Term,
		VotedFor: hs.VotedFor,
		Checksum: 0, // Zero out the checksum for calculation
	}

	tempData, err := json.Marshal(tempHS)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal temp hard state for checksum: %w", &ErrCorrupted{Reason: err.Error()})
	}

	calculatedChecksum := fs.calculateChecksum(tempData)
	if calculatedChecksum != hs.Checksum {
		return 0, 0, fmt.Errorf("checksum mismatch: expected %d, got %d: %w",
			calculatedChecksum, hs.Checksum, &ErrCorrupted{Reason: "checksum validation failed"})
	}

	return hs.Term, hs.VotedFor, nil
}