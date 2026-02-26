package vfs

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"
)

// MaxDiskBytes represents the maximum disk size in bytes (1.44MB).
const MaxDiskBytes = 1474560

// validFilename is the regex for sanitizing filenames.
var validFilename = regexp.MustCompile(`^\.?[a-zA-Z0-9_]{1,12}(\.[a-zA-Z0-9]{1,3})?$`)

var (
	ErrFileNotFound    = errors.New("file not found")
	ErrInvalidFilename = errors.New("invalid filename")
	ErrQuotaExceeded   = errors.New("disk quota exceeded")
)

type FileEntry struct {
	Data     []byte
	Created  time.Time
	Modified time.Time
}

// VirtualDisk represents an in-memory virtual file system.
type VirtualDisk struct {
	Mu         sync.RWMutex
	Files      map[string]*FileEntry
	DirtyFiles map[string]bool
	UsedBytes  int
	Dirty      bool
}

// NewVirtualDisk creates a new instance of VirtualDisk.
func NewVirtualDisk() *VirtualDisk {
	return &VirtualDisk{
		Files:      make(map[string]*FileEntry),
		DirtyFiles: make(map[string]bool),
		UsedBytes:  0,
	}
}

// Write writes data to a file on the virtual disk.
// It validates the filename, checks for disk quota, and deep copies the data.
// If the file already exists, it is overwritten, and the quota usage is updated accordingly.
func (vd *VirtualDisk) Write(filename string, data []byte) error {
	vd.Mu.Lock()
	defer vd.Mu.Unlock()

	if !validFilename.MatchString(filename) {
		return ErrInvalidFilename
	}

	oldSize := 0
	var entry *FileEntry
	if existing, ok := vd.Files[filename]; ok {
		oldSize = len(existing.Data)
		entry = existing
	}

	newSize := len(data)
	if vd.UsedBytes-oldSize+newSize > MaxDiskBytes {
		return ErrQuotaExceeded
	}

	// Deep copy data to prevent external mutations
	newData := make([]byte, newSize)
	copy(newData, data)

	if entry == nil {
		entry = &FileEntry{
			Created: time.Now(),
		}
		vd.Files[filename] = entry
	}
	entry.Data = newData
	entry.Modified = time.Now()

	vd.DirtyFiles[filename] = true
	vd.UsedBytes = vd.UsedBytes - oldSize + newSize
	vd.Dirty = true

	return nil
}

// Read reads data from a file on the virtual disk.
// It returns the file data if it exists, or an error if the file is not found or the filename is invalid.
func (vd *VirtualDisk) Read(filename string) ([]byte, error) {
	vd.Mu.RLock()
	defer vd.Mu.RUnlock()

	if !validFilename.MatchString(filename) {
		return nil, ErrInvalidFilename
	}

	entry, ok := vd.Files[filename]
	if !ok {
		return nil, ErrFileNotFound
	}

	return entry.Data, nil
}

// Size returns the size of a file in bytes.
// It returns an error if the file is not found or the filename is invalid.
func (vd *VirtualDisk) Size(filename string) (int, error) {
	vd.Mu.RLock()
	defer vd.Mu.RUnlock()

	if !validFilename.MatchString(filename) {
		return 0, ErrInvalidFilename
	}

	entry, ok := vd.Files[filename]
	if !ok {
		return 0, ErrFileNotFound
	}

	return len(entry.Data), nil
}

// Delete removes a file from the virtual disk.
func (vd *VirtualDisk) Delete(filename string) error {
	vd.Mu.Lock()
	defer vd.Mu.Unlock()

	if !validFilename.MatchString(filename) {
		return ErrInvalidFilename
	}

	entry, ok := vd.Files[filename]
	if !ok {
		return ErrFileNotFound
	}

	vd.UsedBytes -= len(entry.Data)
	delete(vd.Files, filename)

	// Mark as dirty so it gets removed from persistence too (if it was persisted)
	vd.DirtyFiles[filename] = true
	vd.Dirty = true

	return nil
}

// FreeSpace returns the number of free bytes on the disk.
func (vd *VirtualDisk) FreeSpace() int {
	vd.Mu.RLock()
	defer vd.Mu.RUnlock()
	return MaxDiskBytes - vd.UsedBytes
}

// List returns a sorted list of all filenames in the VFS.
func (vd *VirtualDisk) List() []string {
	vd.Mu.RLock()
	defer vd.Mu.RUnlock()

	keys := make([]string, 0, len(vd.Files))
	for k := range vd.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetMeta returns the creation and modification time of a file.
func (vd *VirtualDisk) GetMeta(filename string) (time.Time, time.Time, error) {
	vd.Mu.RLock()
	defer vd.Mu.RUnlock()

	if !validFilename.MatchString(filename) {
		return time.Time{}, time.Time{}, ErrInvalidFilename
	}

	entry, ok := vd.Files[filename]
	if !ok {
		return time.Time{}, time.Time{}, ErrFileNotFound
	}

	return entry.Created, entry.Modified, nil
}

// LoadFrom populates the VirtualDisk from binary files in the given host directory.
// Files with invalid VFS names are skipped silently.
// Returns nil if the directory does not exist (first run).
func (vd *VirtualDisk) LoadFrom(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	vd.Mu.Lock()
	defer vd.Mu.Unlock()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !validFilename.MatchString(name) {
			continue
		}

		fullPath := filepath.Join(path, name)
		raw, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		info, err := os.Stat(fullPath)

		fileEntry := &FileEntry{
			Data:     raw,
			Modified: time.Now(),
			Created:  time.Now(),
		}

		if info != nil && err == nil {
			fileEntry.Modified = info.ModTime()
			fileEntry.Created = info.ModTime()
		}

		vd.Files[name] = fileEntry
		vd.UsedBytes += len(raw)
	}

	return nil
}

// PersistTo writes all dirty files in the VirtualDisk to the given host directory.
// The directory is created if it does not exist.
// Returns the first write error encountered.
func (vd *VirtualDisk) PersistTo(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	// Snapshot the dirty files under a write lock (to clear flags), then release before doing I/O.
	vd.Mu.Lock()
	snapshot := make(map[string]*FileEntry)
	deletedFiles := make([]string, 0)

	for name := range vd.DirtyFiles {
		if entry, ok := vd.Files[name]; ok {
			// Deep copy the entry data to avoid race conditions
			newData := make([]byte, len(entry.Data))
			copy(newData, entry.Data)
			snapshot[name] = &FileEntry{
				Data:     newData,
				Created:  entry.Created,
				Modified: entry.Modified,
			}
		} else {
			// File was deleted
			deletedFiles = append(deletedFiles, name)
		}
		delete(vd.DirtyFiles, name)
	}
	if len(vd.DirtyFiles) == 0 {
		vd.Dirty = false
	}
	vd.Mu.Unlock()

	var firstErr error

	// Handle Deletions
	for _, name := range deletedFiles {
		err := os.Remove(filepath.Join(path, name))
		if err != nil && !os.IsNotExist(err) {
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	// Handle Writes
	for name, entry := range snapshot {
		if err := os.WriteFile(filepath.Join(path, name), entry.Data, 0644); err != nil {
			// Restore dirty flag on failure
			vd.Mu.Lock()
			vd.DirtyFiles[name] = true
			vd.Dirty = true
			vd.Mu.Unlock()
			if firstErr == nil {
				firstErr = err
			}
		} else {
			// Update file times on host to match VFS times
			_ = os.Chtimes(filepath.Join(path, name), time.Now(), entry.Modified)
		}
	}

	return firstErr
}
