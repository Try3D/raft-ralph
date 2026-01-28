package storage

import (
	"context"
	"hash/crc32"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var ErrCorrupted = fmt.Errorf("storage corrupted")

var ErrIO = fmt.Errorf("I/O error")

type Storage interface {
	SaveVote(ctx context.Context, term, votedFor int) error
	LoadVote(ctx context.Context) (term, votedFor int, err error)
}

type FileStorage struct {
	path string
	mu   sync.Mutex
}

type VoteData struct {
	Term     int `json:"term"`
	VotedFor int `json:"voted_for"`
	Checksum uint32 `json:"checksum"`
}

func NewFileStorage(path string) *FileStorage {
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)
	return &FileStorage{path: path}
}

func (fs *FileStorage) SaveVote(ctx context.Context, term, votedFor int) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	voteData := VoteData{
		Term:     term,
		VotedFor: votedFor,
	}

	data, err := json.Marshal(voteData)
	if err != nil {
		return fmt.Errorf("failed to marshal vote data: %w", err)
	}

	checksum := crc32.ChecksumIEEE(data)
	voteData.Checksum = checksum

	data, err = json.Marshal(voteData)
	if err != nil {
		return fmt.Errorf("failed to marshal vote data with checksum: %w", err)
	}

	tempPath := fs.path + ".tmp"
	file, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = file.Write(data)
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	err = file.Sync()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	err = os.Rename(tempPath, fs.path)
	if err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	dirFile, err := os.Open(filepath.Dir(fs.path))
	if err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}
	defer dirFile.Close()

	err = dirFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync directory: %w", err)
	}

	return nil
}

func (fs *FileStorage) LoadVote(ctx context.Context) (int, int, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := os.ReadFile(fs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, -1, nil
		}
		return 0, 0, fmt.Errorf("failed to read vote file: %w", err)
	}

	var voteData VoteData
	err = json.Unmarshal(data, &voteData)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to unmarshal vote data: %w", err)
	}

	tempData := VoteData{
		Term:     voteData.Term,
		VotedFor: voteData.VotedFor,
	}
	jsonBytes, err := json.Marshal(tempData)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal vote data for checksum verification: %w", err)
	}

	calculatedChecksum := crc32.ChecksumIEEE(jsonBytes)
	if calculatedChecksum != voteData.Checksum {
		return 0, 0, ErrCorrupted
	}

	return voteData.Term, voteData.VotedFor, nil
}